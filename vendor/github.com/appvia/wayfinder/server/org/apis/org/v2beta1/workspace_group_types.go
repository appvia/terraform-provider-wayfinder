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

// +kubebuilder:webhook:name=workspacegroups.org.appvia.io,mutating=false,path=/validate/org.appvia.io/workspacegroups,verbs=create;update;delete,groups="org.appvia.io",resources=workspacegroups,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=workspacegroups.org.appvia.io,mutating=true,path=/mutate/org.appvia.io/workspacegroups,verbs=create;update,groups="org.appvia.io",resources=workspacegroups,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// WorkspaceGroupKind is the kind for a workspace group CRD
const WorkspaceGroupKind = "WorkspaceGroup"

// WorkspaceGroupGVK is the GVK for a WorkspaceGroup
var WorkspaceGroupGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    WorkspaceGroupKind,
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceGroup is the Schema for the workspace groups API
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=workspacegroups,scope=Namespaced,categories={wayfinder},shortName=wsgroup
type WorkspaceGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the desired state of the group
	// +kubebuilder:validation:Required
	Spec GroupSpec `json:"spec,omitempty"`
	// Status is the status of the group
	// +kubebuilder:validation:Optional
	Status WorkspaceGroupStatus `json:"status,omitempty"`
}

func (WorkspaceGroup) GroupVersionKind() schema.GroupVersionKind {
	return WorkspaceGroupGVK
}

func (WorkspaceGroup) APIPath() string {
	return "workspacegroups"
}
func (c *WorkspaceGroup) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *WorkspaceGroup) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*WorkspaceGroup))
}
func (WorkspaceGroup) ListType() corev1alpha1.ObjectList {
	return &WorkspaceGroupList{}
}

func (c *WorkspaceGroup) GetCommonStatus() *corev1alpha1.CommonStatus {
	c.Status.SetStatusIfNotSet(corev1alpha1.SuccessStatus)
	return &c.Status.CommonStatus
}

func (c *WorkspaceGroup) ToSubject() Subject {
	return Subject{
		Kind: SubjectKindWorkspaceGroup,
		Name: c.Name,
	}
}

// WorkspaceGroupStatus defines the status of a group
// +k8s:openapi-gen=true
type WorkspaceGroupStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// WorkspaceGroupList contains a list of WorkspaceGroups
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkspaceGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkspaceGroup `json:"items"`
}

func (WorkspaceGroupList) ObjectType() corev1alpha1.Object {
	return &WorkspaceGroup{}
}
func (l *WorkspaceGroupList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *WorkspaceGroupList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*WorkspaceGroupList))
}

func (l *WorkspaceGroupList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *WorkspaceGroupList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []WorkspaceGroup{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*WorkspaceGroup)))
	}
}
