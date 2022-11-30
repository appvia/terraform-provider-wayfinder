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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// PolicyPlanSpec defines the desired state of policy
type PolicyPlanSpec struct {
	// Description is a summary of what the plan provides
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Description string `json:"description"`
	// Hints provides a list collection of resources which might be
	// required in the rules engine
	// +kubebuilder:validation:Optional
	Hints []metav1.GroupVersionResource `json:"hints,omitempty"`
	// Inputs is a collection of inputs for this policy plan
	// +kubebuilder:validation:Optional
	Inputs []*PolicyInput `json:"inputs,omitempty"`
	// Policy is the actual policy document associated to the plan
	// +kubebuilder:validation:Optional
	Policy *PolicyDecision `json:"policy,omitempty"`
	// Selectors are optional filters which can be used to filter the target
	// Deprecated, use the spec.templates
	// of this policy - essentially it's a list of filters which can be used i.e
	// I can be applied to all Plans or all clusters
	// +kubebuilder:validation:Optional
	Selectors []*Selector `json:"selectors,omitempty"`
	// Templates is a collection of templates used to generate polices on
	// behalf of the subject
	// +kubebuilder:validation:Optional
	Templates []*PolicyTemplate `json:"templates,omitempty"`
	// Target is a target for this policy i.e. wayfinder api or one or more clusters
	// +kubebuilder:validation:Optional
	Target *Target `json:"target,omitempty"`
}

// PolicyTemplate describes a policy template
type PolicyTemplate struct {
	// Name is a descriptive name of the policy template
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// DisableSubjectInjection is used to inform the controller not to inject
	// the subjects associated to the policy
	DisableSubjectInjection *bool `json:"disableSubjectInjection,omitempty"`
	// Policy is the actual policy document associated to the plan
	// +kubebuilder:validation:Optional
	Policy *PolicyDecision `json:"policy,omitempty"`
	// Selectors are optional filters which can be used to filter the target
	// +kubebuilder:validation:Optional
	Selectors []*Selector `json:"selectors,omitempty"`
	// Target is a target for the for the policy - i.e the cluster or clusters the
	// policy should be deployed. Left blank the policy is assumed to apply to the Wayfinder
	// API server itself.
	// +kubebuilder:validation:Optional
	Target *Target `json:"target,omitempty"`
	// Template is the template used to generate the the policy
	// +kubebuilder:validation:Optional
	Template *string `json:"template,omitempty"`
}

// InputType indicates the values
type InputType string

const (
	// ArrayInputType indicates many values
	ArrayInputType = "many"
	// SingleInputType indicates a single value
	SingleInputType = "single"
)

// PolicyInput describes the input required for a policy plan
type PolicyInput struct {
	// APIVersion is the api group the resource input comes from
	// +kubebuilder:validation:Optional
	APIVersion string `json:"apiVersion,omitempty"`
	// Description provides a descriptive reason for why the input is required and
	// how it's related to the policy
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Description string `json:"description"`
	// Enum is a collection of possible values
	// +kubebuilder:validation:Optional
	Enum []string `json:"enum,omitempty"`
	// Format indicates the format for of the input
	// +kubebuilder:validation:Optional
	Format string `json:"format,omitempty"`
	// Name is the name of the input which is injected when templating out the policies
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Required indicates the input is a required parameter
	// +kubebuilder:validation:Optional
	Required bool `json:"required,omitempty"`
	// Resource is the resource inside the group the we need as an input
	// +kubebuilder:validation:Optional
	Resource string `json:"resource,omitempty"`
	// Type indicates the type of value
	// +kubebuilder:validation:Optional
	Type InputType `json:"type,omitempty"`
}

// PolicyPlanStatus defines the observed state of status on a policy
// +k8s:openapi-gen=true
type PolicyPlanStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyPlan is the Schema for the policies API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=policyplans,scope=Namespaced,categories={wayfinder}
type PolicyPlan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PolicyPlanSpec   `json:"spec,omitempty"`
	Status PolicyPlanStatus `json:"status,omitempty"`
}

func (PolicyPlan) APIPath() string {
	return "policyplans"
}
func (r *PolicyPlan) Clone() corev1alpha1.Object {
	return r.DeepCopy()
}
func (r *PolicyPlan) CloneInto(o corev1alpha1.Object) {
	r.DeepCopyInto(o.(*PolicyPlan))
}
func (PolicyPlan) ListType() corev1alpha1.ObjectList {
	return &PolicyPlanList{}
}

// GetCommonStatus implements CommonStatusAware
func (r *PolicyPlan) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &r.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyPlanList contains a list of policies
type PolicyPlanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PolicyPlan `json:"items"`
}

func (PolicyPlanList) ObjectType() corev1alpha1.Object {
	return &PolicyPlan{}
}
func (l *PolicyPlanList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *PolicyPlanList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*PolicyPlanList))
}

func (l *PolicyPlanList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *PolicyPlanList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []PolicyPlan{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*PolicyPlan)))
	}
}
