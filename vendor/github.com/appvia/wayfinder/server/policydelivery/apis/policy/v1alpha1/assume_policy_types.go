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
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// AssumePolicy is the kind for a AssumePolicy
const AssumePolicyKind = "AssumePolicy"

// AssumePolicyGVK is the GVK for a AssumePolicy
var AssumePolicyGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    AssumePolicyKind,
}

// AssumePolicySpec defines the desired state of policy
type AssumePolicySpec struct {
	// Summary is a optional summary the describes the policy
	// +kubebuilder:validation:Optional
	Summary string `json:"summary,omitempty"`
	// Roles is the permitted roles your are able to assume from this
	// policy
	// +kubebuilder:validation:MinSize=1
	// +kubebuilder:validation:Required
	Roles []string `json:"roles"`
	// Constraints is a collection of constraints which control access the roles
	// +kubebuilder:validation:Required
	Constraints Constraints `json:"constraints"`
}

// AssumePolicyStatus defines the observed state of status on a policy
// +k8s:openapi-gen=true
type AssumePolicyStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AssumePolicy is the Schema for the policies API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=assumepolicies,scope=Namespaced,categories={wayfinder}
type AssumePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AssumePolicySpec   `json:"spec,omitempty"`
	Status AssumePolicyStatus `json:"status,omitempty"`
}

func (AssumePolicy) APIPath() string {
	return "assumepolicies"
}
func (c *AssumePolicy) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *AssumePolicy) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*AssumePolicy))
}
func (AssumePolicy) ListType() corev1alpha1.ObjectList {
	return &AssumePolicyList{}
}

// GetCommonStatus implements CommonStatusAware for CloudAccount
func (c *AssumePolicy) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AssumePolicyList contains a list of policies
type AssumePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AssumePolicy `json:"items"`
}

func (AssumePolicyList) ObjectType() corev1alpha1.Object {
	return &AssumePolicy{}
}
func (l *AssumePolicyList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *AssumePolicyList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*AssumePolicyList))
}

// GetItems returns a list of abstract objects
func (l *AssumePolicyList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))

	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}

	return res
}

// SetItems casts and sets the list
func (l *AssumePolicyList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []AssumePolicy{}

	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*AssumePolicy)))
	}
}
