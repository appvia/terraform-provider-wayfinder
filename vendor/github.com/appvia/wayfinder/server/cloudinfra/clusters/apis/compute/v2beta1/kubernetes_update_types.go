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

// +kubebuilder:webhook:name=kubernetesupdates.compute.appvia.io,mutating=false,path=/validate/compute.appvia.io/kubernetesupdates,verbs=create;update;delete,groups="compute.appvia.io",resources=kubernetesupdates,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=kubernetesupdates.compute.appvia.io,mutating=true,path=/mutate/compute.appvia.io/kubernetesupdates,verbs=create;update,groups="compute.appvia.io",resources=kubernetesupdates,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// KubernetesUpdateKind is the kind for a KubernetesUpdate
const KubernetesUpdateKind = "KubernetesUpdate"

// KubernetesUpdateGVK is the GVK for a KubernetesUpdate
var KubernetesUpdateGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    KubernetesUpdateKind,
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubernetesUpdate is the schema for Kubernetes version updates to clusters
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=kubernetesupdates,scope=Namespaced,categories={wayfinder}
type KubernetesUpdate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KubernetesUpdateSpec   `json:"spec,omitempty"`
	Status KubernetesUpdateStatus `json:"status,omitempty"`
}

func (KubernetesUpdate) APIPath() string {
	return "kubernetesupdates"
}
func (k *KubernetesUpdate) Clone() corev1alpha1.Object {
	return k.DeepCopy()
}
func (k *KubernetesUpdate) CloneInto(o corev1alpha1.Object) {
	k.DeepCopyInto(o.(*KubernetesUpdate))
}
func (KubernetesUpdate) ListType() corev1alpha1.ObjectList {
	return &KubernetesUpdateList{}
}
func (k *KubernetesUpdate) IsReady() bool {
	return k.Status.InCondition(corev1alpha1.ConditionReady)
}

// IsComplete implements the IsComplete method for the UpdateObject interface
func (k *KubernetesUpdate) IsComplete() bool {
	return k.Status.EstimatedPercentageComplete == 100
}

// EstimatePercentageComplete implements the EstimatePercentageComplete method for the UpdateObject interface
func (k *KubernetesUpdate) EstimatePercentageComplete() int {
	return k.Status.EstimatedPercentageComplete
}

// IsAwaitingApproval implements the IsAwaitingApproval method for the UpdateObject interface
func (k *KubernetesUpdate) IsAwaitingApproval() bool {
	return k.Spec.AwaitingApproval
}

// GetPreRequisite implements the GetPreRequisite method for the UpdateObject interface
func (k *KubernetesUpdate) GetPreRequisite() *corev1alpha1.UpdateStep {
	return k.Spec.PreRequisite
}

// KubernetesUpdateSpec defines an update to the kubernetes version
type KubernetesUpdateSpec struct {
	corev1alpha1.UpdateSpec `json:",inline"`
	// TargetKubernetesVersion is the specific Kubernetes version to use for the new node pool spec
	// - Normally be an increment of the current, existing Kubernetes version
	// - should satisfy the rules of the Kubernetes versioning scheme as per https://kubernetes.io/docs/setup/release/version-skew-policy/
	// +kubebuilder:validation:Required
	TargetKubernetesVersion string `json:"targetKubernetesVersion"`
}

// KubernetesUpdateStatus defines the observed state of an update
type KubernetesUpdateStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// StartTime is the time the update was started
	// Is used to estimate the percentage complete time
	// +kubebuilder:validation:Optional
	StartTime metav1.Time `json:"startTime,omitempty"`
	// EstimatedPercentageComplete is the estimated percentage complete of the update
	// - Based on the time from StartTime and a test of actual updates
	// +kubebuilder:validation:Optional
	EstimatedPercentageComplete int `json:"estimatedPercentageComplete,omitempty"`
}

// GetCommonStatus implements CommonStatusAware
func (k *KubernetesUpdate) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &k.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubernetesUpdateList provides a list of KubernetesUpdate objects
type KubernetesUpdateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubernetesUpdate `json:"items"`
}

func (KubernetesUpdateList) ObjectType() corev1alpha1.Object {
	return &KubernetesUpdate{}
}
func (l *KubernetesUpdateList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *KubernetesUpdateList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*KubernetesUpdateList))
}

func (l *KubernetesUpdateList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *KubernetesUpdateList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []KubernetesUpdate{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*KubernetesUpdate)))
	}
}
