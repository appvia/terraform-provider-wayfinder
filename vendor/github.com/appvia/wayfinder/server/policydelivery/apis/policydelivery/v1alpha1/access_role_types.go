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
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// +kubebuilder:webhook:name=accessroles.policydelivery.appvia.io,mutating=false,path=/validate/policydelivery.appvia.io/accessroles,verbs=create;update;delete,groups="policydelivery.appvia.io",resources=accessroles,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// AccessRoleKind is the kind for an Access Role (a collection of Kubernetes policy rules)
// - Dictates what access can be granted
const AccessRoleKind = "AccessRole"

// AccessRoleGVK is the GVK for an AccessRole
var AccessRoleGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    AccessRoleKind,
}

// GlobalOrScopedAccessRole represents either type of access role in a consistent way
// +kubebuilder:object:generate=false
type GlobalOrScopedAccessRole interface {
	corev1alpha1.Object
	// GetAccessRoleSpec gives the applicable spec for this role
	GetAccessRoleSpec() AccessRoleSpec
}

// NewAccessRole creates a new AccessRole with the specified name and namespace
func NewAccessRole(name, namespace string) *AccessRole {
	return &AccessRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       AccessRoleKind,
			APIVersion: GroupVersion.String(),
		},
	}
}

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AccessRole represents a set of Kubernetes RBAC rules that can use used for user access
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=accessroles,scope=Namespaced,categories={wayfinder}
type AccessRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the specification of the access role
	// +kubebuilder:validation:Required
	Spec AccessRoleSpec `json:"spec,omitempty"`
	// Status is the status of the access role
	// +kubebuilder:validation:Optional
	Status AccessRoleStatus `json:"status,omitempty"`
}

func (AccessRole) APIPath() string {
	return "accessroles"
}
func (c *AccessRole) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *AccessRole) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*AccessRole))
}
func (AccessRole) ListType() corev1alpha1.ObjectList {
	return &AccessRoleList{}
}

func (c *AccessRole) GetAccessRoleSpec() AccessRoleSpec {
	return c.Spec
}

// AccessRoleSpec defines the desired state of AccessRole
// +k8s:openapi-gen=true
type AccessRoleSpec struct {
	// ClusterScope indicates if the access role is cluster-wide or not
	// will be used to validate if a AccessPolicyTarget is valid or not
	ClusterScope bool `json:"clusterScope,omitempty"`
	// Description is a human-readable description of the role
	// Should describe what the role would provide access to when bound to a user
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
	// Rules is rules
	// +kubebuilder:validation:Required
	Rules []rbacv1.PolicyRule `json:"rules"`
}

// AccessPolicyStatus defines the status of an AccessPolicy
// +k8s:openapi-gen=true
type AccessRoleStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// GetCommonStatus implements CommonStatusAware
func (c *AccessRole) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AccessRoleList provides a list of access roles
type AccessRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AccessRole `json:"items"`
}

func (AccessRoleList) ObjectType() corev1alpha1.Object {
	return &AccessRole{}
}
func (l *AccessRoleList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *AccessRoleList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*AccessRoleList))
}

func (l *AccessRoleList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *AccessRoleList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []AccessRole{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*AccessRole)))
	}
}
