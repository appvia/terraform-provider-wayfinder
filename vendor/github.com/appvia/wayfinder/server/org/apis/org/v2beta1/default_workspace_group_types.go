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

// +kubebuilder:webhook:name=defaultworkspacegroups.org.appvia.io,mutating=false,path=/validate/org.appvia.io/defaultworkspacegroups,verbs=create;update;delete,groups="org.appvia.io",resources=defaultworkspacegroups,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=defaultworkspacegroups.org.appvia.io,mutating=true,path=/mutate/org.appvia.io/defaultworkspacegroups,verbs=create;update,groups="org.appvia.io",resources=defaultworkspacegroups,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// DefaultWorkspaceGroupKind is the kind for a workspace group CRD
const DefaultWorkspaceGroupKind = "DefaultWorkspaceGroup"

// DefaultWorkspaceGroupGVK is the GVK for a DefaultWorkspaceGroup
var DefaultWorkspaceGroupGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    DefaultWorkspaceGroupKind,
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DefaultWorkspaceGroup represents a desire for a default workspace group which will be created in
// each workspace when that workspace is created.
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=defaultworkspacegroups,scope=Cluster,categories={wayfinder}
type DefaultWorkspaceGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the desired state of the group
	// +kubebuilder:validation:Required
	Spec DefaultWorkspaceGroupSpec `json:"spec,omitempty"`
	// Status is the status of the group
	// +kubebuilder:validation:Optional
	Status DefaultWorkspaceGroupStatus `json:"status,omitempty"`
}

func (DefaultWorkspaceGroup) GroupVersionKind() schema.GroupVersionKind {
	return DefaultWorkspaceGroupGVK
}

func (DefaultWorkspaceGroup) APIPath() string {
	return "defaultworkspacegroups"
}
func (c *DefaultWorkspaceGroup) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *DefaultWorkspaceGroup) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*DefaultWorkspaceGroup))
}
func (DefaultWorkspaceGroup) ListType() corev1alpha1.ObjectList {
	return &DefaultWorkspaceGroupList{}
}

func (c *DefaultWorkspaceGroup) GetCommonStatus() *corev1alpha1.CommonStatus {
	c.Status.SetStatusIfNotSet(corev1alpha1.SuccessStatus)
	return &c.Status.CommonStatus
}

// DefaultWorkspaceGroupSpec defines the specification for a default workspace group
// +k8s:openapi-gen=true
type DefaultWorkspaceGroupSpec struct {
	// Description provides a human-readable description of this group
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Description string `json:"description"`
}

// DefaultWorkspaceGroupStatus defines the status of a default group
// +k8s:openapi-gen=true
type DefaultWorkspaceGroupStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// DefaultWorkspaceGroupList contains a list of WorkspaceGroups
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DefaultWorkspaceGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DefaultWorkspaceGroup `json:"items"`
}

func (DefaultWorkspaceGroupList) ObjectType() corev1alpha1.Object {
	return &WorkspaceGroup{}
}
func (l *DefaultWorkspaceGroupList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *DefaultWorkspaceGroupList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*DefaultWorkspaceGroupList))
}

func (l *DefaultWorkspaceGroupList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *DefaultWorkspaceGroupList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []DefaultWorkspaceGroup{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*DefaultWorkspaceGroup)))
	}
}
