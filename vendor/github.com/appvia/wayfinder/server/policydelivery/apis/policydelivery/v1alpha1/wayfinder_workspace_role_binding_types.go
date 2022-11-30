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

// +kubebuilder:webhook:name=wayfinderworkspacerolebindings.policydelivery.appvia.io,mutating=false,path=/validate/policydelivery.appvia.io/wayfinderworkspacerolebindings,verbs=create;update;delete,groups="policydelivery.appvia.io",resources=wayfinderworkspacerolebindings,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// WayfinderWorkspaceRoleBindingKind is the kind for granting a specific subject specific access
const (
	WayfinderWorkspaceRoleBindingKind = "WayfinderWorkspaceRoleBinding"
)

// WayfinderWorkspaceRoleBindingGVK is the GVK for a WayfinderWorkspaceRoleBinding
var WayfinderWorkspaceRoleBindingGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    WayfinderWorkspaceRoleBindingKind,
}

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WayfinderWorkspaceRoleBinding represents actual access to a cluster for subjects
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=wayfinderworkspacerolebindings,scope=Namespaced,categories={wayfinder},shortName=wsrolebinding
type WayfinderWorkspaceRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WayfinderWorkspaceRoleBindingSpec   `json:"spec,omitempty"`
	Status WayfinderWorkspaceRoleBindingStatus `json:"status,omitempty"`
}

func (WayfinderWorkspaceRoleBinding) APIPath() string {
	return "wayfinderworkspacerolebindings"
}
func (c *WayfinderWorkspaceRoleBinding) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *WayfinderWorkspaceRoleBinding) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*WayfinderWorkspaceRoleBinding))
}
func (WayfinderWorkspaceRoleBinding) ListType() corev1alpha1.ObjectList {
	return &WayfinderWorkspaceRoleBindingList{}
}

// WayfinderWorkspaceRoleBindingStatus defines the status of an WayfinderWorkspaceRoleBinding
// +k8s:openapi-gen=true
type WayfinderWorkspaceRoleBindingStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// GetCommonStatus implements CommonStatusAware
func (c *WayfinderWorkspaceRoleBinding) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// WayfinderWorkspaceRoleBindingSpec defines the desired state of WayfinderWorkspaceRoleBinding
// Will create a ClusterPolicy with a role binding only
// +k8s:openapi-gen=true
type WayfinderWorkspaceRoleBindingSpec struct {
	// RoleRef is the role being bound - either a WayfinderRole or WayfinderWorkspaceRole
	// +kubebuilder:validation:Required
	RoleRef RoleRef `json:"roleRef"`

	// Subjects is the list of one or more subjects that this role is bound to
	// +kubebuilder:validation:Required
	Subjects []Subject `json:"subjects"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WayfinderWorkspaceRoleBindingList provides a list of access role bindings
type WayfinderWorkspaceRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WayfinderWorkspaceRoleBinding `json:"items"`
}

func (WayfinderWorkspaceRoleBindingList) ObjectType() corev1alpha1.Object {
	return &WayfinderWorkspaceRoleBinding{}
}
func (l *WayfinderWorkspaceRoleBindingList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *WayfinderWorkspaceRoleBindingList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*WayfinderWorkspaceRoleBindingList))
}

func (l *WayfinderWorkspaceRoleBindingList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *WayfinderWorkspaceRoleBindingList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []WayfinderWorkspaceRoleBinding{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*WayfinderWorkspaceRoleBinding)))
	}
}
