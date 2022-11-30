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

// WorkspaceMemberKind is the kind for a workspace member CRD
const WorkspaceMemberKind = "WorkspaceMember"

// WorkspaceMemberGroupVersionKind is the GVK for a workspace CRD
func WorkspaceMemberGroupVersionKind() schema.GroupVersionKind {
	return schema.FromAPIVersionAndKind(GroupVersion.String(), WorkspaceMemberKind)
}

// WorkspaceMemberSpec defines the desired state of workspace member
// +k8s:openapi-gen=true
type WorkspaceMemberSpec struct {
	// Username is the user being bound to the workspace
	// +kubebuilder:validation:Required
	Username string `json:"username"`
}

// WorkspaceMemberStatus defines the observed state of workspace member
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceMember is the Schema for members of the workspace API
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=members
type WorkspaceMember struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec WorkspaceMemberSpec `json:"spec,omitempty"`
}

func (WorkspaceMember) APIPath() string {
	return "members"
}
func (c *WorkspaceMember) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *WorkspaceMember) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*WorkspaceMember))
}
func (WorkspaceMember) ListType() corev1alpha1.ObjectList {
	return &WorkspaceMemberList{}
}

func (WorkspaceMember) GetCommonStatus() *corev1alpha1.CommonStatus {
	return nil
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceMemberList contains a list of workspace members
type WorkspaceMemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkspaceMember `json:"items"`
}

func (WorkspaceMemberList) ObjectType() corev1alpha1.Object {
	return &WorkspaceMember{}
}
func (l *WorkspaceMemberList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *WorkspaceMemberList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*WorkspaceMemberList))
}
func (l *WorkspaceMemberList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}
func (l *WorkspaceMemberList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []WorkspaceMember{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*WorkspaceMember)))
	}
}
