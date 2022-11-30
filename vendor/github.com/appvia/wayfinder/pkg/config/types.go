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
	"strings"

	"github.com/appvia/wayfinder/pkg/common"
)

// ReleaseConfig is a wayfinder release configuration
type ReleaseConfig struct {
	// LastUpdate defines the last time the release config was updated via automation
	LastUpdate string `json:"lastUpdate,omitempty" yaml:"lastUpdate,omitempty"`
	// Charts defines all possible versions to be used by particular wayfinder release
	Charts K8SProvidersVersionConfig `json:"providersConfig,omitempty" yaml:"providersConfig,omitempty"`
	// GlobalCharts defines all possible versions to be used by all wayfinder releases
	GlobalCharts ChartList `json:"globalCharts,omitempty" yaml:"globalCharts,omitempty"`
}

// K8SProvidersVersionConfig is a particular k8s version configuration
// Example: 1.22.5: ...
type K8SProvidersVersionConfig map[string]ProvidersVersionConfig

// ProvidersConfig is all providers configurations
type ProvidersVersionConfig struct {
	// AKS supported versions
	AKS ProviderVersionConfig `json:"aks,omitempty" yaml:"aks,omitempty"`
	// EKS supported versions
	EKS ProviderVersionConfig `json:"eks,omitempty" yaml:"eks,omitempty"`
	// GKE supported versions
	GKE ProviderVersionConfig `json:"gke,omitempty" yaml:"gke,omitempty"`
}

// GetProvider returns the provider version config for the specified provider
func (p ProvidersVersionConfig) GetProvider(provider string) (ProviderVersionConfig, error) {
	switch strings.ToLower(provider) {
	case strings.ToLower(common.ClusterProviderAzure):
		return p.AKS, nil
	case strings.ToLower(common.ClusterProviderGCP):
		return p.GKE, nil
	case strings.ToLower(common.ClusterProviderAWS):
		return p.EKS, nil
	default:
		return ProviderVersionConfig{}, fmt.Errorf("provider %s not supported", provider)
	}
}

type ChartList map[string]ChartConfig

// Populated indicates whether the chart list has any charts in it
func (c ChartList) Populated() bool {
	return len(c) > 0
}

type ProviderVersionConfig struct {
	// Charts defines overrides for global chart values. It is used to deviate from global config for individual releases
	Charts ChartList `json:"charts,omitempty" yaml:"charts,omitempty"`
	// Images defines other images used in the release
	Images Images `json:"images,omitempty" yaml:"images,omitempty"`
}

// ChartConfig is an individual chart configuration
type ChartConfig struct {
	Version  string `json:"version,omitempty" yaml:"version,omitempty"`
	External bool   `json:"external,omitempty" yaml:"external,omitempty"`
	Values   []byte `json:"values,omitempty" yaml:"values,omitempty"`
}

// Images defines other images used in the release
type Images struct {
	HelmController   string `json:"helmController,omitempty" yaml:"helmController,omitempty"`
	SourceController string `json:"sourceController,omitempty" yaml:"sourceController,omitempty"`
}

// Populated indicates whether all required images are populated
func (i Images) Populated() bool {
	return i.HelmController != "" && i.SourceController != ""
}
