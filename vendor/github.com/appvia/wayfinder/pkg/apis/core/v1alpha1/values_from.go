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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Value is a default value parameter
type Value struct {
	// Path is the path into the helm values
	// +kubebuilder:validation:Required
	Path string `json:"path"`
	// SpecPath is the path into the release spec.
	// +kubebuilder:validation:Optional
	SpecPath *bool `json:"specPath,omitempty"`
	// Key is a path into the resource data
	// +kubebuilder:validation:Required
	Key string `json:"key"`
	// ValuePrefix is a prefix which will be prepended to the value patched into the helm values. If
	// specified, this means the value from the key will be coerced to a string
	// +kubebuilder:validation:Optional
	ValuePrefix string `json:"valuePrefix,omitempty"`
	// ValueSuffix is a suffix which will be appended to the value patched into the helm values. If
	// specified, this means the value from the key will be coerced to a string
	// +kubebuilder:validation:Optional
	ValueSuffix string `json:"valueSuffix,omitempty"`
}

// ClusterValuesFrom is used to reference a values from an associated cluster
type ClusterValuesFrom struct {
	Value `json:",inline"`
}

// CloudAccessConfigValuesFrom is used to reference a values from an associated cloud access config
type CloudAccessConfigValuesFrom struct {
	Value `json:",inline"`
}

// DomainValuesFrom is used to reference the default domain attached to the cluster
// We support zones with `direct` mode only. `createChildZone` will create child zone
// with same selector with `direct` mode.
type DomainValuesFrom struct {
	Value `json:",inline"`
	// Optional is a flag to indicate if the value is required.
	// +kubebuilder:validation:Optional
	Optional bool `json:"optional,omitempty"`
	// Limit is a flag to indicate if the value is limited to specific number
	// of values
	// +kubebuilder:validation:Optional
	Limit int64 `json:"limit,omitempty"`
	// AsArray places the matching values(s) from the domains into the value as an
	// array. If false, multiple values will be comma-concatenated into a string.
	// +kubebuilder:validation:Optional
	AsArray bool `json:"asArray,omitempty"`
	// MatchLabels is used to find one or more specific domain resources for
	// inject - be careful unless intended not to match multiple as we will inject
	// the zone names as an array
	MatchLabels metav1.LabelSelector `json:"matchLabels,omitempty"`
}

// SecretValuesFrom is used to define a reference to a secret
type SecretValuesFrom struct {
	v1.SecretReference `json:",inline"`
	Value              `json:",inline"`
}

// ResourceValuesFrom is used to define a reference to a resource
type ResourceValuesFrom struct {
	Ownership `json:",inline"`
	Value     `json:",inline"`
}

// ValuesFrom defines a means to extract a value out of a resource and into the values
// for an arbitrary destination, such as a helm chart or cloud resource plan
type ValuesFrom struct {
	// Cluster is used to extract a piece of data out of the associated
	// cluster resources and inject into the path defined
	// +kubebuilder:validation:Optional
	Cluster *ClusterValuesFrom `json:"cluster,omitempty"`
	// CloudAccessConfig is used to extract a piece of data out of the cloud
	// access config associated with the cluster and inject into the path defined
	// +kubebuilder:validation:Optional
	CloudAccessConfig *CloudAccessConfigValuesFrom `json:"cloudAccessConfig,omitempty"`
	// Resource is used to filter on and extract the details from one of more
	// managed resources in Wayfinder.
	// +kubebuilder:validation:Optional
	Resource *ResourceValuesFrom `json:"resource,omitempty"`
	// Secret is used to reference a secret in wayfinder
	// +kubebuilder:validation:Optional
	Secret *SecretValuesFrom `json:"secret,omitempty"`
	// Domain is used to reference a domains in wayfinder
	// +kubebuilder:validation:Optional
	Domain *DomainValuesFrom `json:"domain,omitempty"`
}
