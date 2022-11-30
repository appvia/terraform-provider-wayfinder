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

// ResourceSelector is a resource selector
type ResourceSelector struct {
	// NonResourceURLs are urls which do not map to resources by require
	// some level of policy control
	NonResourceURLs []string `json:"nonResourceURLs,omitempty"`
	// Groups is a collection of api grouprs to filter on
	// +kubebuilder:validation:Optional
	Groups []string `json:"groups,omitempty"`
	// Resources is a collection of resources under those groups
	// +kubebuilder:validation:Optional
	Resources []string `json:"resources,omitempty"`
	// SubResources is a collection of subresource under the resource type
	// Deprecated field please use resource/subresource format
	// +kubebuilder:validation:Optional
	SubResources []string `json:"subresources,omitempty"`
	// ResourceNames is a collection of resource names
	// +kubebuilder:validation:Optional
	ResourceNames []string `json:"resourceNames,omitempty"`
	// Labels a collection of labels to filter the resource by
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`
	// Verbs are actions on the resources themselves
	// +kubebuilder:validation:Optional
	Verbs []string `json:"verbs,omitempty"`
}

// ActionSelector is used to filter on the operation type
type ActionSelector struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Verbs []string `json:"verbs"`
}

// SubjectSelector is used to filter down in the caller
type SubjectSelector struct {
	// Subjects is a collection of subjects / username to filter on
	// +kubebuilder:validation:Optional
	Subjects []string `json:"subjects,omitempty"`
	// Roles is a collection of roles the user has access to
	// +kubebuilder:validation:Optional
	Roles []string `json:"roles,omitempty"`
	// Groups is a collection of groups the user is a member of
	// +kubebuilder:validation:Optional
	Groups []string `json:"groups,omitempty"`
	// Scopes is a collection of scopes for the identity
	// +kubebuilder:validation:Optional
	Scopes []string `json:"scopes,omitempty"`
}
