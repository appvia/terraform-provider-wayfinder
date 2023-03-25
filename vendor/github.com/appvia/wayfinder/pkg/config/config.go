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
	"errors"
	"sync"
)

var (
	confLock      = &sync.RWMutex{}
	currentConfig *Config
)

// Current returns the current platform configuration. This will panic if SetCurrentConfig
// has not been called.
func Current() Config {
	confLock.RLock()
	defer confLock.RUnlock()
	if currentConfig == nil {
		panic("no current config has been set, must call SetCurrent before Current is valid")
	}

	return *currentConfig
}

// SetCurrentConfig sets the current platform configuration. After this is called, CurrentConfig()
// will return the set config.
func SetCurrent(c Config) {
	confLock.Lock()
	defer confLock.Unlock()

	currentConfig = &c
}

// Config is the shared configuration available to controllers, webhooks, API server and init
type Config struct {
	// ChartsImage is the Docker image for the Wayfinder Helm charts
	ChartsImage string `json:"charts-image,omitempty"`
	// AuthProxyImage is the image to use for oidc proxy
	AuthProxyImage string `json:"auth-proxy-image,omitempty"`
	// FluxHelmControllerImage is the image to use for the flux helm controller
	FluxHelmControllerImage string `json:"flux-helm-controller-image,omitempty"`
	// FluxSourceControllerImage is the image to use for the flux source controller
	FluxSourceControllerImage string `json:"flux-source-controller-image,omitempty"`
	// CertificateAuthority is the path to a CA
	CertificateAuthority string `json:"certificate-authority,omitempty"`
	// CertificateAuthorityKey is the path to the private key
	CertificateAuthorityKey string `json:"certificate-authority-key,omitempty"`
	// FeatureGates defines which feature gates should be enabled/disabled
	FeatureGates map[string]bool `json:"feature-gates,omitempty"`
	// Licence key for this installation
	LicenseKey string `json:"license-key"`
	// PublicUIURL is the public url for the Wayfinder UI
	PublicUIURL string `json:"ui-public-url,omitempty"`
	// PublicAPIURL is the public url for the Wayfinder API
	PublicAPIURL string `json:"api-public-url,omitempty"`
	// APICACertificate is the CA certificate use for the internal API url, if TLS is enabled
	// MDH 2022-05-11: Pretty certain this is unused, but scared to remove it
	APICACertificate string `json:"api-ca-certificate"`
	// InstanceIdentifier is a unique identifier for this instance of Wayfinder - leave blank for
	// a new one to be assigned, keep the same for continuity of costs data when rebuilding an
	// instance of Wayfinder.
	InstanceIdentifier string `json:"instance-identifier,omitempty"`
	// KubeNameSpace is the namespace to use when using kubernetes assets e.g. the wayfinder service account. Need to use for workload identity
	KubeNameSpace string `json:"kube-namespace,omitempty"`
	// KognizerCollectorURL is the URL for the Kognizer collector, used to configure the agent on clusters
	KognizerCollectorURL string `json:"kognizer-collector-url,omitempty"`
	// KognizerCollectorToken is the auth token for the Kognizer collector, used to configure the agent on clusters
	KognizerCollectorToken string `json:"kognizer-collector-token,omitempty"`
	// KognizerCollectorImage is the image to deploy for the Kognizer agent
	KognizerCollectorImage string `json:"kognizer-collector-image,omitempty"`
	// Verbose is debug logging for this and deployed services e.g. oidc:proxy
	Verbose bool `json:"verbose,omitempty"`

	// DedicatedCluster is true if this instance of Wayfinder is running in a dedicated management
	// cluster which it owns, instead of pointing at a different Kube API for data storage
	//
	// i.e. this will be false in 'W-in-K' mode and true for a full install.
	DedicatedCluster bool `json:"dedicatedCluster,omitempty"`
}

// IsValid checks the options
func (c Config) IsValid() error {
	if c.CertificateAuthority == "" {
		return errors.New("no certificate authority has been defined")
	}
	if c.CertificateAuthorityKey == "" {
		return errors.New("no certificate authority key")
	}

	return nil
}

// HasCertificateAuthorityKey check if we have a key
func (c Config) HasCertificateAuthorityKey() bool {
	return c.CertificateAuthorityKey != ""
}

// HasCertificateAuthority checks if we have a CA
func (c Config) HasCertificateAuthority() bool {
	return c.CertificateAuthority != ""
}

// IsFeatureGateEnabled checks if the gate is enabled in the API
func (c Config) IsFeatureGateEnabled(gate string) bool {
	if c.FeatureGates == nil {
		return false
	}

	return c.FeatureGates[gate]
}

// UIConfigMap is the name of a config map that stores the UI config, the content of which is
// controlled by the UI
const UIConfigMap = "portal-config"
