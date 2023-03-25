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

// PlanObject is the interface that all plan compatible objects must implement
// +kubebuilder:object:generate=false
type PlanObject interface {
	Object
	// GetPolicies Will get the policies for the object
	GetPolicies() []*PlanPolicy
}

// PlanSpec defines the desired state of Plan
// +k8s:openapi-gen=true
type PlanSpec struct {
	// Allocation defines one or more workspaces which are permitted to access
	// this plan
	// +kubebuilder:validation:Optional
	Allocation ResourceAllocation `json:"allocation,omitempty"`
	// Labels is a collection of labels for this plan
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`
	// Policies are a collection of policies related to the use of the plan
	// +kubebuilder:validation:Optional
	Policies []*PlanPolicy `json:"policies,omitempty"`
}

// PlanPolicy defines possible entries for a spec
type PlanPolicy struct {
	// Editable indicates the entry can or cannot be changed
	// +kubebuilder:validation:Required
	Editable *bool `json:"editable,omitempty"`
	// Enum is a collection of possible values
	// +kubebuilder:validation:Optional
	Enum []string `json:"enum,omitempty"`
	// Max is a max to the value
	// +kubebuilder:validation:Optional
	Max *int64 `json:"max,omitempty"`
	// Min is a minimum to the value
	// +kubebuilder:validation:Optional
	Min *int64 `json:"min,omitempty"`
	// Path is the a json path to the value
	// +kubebuilder:validation:Required
	Path string `json:"path,omitempty"`
	// Pattern is used as regex constraint on the input
	// +kubebuilder:validation:Optional
	Pattern string `json:"pattern,omitempty"`
	// Summary provides an optional description to the field attribute
	// +kubebuilder:validation:Optional
	Summary string `json:"summary,omitempty"`
}
