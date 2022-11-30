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

// ClusterPlanKind is the kind for a ClusterPlan
const ClusterPlanKind = "ClusterPlan"

// ClusterPlanGVK is the GVK for a ClusterPlan
var ClusterPlanGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    ClusterPlanKind,
}

// ClusterPlanSpec defines the desired state of Plan
// +k8s:openapi-gen=true
type ClusterPlanSpec struct {
	corev1alpha1.PlanSpec `json:",inline"`
	// Summary provides a short title summary for the cluster plan
	// It should describe a cluster in terms that are meaningful for developers:
	// e.g. a cluster with very large nodes suitable for memory intensive applications
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Summary string `json:"summary"`
	// DefaultNetworkFabricPlan is the default network fabric plan to use for this cluster
	// this should indicate a networkFabricPlan that is compatible with this cluster plan
	// Could extend this to be an array if we want to drive
	DefaultNetworkFabricPlan string `json:"defaultNetworkFabricPlan,omitempty"`
	// Template are the key+value pairs describing a cluster configuration
	// +kubebuilder:validation:Required
	Template ClusterSpec `json:"template"`
}

// ClusterPlanStatus defines the observed state of Plan
// +k8s:openapi-gen=true
type ClusterPlanStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// +kubebuilder:webhook:name=clusterplans.compute.appvia.io,mutating=false,path=/validate/compute.appvia.io/clusterplans,verbs=create;update;delete,groups="compute.appvia.io",resources=clusterplans,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=clusterplans.compute.appvia.io,mutating=true,path=/mutate/compute.appvia.io/clusterplans,verbs=create;update,groups="compute.appvia.io",resources=clusterplans,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterPlan is the Schema for the plans API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=clusterplans,scope=Cluster,categories={wayfinder}
type ClusterPlan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterPlanSpec   `json:"spec,omitempty"`
	Status ClusterPlanStatus `json:"status,omitempty"`
}

func (ClusterPlan) APIPath() string {
	return "clusterplans"
}
func (p *ClusterPlan) Clone() corev1alpha1.Object {
	return p.DeepCopy()
}
func (p *ClusterPlan) CloneInto(o corev1alpha1.Object) {
	p.DeepCopyInto(o.(*ClusterPlan))
}
func (ClusterPlan) ListType() corev1alpha1.ObjectList {
	return &ClusterPlanList{}
}

func (p *ClusterPlan) GetCommonStatus() *corev1alpha1.CommonStatus {
	p.Status.SetStatusIfNotSet(corev1alpha1.SuccessStatus)
	return &p.Status.CommonStatus
}

// Implements the corev1alpha1.PlanObject interface
func (p *ClusterPlan) GetPolicies() []*corev1alpha1.PlanPolicy {
	return p.Spec.Policies
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterPlanList contains a list of ClusterPlans
type ClusterPlanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterPlan `json:"items"`
}

func (ClusterPlanList) ObjectType() corev1alpha1.Object {
	return &ClusterPlan{}
}
func (l *ClusterPlanList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *ClusterPlanList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*ClusterPlanList))
}

// AllocatedToWorkspace implements the Allocatable interface for Plan, it will return true if the
// specified workspace is allowed to use this plan according to the allocation settings on the
// spec.
func (p *ClusterPlan) AllocatedToWorkspace(ws corev1alpha1.WorkspaceKey) bool {
	return p.Spec.Allocation.Allowed(p.Namespace, ws)
}

func (l *ClusterPlanList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *ClusterPlanList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []ClusterPlan{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*ClusterPlan)))
	}
}
