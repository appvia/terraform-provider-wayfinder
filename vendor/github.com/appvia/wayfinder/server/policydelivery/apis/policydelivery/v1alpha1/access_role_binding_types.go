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
	"k8s.io/apimachinery/pkg/types"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// +kubebuilder:webhook:name=accessrolebindings.policydelivery.appvia.io,mutating=false,path=/validate/policydelivery.appvia.io/accessrolebindings,verbs=create;update;delete,groups="policydelivery.appvia.io",resources=accessrolebindings,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// AccessRoleBindingKind is the kind for granting a specific subject specific access
const (
	AccessRoleBindingKind = "AccessRoleBinding"
	// AccessRoleBindingParamSubject is the parameter name for filtering by subject
	AccessRoleBindingParamSubject = "subject"
	// AccessRoleBindingParamSubjectKind is the parameter name for filtering by subject kind
	AccessRoleBindingParamSubjectKind = "subjectkind"
	// AccessRoleBindingParamSubject is the parameter name for filtering by role
	AccessRoleBindingParamRole = "role"
)

// AccessRoleBindingGVK is the GVK for a AccessRoleBinding
var AccessRoleBindingGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    AccessRoleBindingKind,
}

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AccessRoleBinding represents actual access to a cluster for subjects
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=accessrolebindings,scope=Namespaced,categories={wayfinder}
type AccessRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccessRoleBindingSpec   `json:"spec,omitempty"`
	Status AccessRoleBindingStatus `json:"status,omitempty"`
}

func (AccessRoleBinding) APIPath() string {
	return "accessrolebindings"
}
func (a *AccessRoleBinding) Clone() corev1alpha1.Object {
	return a.DeepCopy()
}
func (a *AccessRoleBinding) CloneInto(o corev1alpha1.Object) {
	a.DeepCopyInto(o.(*AccessRoleBinding))
}
func (AccessRoleBinding) ListType() corev1alpha1.ObjectList {
	return &AccessRoleBindingList{}
}

// GetNamespacedName gets the NamespacedName for the AccessRoleBinding
func (a *AccessRoleBinding) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: a.Namespace,
		Name:      a.Name,
	}
}

// GetNamespaceClaimNamespaceName gets a NamespacedName for the NamespaceClaim referenced
func (a *AccessRoleBinding) GetNamespaceClaimNamespaceName() types.NamespacedName {
	if a.Spec.NamespaceClaimRef == nil {
		return types.NamespacedName{}
	}
	return types.NamespacedName{
		Namespace: a.Namespace,
		Name:      *a.Spec.NamespaceClaimRef,
	}
}

// GetAccessRoleNamespacedName gets a NamespacedName for the AccessRole referenced
func (a *AccessRoleBinding) GetAccessRoleNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: a.Namespace,
		Name:      string(a.Spec.AccessRoleRef),
	}
}

// AccessRoleBindingStatus defines the status of an AccessRoleBinding
// +k8s:openapi-gen=true
type AccessRoleBindingStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// GetCommonStatus implements CommonStatusAware
func (a *AccessRoleBinding) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &a.Status.CommonStatus
}

// AccessRoleBindingSpec defines the desired state of AccessRoleBinding
// Will create a ClusterPolicy with a role binding only
// +k8s:openapi-gen=true
type AccessRoleBindingSpec struct {
	// Expiration is the expiration time for any policy to exist
	// Omitting this will make the policy persistent
	// +kubebuilder:validation:Optional
	Expiration *metav1.Time `json:"expiration,omitempty"`

	// AccessRoleRef is the reference to the AccessRole (in this namespace)
	// +kubebuilder:validation:Required
	AccessRoleRef string `json:"accessRoleRef"`

	// ClusterRef provides reference to the cluster for this access
	// Must be specified if NamespaceClaimRef is not set
	// Must be specified if the reference role specified ClusterWide
	// +kubebuilder:validation:Optional
	ClusterRef *corev1alpha1.Ownership `json:"clusterRef,omitempty"`

	// NamespaceClaimRef is the namespacesClaim this binding is valid for
	// Will create a roleBinding for the namespace referenced
	// Must be specified if the AccessRoleRef specifies ClusterScope=false
	// Is an error if the ClusterScope=true
	// +kubebuilder:validation:Optional
	NamespaceClaimRef *string `json:"namespaceClaimRef,omitempty"`

	// Subject is the user or robot that has this access in Wayfinder
	// +kubebuilder:validation:Required
	Subject Subject `json:"subject"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AccessRoleBindingList provides a list of access role bindings
type AccessRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AccessRoleBinding `json:"items"`
}

func (AccessRoleBindingList) ObjectType() corev1alpha1.Object {
	return &AccessRoleBinding{}
}
func (l *AccessRoleBindingList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *AccessRoleBindingList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*AccessRoleBindingList))
}

func (l *AccessRoleBindingList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *AccessRoleBindingList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []AccessRoleBinding{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*AccessRoleBinding)))
	}
}
