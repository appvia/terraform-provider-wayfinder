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

// WorkspaceMemberRoleKind is the kind for a workspace member role CRD
const WorkspaceMemberRoleKind = "WorkspaceMemberRole"

// WorkspaceMemberGroupVersionKind is the GVK for a workspace member role CRD
func WorkspaceMemberRoleGroupVersionKind() schema.GroupVersionKind {
	return schema.FromAPIVersionAndKind(GroupVersion.String(), WorkspaceMemberRoleKind)
}

// WorkspaceMemberRoleSpec defines the desired state of WorkspaceMemberRole
type WorkspaceMemberRoleSpec struct {
	// User is the user in the workspace whom has the role
	User string `json:"user"`
	// Role is the role they have
	Role string `json:"role"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceMemberRole is the Schema for the workspace member roles API
type WorkspaceMemberRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec WorkspaceMemberRoleSpec `json:"spec,omitempty"`
}

func (WorkspaceMemberRole) APIPath() string {
	return "workspacememberroles"
}
func (c *WorkspaceMemberRole) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *WorkspaceMemberRole) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*WorkspaceMemberRole))
}
func (WorkspaceMemberRole) ListType() corev1alpha1.ObjectList {
	return &WorkspaceMemberRoleList{}
}

func (WorkspaceMemberRole) GetCommonStatus() *corev1alpha1.CommonStatus {
	return nil
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceMemberRoleList contains a list of WorkspaceMemberRole
type WorkspaceMemberRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkspaceMemberRole `json:"items"`
}

func (WorkspaceMemberRoleList) ObjectType() corev1alpha1.Object {
	return &WorkspaceMemberRole{}
}
func (l *WorkspaceMemberRoleList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *WorkspaceMemberRoleList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*WorkspaceMemberRoleList))
}
func (l *WorkspaceMemberRoleList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}
func (l *WorkspaceMemberRoleList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []WorkspaceMemberRole{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*WorkspaceMemberRole)))
	}
}
