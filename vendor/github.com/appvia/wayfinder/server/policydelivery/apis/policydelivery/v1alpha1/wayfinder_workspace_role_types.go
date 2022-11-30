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

// +kubebuilder:webhook:name=wayfinderworkspaceroles.policydelivery.appvia.io,mutating=false,path=/validate/policydelivery.appvia.io/wayfinderworkspaceroles,verbs=create;update;delete,groups="policydelivery.appvia.io",resources=wayfinderworkspaceroles,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// WayfinderWorkspaceRoleKind is the kind for a Wayfinder Role which provides permissions to access
// objects in Wayfinder at workspace level
const WayfinderWorkspaceRoleKind = "WayfinderWorkspaceRole"

// WayfinderWorkspaceRoleGVK is the GVK for an AccessRole
var WayfinderWorkspaceRoleGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    WayfinderWorkspaceRoleKind,
}

// NewWayfinderWorkspaceRole creates a new WayfinderWorkspaceRole with the specified name
func NewWayfinderWorkspaceRole(name, namespace string) *WayfinderWorkspaceRole {
	return &WayfinderWorkspaceRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WayfinderWorkspaceRole represents a set of Kubernetes RBAC-style rules that can describe
// permission for users against Wayfinder itself
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=wayfinderworkspaceroles,scope=Namespaced,categories={wayfinder},shortName=wsrole
type WayfinderWorkspaceRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the specification of the role
	// +kubebuilder:validation:Required
	Spec WayfinderWorkspaceRoleSpec `json:"spec,omitempty"`
	// Status is the status of the role
	// +kubebuilder:validation:Optional
	Status WayfinderWorkspaceRoleStatus `json:"status,omitempty"`
}

func (WayfinderWorkspaceRole) APIPath() string {
	return "wayfinderworkspaceroles"
}
func (c *WayfinderWorkspaceRole) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *WayfinderWorkspaceRole) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*WayfinderWorkspaceRole))
}
func (WayfinderWorkspaceRole) ListType() corev1alpha1.ObjectList {
	return &WayfinderWorkspaceRoleList{}
}

// GetRules implements RBACRuleSource for WayfinderWorkspaceRole
func (c *WayfinderWorkspaceRole) GetRules() []rbacv1.PolicyRule {
	if c == nil {
		return []rbacv1.PolicyRule{}
	}
	return c.Spec.Rules
}

// WayfinderWorkspaceRoleSpec defines a set of RBAC rules to workspace-level Wayfinder objects
// +k8s:openapi-gen=true
type WayfinderWorkspaceRoleSpec struct {
	// Description is a human-readable description of the role
	// Should describe what the role would provide access to when bound to a user/group/robot
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
	// Rules is rules
	// +kubebuilder:validation:Required
	Rules []rbacv1.PolicyRule `json:"rules"`
}

// WayfinderWorkspaceRoleStatus defines the status of a WayfinderWorkspaceRole
// +k8s:openapi-gen=true
type WayfinderWorkspaceRoleStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// GetCommonStatus implements CommonStatusAware
func (c *WayfinderWorkspaceRole) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WayfinderWorkspaceRoleList provides a list of global access roles
type WayfinderWorkspaceRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WayfinderWorkspaceRole `json:"items"`
}

func (WayfinderWorkspaceRoleList) ObjectType() corev1alpha1.Object {
	return &WayfinderWorkspaceRole{}
}
func (l *WayfinderWorkspaceRoleList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *WayfinderWorkspaceRoleList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*WayfinderWorkspaceRoleList))
}

func (l *WayfinderWorkspaceRoleList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *WayfinderWorkspaceRoleList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []WayfinderWorkspaceRole{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*WayfinderWorkspaceRole)))
	}
}
