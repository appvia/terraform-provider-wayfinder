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

package v2beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// NetworkFabricPlnaKind is the kind for a NetworkFabricPlan
const NetworkFabricPlanKind = "NetworkFabricPlan"

// NetworkFabricPlanGVK is the GVK for a NetworkFabricPlan
var NetworkFabricPlanGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    NetworkFabricPlanKind,
}

// NetworkFabricPlanSpec defines the desired state of Plan
// +k8s:openapi-gen=true
type NetworkFabricPlanSpec struct {
	corev1alpha1.PlanSpec `json:",inline"`
	// Summary provides a short title summary for the network plan
	// It should describe a network in terms that are meaningful for developers:
	// e.g. highest availability network expensive but resilient
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Summary string `json:"summary"`
	// Template are the key+value pairs describing a network configuration
	// +kubebuilder:validation:Required
	Template NetworkFabricSpec `json:"template"`
}

// NetworkFabricPlanStatus defines the observed state of Plan
// +k8s:openapi-gen=true
type NetworkFabricPlanStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// +kubebuilder:webhook:name=networkfabricplans.networking.appvia.io,mutating=false,path=/validate/networking.appvia.io/networkfabricplans,verbs=create;update;delete,groups="networking.appvia.io",resources=networkfabricplans,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=networkfabricplans.networking.appvia.io,mutating=true,path=/mutate/networking.appvia.io/networkfabricplans,verbs=create;update,groups="networking.appvia.io",resources=networkfabricplans,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkFabricPlan is the Schema for the plans API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=networkfabricplans,scope=Cluster,categories={wayfinder}
type NetworkFabricPlan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkFabricPlanSpec   `json:"spec,omitempty"`
	Status NetworkFabricPlanStatus `json:"status,omitempty"`
}

func (NetworkFabricPlan) APIPath() string {
	return "networkfabricplans"
}
func (p *NetworkFabricPlan) Clone() corev1alpha1.Object {
	return p.DeepCopy()
}
func (p *NetworkFabricPlan) CloneInto(o corev1alpha1.Object) {
	p.DeepCopyInto(o.(*NetworkFabricPlan))
}
func (NetworkFabricPlan) ListType() corev1alpha1.ObjectList {
	return &NetworkFabricPlanList{}
}

func (p *NetworkFabricPlan) GetCommonStatus() *corev1alpha1.CommonStatus {
	p.Status.SetStatusIfNotSet(corev1alpha1.SuccessStatus)
	return &p.Status.CommonStatus
}

// GetPolicies implements PlanObject
func (p *NetworkFabricPlan) GetPolicies() []*corev1alpha1.PlanPolicy {
	return p.Spec.PlanSpec.Policies
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkFabricPlanList contains a list of NetworkFabricPlans
type NetworkFabricPlanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkFabricPlan `json:"items"`
}

func (NetworkFabricPlanList) ObjectType() corev1alpha1.Object {
	return &NetworkFabricPlan{}
}
func (l *NetworkFabricPlanList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *NetworkFabricPlanList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*NetworkFabricPlanList))
}

// AllocatedToWorkspace implements the Allocatable interface for Plan, it will return true if the
// specified workspace is allowed to use this plan according to the allocation settings on the
// spec.
func (p *NetworkFabricPlan) AllocatedToWorkspace(ws corev1alpha1.WorkspaceKey) bool {
	return p.Spec.Allocation.Allowed(p.Namespace, ws)
}

func (l *NetworkFabricPlanList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *NetworkFabricPlanList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []NetworkFabricPlan{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*NetworkFabricPlan)))
	}
}
