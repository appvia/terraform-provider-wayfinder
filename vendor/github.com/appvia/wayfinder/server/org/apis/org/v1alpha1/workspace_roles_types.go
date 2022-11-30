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

// WorkspaceRoleKind is the kind for a workspace role CRD
const WorkspaceRoleKind = "WorkspaceRole"

// WorkspaceGroupVersionKind is the GVK for a workspace role CRD
func WorkspaceRoleGroupVersionKind() schema.GroupVersionKind {
	return schema.FromAPIVersionAndKind(GroupVersion.String(), WorkspaceRoleKind)
}

// WorkspaceRoleSpec defines the desired state of WorkspaceRole
// +k8s:openapi-gen=true
type WorkspaceRoleSpec struct {
	// Description is a description for the workspace role
	Description string `json:"description"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceRole is the Schema for the workspace roles API
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=workspaceroles
type WorkspaceRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec WorkspaceRoleSpec `json:"spec,omitempty"`
}

func (WorkspaceRole) APIPath() string {
	return "workspaceroles"
}
func (c *WorkspaceRole) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *WorkspaceRole) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*WorkspaceRole))
}
func (WorkspaceRole) ListType() corev1alpha1.ObjectList {
	return &WorkspaceMemberList{}
}

func (WorkspaceRole) GetCommonStatus() *corev1alpha1.CommonStatus {
	return nil
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceRoleList contains a list of WorkspaceRole
type WorkspaceRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkspaceRole `json:"items"`
}

func (WorkspaceRoleList) ObjectType() corev1alpha1.Object {
	return &WorkspaceRole{}
}
func (l *WorkspaceRoleList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *WorkspaceRoleList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*WorkspaceRoleList))
}

func (l *WorkspaceRoleList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *WorkspaceRoleList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []WorkspaceRole{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*WorkspaceRole)))
	}
}
