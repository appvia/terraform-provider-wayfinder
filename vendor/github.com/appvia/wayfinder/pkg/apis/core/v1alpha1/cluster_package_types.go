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

type ClusterPackage struct {
	// Name is the name of the package version
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name   string             `json:"name"`
	Status CommonStatus       `json:"status,omitempty"`
	Spec   ClusterPackageSpec `json:"spec"`
}

type ClusterPackageSpec struct {
	// PackageName is the name of the package the version satisfies
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	PackageName string `json:"packageName"`
	// Description is the description of the package
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Description string `json:"description,omitempty"`
	// Version is the semantic version of the package version
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Version string `json:"version"`
	// ChartVersion is the version of the underlying helm chart
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	ChartVersion string `json:"chartVersion"`
}

type ClusterPackageCapability struct {
	// Exposed defines whether the package is exposed as a capability
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	Exposed bool `json:"exposed"`
	// Enabled states if capability is enabled on the cluster
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Enabled bool `json:"enabled"`
	// ReadOnly states if the capability can/cannot be enabled
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	ReadOnly bool `json:"readOnly"`
}

// ClusterPackagesList is a resource containing a list of ClusterPackage objects.
type ClusterPackagesList struct {
	// Items is the list of ClusterPackages.
	Items []ClusterPackage `json:"items"`
}
