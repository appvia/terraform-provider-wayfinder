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
	"k8s.io/apimachinery/pkg/types"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// +kubebuilder:webhook:name=wayfinderrolebindings.org.appvia.io,mutating=false,path=/validate/org.appvia.io/wayfinderrolebindings,verbs=create;update;delete,groups="org.appvia.io",resources=wayfinderrolebindings,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=wayfinderrolebindings.org.appvia.io,mutating=true,path=/mutate/org.appvia.io/wayfinderrolebindings,verbs=create;update,groups="org.appvia.io",resources=wayfinderrolebindings,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// WayfinderRoleBindingKind is the kind for granting a specific subject specific access
const (
	WayfinderRoleBindingKind = "WayfinderRoleBinding"
)

// WayfinderRoleBindingGVK is the GVK for a WayfinderRoleBinding
var WayfinderRoleBindingGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    WayfinderRoleBindingKind,
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WayfinderRoleBinding represents actual access to a cluster for subjects
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=wayfinderrolebindings,scope=Cluster,categories={wayfinder},shortName=wfrolebinding
type WayfinderRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WayfinderRoleBindingSpec   `json:"spec,omitempty"`
	Status WayfinderRoleBindingStatus `json:"status,omitempty"`
}

func (WayfinderRoleBinding) GroupVersionKind() schema.GroupVersionKind {
	return WayfinderRoleBindingGVK
}

func (WayfinderRoleBinding) APIPath() string {
	return "wayfinderrolebindings"
}
func (c *WayfinderRoleBinding) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *WayfinderRoleBinding) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*WayfinderRoleBinding))
}
func (WayfinderRoleBinding) ListType() corev1alpha1.ObjectList {
	return &WayfinderRoleBindingList{}
}

// WayfinderRoleBindingStatus defines the status of an WayfinderRoleBinding
// +k8s:openapi-gen=true
type WayfinderRoleBindingStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// GetCommonStatus implements CommonStatusAware
func (c *WayfinderRoleBinding) GetCommonStatus() *corev1alpha1.CommonStatus {
	c.Status.SetStatusIfNotSet(corev1alpha1.SuccessStatus)
	return &c.Status.CommonStatus
}

// RoleRef defines a reference to a Wayfinder or Workspace role
type RoleRef struct {
	// Kind of object being referenced, for future compatibility. At this time, the only value used
	// is WayfinderRole.
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`
	// Name of the object being referenced.
	Name string `json:"name"`
	// Namespace of the referenced object (not applicable to WayfinderRole).
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
}

func (r RoleRef) ToNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: r.Namespace,
		Name:      r.Name,
	}
}

// WayfinderRoleBindingSpec defines the desired state of WayfinderRoleBinding
// Will create a ClusterPolicy with a role binding only
// +k8s:openapi-gen=true
type WayfinderRoleBindingSpec struct {
	// RoleRef is the role being bound
	// +kubebuilder:validation:Required
	RoleRef RoleRef `json:"roleRef"`

	// Subjects is the list of one or more subjects that this role is bound to
	// +kubebuilder:validation:Required
	Subjects []Subject `json:"subjects"`
}

// HasSubject returns true if the role binding contains the subject
func (g WayfinderRoleBindingSpec) HasSubject(subject Subject) bool {
	for _, s := range g.Subjects {
		if s.Equals(subject) {
			return true
		}
	}

	return false
}

// WayfinderRoleBindingList provides a list of access role bindings
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WayfinderRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WayfinderRoleBinding `json:"items"`
}

func (WayfinderRoleBindingList) ObjectType() corev1alpha1.Object {
	return &WayfinderRoleBinding{}
}
func (l *WayfinderRoleBindingList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *WayfinderRoleBindingList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*WayfinderRoleBindingList))
}

func (l *WayfinderRoleBindingList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *WayfinderRoleBindingList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []WayfinderRoleBinding{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*WayfinderRoleBinding)))
	}
}
