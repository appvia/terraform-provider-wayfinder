/**
 * Copyright 2021 Appvia Ltd <info@appvia.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
	"sigs.k8s.io/yaml"

	"github.com/appvia/wayfinder/pkg/common"
	semverutil "github.com/appvia/wayfinder/pkg/utils/semver"
	"github.com/appvia/wayfinder/pkg/version"
)

// This is bit confusing. Sorry :(
// We embed release-config.yaml using bindata.go as embed does not allows us to walk path back to root
// When we write file we are using soft link file in data/ directory which is same as release-config.yaml in root.
// Hard links are not supported by GIT and soft links are not supported by embed.

var (
	filePathPattern  = "rconfig/release-%s.yaml"
	fileAssetPattern = "release-%s.yaml"
)

var _ Configurator = &configurator{}

type Configurator interface {
	// GetConfig returns full configuration
	GetConfig(wfVersion string) (*ReleaseConfig, error)

	// GetSupportedK8SVersions returns the list of currently-supported K8S major.minor versions for
	// the given provider
	GetSupportedK8SVersions(provider, wfVersion string) ([]string, error)

	// PersistConfig persists configuration
	PersistConfig(config *ReleaseConfig) error

	// GetVersions will return config for particular provider, wayfinder version and k8s version
	GetVersions(provider, kVersion, wfVersion string) (*ReleaseConfig, error)

	// GetChartVersion will return version for request chart for provider and wayfinder version
	// wfVersion and kversion (k8s version) can be provided as 'latest'
	GetChartVersion(provider, kVersion, wfVersion, chartName string) (string, error)

	// GetImages will return images for particular provider, wayfinder version and k8s version
	GetImages(provider, kVersion, wfVersion string) (*Images, error)

	// GetGlobalCharts will return list of charts that are available for all providers
	GetGlobalCharts() (*ChartList, error)
}

func New() *configurator {
	return &configurator{}
}

type configurator struct {
	releseseConfig *ReleaseConfig
}

func (c configurator) GetConfig(wfversion string) (*ReleaseConfig, error) {
	return getConfig(wfversion)
}

func (c configurator) GetSupportedK8SVersions(provider string, wfVersion string) ([]string, error) {
	var cf *ReleaseConfig
	if c.releseseConfig != nil {
		cf = c.releseseConfig
	} else {
		var err error
		cf, err = getConfig(wfVersion)
		if err != nil {
			return nil, err
		}
	}

	versions := []string{}
	for v := range cf.Charts {
		pvc, err := cf.Charts[v].GetProvider(provider)
		if err != nil {
			return nil, err
		}
		if pvc.Charts.Populated() && pvc.Images.Populated() {
			versions = append(versions, v)
		}
	}

	sort.Strings(versions)

	return versions, nil
}

var releaseRegexp = regexp.MustCompile(`^(v|x)[0-9]+\.[0-9]+\.[0-9]+($|-[a-z]+[0-9]+)`)

func getConfig(wfversion string) (*ReleaseConfig, error) {
	var config ReleaseConfig

	// if we're running a tagged build, i.e. a version string that matches v1.2.3 (or v1.2.3-blah4)
	// use the specific versioned configuration, else we need to load release-latest.yaml from data/
	// directory
	if releaseRegexp.MatchString(wfversion) {
		var err error
		wfversion, err = parseVersionPatch(wfversion)
		if err != nil {
			return nil, err
		}
	} else {
		wfversion = "latest"
	}

	data, err := Asset(fmt.Sprintf(fileAssetPattern, wfversion))
	if err != nil {
		if os.IsNotExist(err) {
			return &config, nil
		}
		return nil, err
	}

	if err = yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c configurator) PersistConfig(config *ReleaseConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(fmt.Sprintf(filePathPattern, version.Release), data, 0644)
}

func (c configurator) GetVersions(provider, kVersion, wfversion string) (*ReleaseConfig, error) {
	var err error
	if c.releseseConfig == nil {
		c.releseseConfig, err = getConfig(wfversion)
		if err != nil {
			return nil, err
		}
	}

	// shorted version of k8s version
	kVersion, err = parseVersionMinor(kVersion)
	if err != nil {
		return nil, err
	}

	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}

	var resultConfig, tempConfig ReleaseConfig
	if kVersion == "" { // return all versions
		tempConfig = *c.releseseConfig
	} else {
		if val, ok := c.releseseConfig.Charts[kVersion]; !ok {
			return nil, fmt.Errorf("kubernetes version %s not supported", kVersion)
		} else {
			tempConfig.Charts = make(map[string]ProvidersVersionConfig)
			tempConfig.Charts[kVersion] = val
		}
	}

	// if provider is not specified, return all versions
	if provider == "" {
		return &tempConfig, nil
	} else {
		resultConfig.Charts = make(map[string]ProvidersVersionConfig)
		switch strings.ToLower(provider) {
		case strings.ToLower(common.ClusterProviderAzure):
			for kver, chart := range tempConfig.Charts {
				resultConfig.Charts[kver] = ProvidersVersionConfig{
					AKS: chart.AKS,
				}
			}
		case strings.ToLower(common.ClusterProviderGCP):
			for kver, chart := range tempConfig.Charts {
				resultConfig.Charts[kver] = ProvidersVersionConfig{
					GKE: chart.GKE,
				}
			}
		case strings.ToLower(common.ClusterProviderAWS):
			for kver, chart := range tempConfig.Charts {
				resultConfig.Charts[kver] = ProvidersVersionConfig{
					EKS: chart.EKS,
				}
			}
		default:
			return nil, fmt.Errorf("provider %s not supported", provider)
		}

	}
	return &resultConfig, nil
}

func (c configurator) GetChartVersion(provider, kVersion, wfversion, chartName string) (string, error) {
	versions, err := c.GetVersions(provider, kVersion, wfversion)
	if err != nil {
		return "", err
	}

	if kVersion == "latest" {
		kVersion, err = getLatestVersion(versions)
		if err != nil {
			return "", err
		}
	} else {
		kVersion, err = parseVersionMinor(kVersion)
		if err != nil {
			return "", err
		}
	}

	errorF := "chart %s not found for provider %s and kVersion %s"

	switch strings.ToLower(provider) {
	case strings.ToLower(common.ClusterProviderAzure):
		if val, ok := versions.Charts[kVersion].AKS.Charts[chartName]; !ok {
			return "", fmt.Errorf(errorF, chartName, provider, kVersion)
		} else {
			return val.Version, nil
		}
	case strings.ToLower(common.ClusterProviderGCP):
		if val, ok := versions.Charts[kVersion].GKE.Charts[chartName]; !ok {
			return "", fmt.Errorf(errorF, chartName, provider, kVersion)
		} else {
			return val.Version, nil
		}
	case strings.ToLower(common.ClusterProviderAWS):
		if val, ok := versions.Charts[kVersion].EKS.Charts[chartName]; !ok {
			return "", fmt.Errorf(errorF, chartName, provider, kVersion)
		} else {
			return val.Version, nil
		}
	}

	return "", fmt.Errorf(errorF, chartName, provider, kVersion)
}

func (c configurator) GetGlobalCharts() (*ChartList, error) {
	var err error
	c.releseseConfig, err = getConfig(version.Release)
	if err != nil {
		return nil, err
	}

	return &c.releseseConfig.GlobalCharts, nil
}

func (c configurator) GetImages(provider, kVersion, wfVersion string) (*Images, error) {
	versions, err := c.GetVersions(provider, kVersion, wfVersion)
	if err != nil {
		return nil, err
	}

	if kVersion == "latest" {
		kVersion, err = getLatestVersion(versions)
		if err != nil {
			return nil, err
		}
	} else {
		kVersion, err = parseVersionMinor(kVersion)
		if err != nil {
			return nil, err
		}
	}

	switch strings.ToLower(provider) {
	case strings.ToLower(common.ClusterProviderAzure):
		img := versions.Charts[kVersion].AKS.Images
		return &img, nil
	case strings.ToLower(common.ClusterProviderGCP):
		img := versions.Charts[kVersion].GKE.Images
		return &img, nil
	case strings.ToLower(common.ClusterProviderAWS):
		img := versions.Charts[kVersion].EKS.Images
		return &img, nil
	}

	return nil, fmt.Errorf("images not found for provider %s and kVersion %s ", provider, kVersion)
}

func getLatestVersion(versions *ReleaseConfig) (string, error) {
	latest := "v0.0.0"
	result := "v0.0.0"
	for kversion := range versions.Charts {
		var v string
		if !strings.HasPrefix(kversion, "v") {
			v = "v" + kversion
		} else {
			v = kversion
		}

		if semver.Compare(v, latest) > 0 {
			latest = v
			result = kversion
		}
	}

	if result == "v0.0.0" {
		return "", fmt.Errorf("latest version not found")
	}

	return result, nil
}

func parseVersionPatch(v string) (string, error) {
	sv, err := semverutil.NewVersion(v)
	if err != nil {
		return "", fmt.Errorf("invalid version %s", v)
	}
	return fmt.Sprintf("v%d.%d.%d", sv.Major(), sv.Minor(), sv.Patch()), nil
}

func parseVersionMinor(v string) (string, error) {
	sv, err := semverutil.NewVersion(v)
	if err != nil {
		return "", fmt.Errorf("invalid version %s", v)
	}
	return fmt.Sprintf("v%d.%d", sv.Major(), sv.Minor()), nil
}
