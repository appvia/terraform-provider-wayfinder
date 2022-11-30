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

// WorkspaceKind is the kind for a workspace CRD
const WorkspaceKind = "Workspace"

// WorkspaceGroupVersionKind is the GVK for a workspace CRD
func WorkspaceGroupVersionKind() schema.GroupVersionKind {
	return schema.FromAPIVersionAndKind(GroupVersion.String(), WorkspaceKind)
}

// WorkspaceSpec defines the desired state of workspace
// +k8s:openapi-gen=true
type WorkspaceSpec struct {
	// Key is the unique identifier for this workspace
	Key corev1alpha1.WorkspaceKey `json:"key"`
	// Summary is a summary name for this workspace
	Summary string `json:"summary"`
	// Description is a description for the workspace
	Description string `json:"description"`
}

// WorkspaceStatus defines the observed state of workspace
// +k8s:openapi-gen=true
type WorkspaceStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Workspace is the Schema for the workspace API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=workspaces
type Workspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkspaceSpec   `json:"spec,omitempty"`
	Status WorkspaceStatus `json:"status,omitempty"`
}

func (Workspace) APIPath() string {
	return "workspaces"
}
func (t *Workspace) Clone() corev1alpha1.Object {
	return t.DeepCopy()
}
func (t *Workspace) CloneInto(o corev1alpha1.Object) {
	t.DeepCopyInto(o.(*Workspace))
}
func (Workspace) ListType() corev1alpha1.ObjectList {
	return &WorkspaceList{}
}

// GetCommonStatus implements CommonStatusAware
func (t *Workspace) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &t.Status.CommonStatus
}

// Key returns the unique key for this workspace
func (t Workspace) Key() corev1alpha1.WorkspaceKey {
	return corev1alpha1.WorkspaceKey(t.ObjectMeta.Name)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceList contains a list of workspaces
type WorkspaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workspace `json:"items"`
}

func (WorkspaceList) ObjectType() corev1alpha1.Object {
	return &Workspace{}
}
func (l *WorkspaceList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *WorkspaceList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*WorkspaceList))
}

func (l *WorkspaceList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *WorkspaceList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []Workspace{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*Workspace)))
	}
}
