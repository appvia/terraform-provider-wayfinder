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
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// +kubebuilder:webhook:name=policies.policy.appvia.io,mutating=false,path=/validate/policy.appvia.io/policies,verbs=create;update;delete,groups="policy.appvia.io",resources=policy,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// PolicyKind is the kind for a Policy
const PolicyKind = "Policy"

// PolicyGVK is the GVK for a Policy
var PolicyGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    PolicyKind,
}

// PolicySpec defines the desired state of policy
type PolicySpec struct {
	// Inputs are parameters to the plan templates
	// +kubebuilder:validation:Optional
	Inputs *apiextv1.JSON `json:"inputs,omitempty"`
	// Hints provides a list collection of resources which might be
	// required in the rules engine
	// +kubebuilder:validation:Optional
	Hints []metav1.GroupVersionResource `json:"hints,omitempty"`
	// Policy defines the policy definition itself
	// +kubebuilder:validation:Optional
	Policy *PolicyDecision `json:"policy,omitempty"`
	// PolicyRef is used to refer to an inbuild wayfinder policy rather than
	// defining a inline policy - we find the plan and copy onto the status for
	// reference and implementation
	// +kubebuilder:validation:Optional
	PolicyRef *PolicyReference `json:"policyRef,omitempty"`
	// Selectors is the resource we are filtering on
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:Optional
	Selectors []*Selector `json:"selectors,omitempty"`
	// Target is the essentially the location the policy should be positioned
	// If no target is supplied we assume it's destined to the wayfinder api
	// +kubebuilder:validation:Optional
	Target *Target `json:"target,omitempty"`
}

// PolicyReference is used to reference an inbuilt policy document
type PolicyReference struct {
	// Name is the name of inbult policy we are referring to
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Namespace is the namespace the policy plan exists in
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
}

// PolicyDecision defines the structure of a inline policy
type PolicyDecision struct {
	// RolesDecision indicates a role is provided as an outcome
	RolesDecision *[]string `json:"rolesDecision,omitempty"`
	// Decision is an inline decision on the action
	// +kubebuilder:validation:Optional
	Decision *Decision `json:"decision,omitempty"`
	// Policy contains the inline rego template to apply
	// +kubebuilder:validation:Optional
	Policy *string `json:"policy,omitempty"`
}

// PolicyStatus defines the observed state of status on a policy
// +k8s:openapi-gen=true
type PolicyStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// Plan is a copy of the plans the policy is based on if any
	// +kubebuilder:validation:Optional
	Plan *PolicyPlan `json:"plan,omitempty"`
	// PlanRevision is the revision of the parent plan
	// +kubebuilder:validation:Optional
	PlanRevision *int64 `json:"planRevision,omitempty"`
}

// GetCommonStatus implements CommonStatusAware
func (r *Policy) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &r.Status.CommonStatus
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Policy is the Schema for the policies API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=policies,scope=Namespaced,categories={wayfinder}
type Policy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PolicySpec   `json:"spec,omitempty"`
	Status PolicyStatus `json:"status,omitempty"`
}

func (Policy) APIPath() string {
	return "policies"
}
func (r *Policy) Clone() corev1alpha1.Object {
	return r.DeepCopy()
}
func (r *Policy) CloneInto(o corev1alpha1.Object) {
	r.DeepCopyInto(o.(*Policy))
}
func (Policy) ListType() corev1alpha1.ObjectList {
	return &PolicyList{}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyList contains a list of policies
type PolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Policy `json:"items"`
}

func (PolicyList) ObjectType() corev1alpha1.Object {
	return &Policy{}
}
func (l *PolicyList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *PolicyList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*PolicyList))
}

func (l *PolicyList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *PolicyList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []Policy{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*Policy)))
	}
}
