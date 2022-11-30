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

// +kubebuilder:webhook:name=workspacerolebindings.org.appvia.io,mutating=false,path=/validate/org.appvia.io/workspacerolebindings,verbs=create;update;delete,groups="org.appvia.io",resources=workspacerolebindings,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=workspacerolebindings.org.appvia.io,mutating=true,path=/mutate/org.appvia.io/workspacerolebindings,verbs=create;update,groups="org.appvia.io",resources=workspacerolebindings,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// WorkspaceRoleBindingKind is the kind for granting a specific subject specific access
const (
	WorkspaceRoleBindingKind = "WorkspaceRoleBinding"
)

// WorkspaceRoleBindingGVK is the GVK for a WorkspaceRoleBinding
var WorkspaceRoleBindingGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    WorkspaceRoleBindingKind,
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceRoleBinding represents the binding of a workspace role to a workspace group or user
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=wayfinderworkspacerolebindings,scope=Namespaced,categories={wayfinder},shortName=wsrolebinding
type WorkspaceRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkspaceRoleBindingSpec   `json:"spec,omitempty"`
	Status WorkspaceRoleBindingStatus `json:"status,omitempty"`
}

func (WorkspaceRoleBinding) GroupVersionKind() schema.GroupVersionKind {
	return WorkspaceRoleBindingGVK
}
func (WorkspaceRoleBinding) APIPath() string {
	return "workspacerolebindings"
}
func (c *WorkspaceRoleBinding) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *WorkspaceRoleBinding) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*WorkspaceRoleBinding))
}
func (WorkspaceRoleBinding) ListType() corev1alpha1.ObjectList {
	return &WorkspaceRoleBindingList{}
}

// WorkspaceRoleBindingStatus defines the status of a WorkspaceRoleBinding
// +k8s:openapi-gen=true
type WorkspaceRoleBindingStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// GetCommonStatus implements CommonStatusAware
func (c *WorkspaceRoleBinding) GetCommonStatus() *corev1alpha1.CommonStatus {
	c.Status.SetStatusIfNotSet(corev1alpha1.SuccessStatus)
	return &c.Status.CommonStatus
}

// WorkspaceRoleBindingSpec defines the desired state of WorkspaceRoleBinding
// +k8s:openapi-gen=true
type WorkspaceRoleBindingSpec struct {
	// RoleRef is the role being bound
	// +kubebuilder:validation:Required
	RoleRef RoleRef `json:"roleRef"`

	// Subjects is the list of one or more subjects that this role is bound to
	// +kubebuilder:validation:Required
	Subjects []Subject `json:"subjects"`
}

// HasSubject returns true if the role binding contains the subject
func (g WorkspaceRoleBindingSpec) HasSubject(subject Subject) bool {
	for _, s := range g.Subjects {
		if s.Equals(subject) {
			return true
		}
	}

	return false
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceRoleBindingList provides a list of access role bindings
type WorkspaceRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkspaceRoleBinding `json:"items"`
}

func (WorkspaceRoleBindingList) ObjectType() corev1alpha1.Object {
	return &WorkspaceRoleBinding{}
}
func (l *WorkspaceRoleBindingList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *WorkspaceRoleBindingList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*WorkspaceRoleBindingList))
}

func (l *WorkspaceRoleBindingList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *WorkspaceRoleBindingList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []WorkspaceRoleBinding{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*WorkspaceRoleBinding)))
	}
}
