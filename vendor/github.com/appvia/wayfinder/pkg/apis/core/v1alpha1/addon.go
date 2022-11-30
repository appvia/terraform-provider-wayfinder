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

type ClusterCapability struct {
	// Name is the name of the capability
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name   string                `json:"name"`
	Status CommonStatus          `json:"status,omitempty"`
	Spec   ClusterCapabilitySpec `json:"spec"`
}

// ClusterCapabilitySpec defines the state of the capability on the cluster
type ClusterCapabilitySpec struct {
	// Description is the description of the capability
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Description string `json:"description,omitempty"`
	// Enabled states if capability is enabled
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Enabled bool `json:"enabled"`
	// ReadOnly states if the capability can/cannot be enabled
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	ReadOnly bool `json:"readOnly"`

	// EnableLabel is internal struct to store the label with which addon
	// should be enabled. Not marshaled in API responses
	EnableLabel string `json:"-"`
}

// ClusterCapabilitiesList is a resource containing a list of ClusterCapability objects.
type ClusterCapabilitiesList struct {
	// Items is the list of ClusterCapabilities.
	Items []ClusterCapability `json:"items"`
}
