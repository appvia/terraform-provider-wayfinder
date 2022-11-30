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

// +kubebuilder:webhook:name=wayfinderroles.policydelivery.appvia.io,mutating=false,path=/validate/policydelivery.appvia.io/wayfinderroles,verbs=create;update;delete,groups="policydelivery.appvia.io",resources=wayfinderroles,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// WayfinderRoleKind is the kind for a Wayfinder Role which provides permissions to access objects
// in Wayfinder at workspace or global level
const WayfinderRoleKind = "WayfinderRole"

// WayfinderRoleGVK is the GVK for an AccessRole
var WayfinderRoleGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    WayfinderRoleKind,
}

// RBACRuleSource is an interface implemented by the various sources of RBAC rules in this API
// +kubebuilder:object:generate=false
type RBACRuleSource interface {
	GetRules() []rbacv1.PolicyRule
}

// NewWayfinderRole creates a new WayfinderRole with the specified name
func NewWayfinderRole(name, namespace string) *WayfinderRole {
	return &WayfinderRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WayfinderRole represents a set of Kubernetes RBAC-style rules that can describe permission for
// users against Wayfinder itself
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=wayfinderroles,scope=Cluster,categories={wayfinder},shortName=wfrole
type WayfinderRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the specification of the role
	// +kubebuilder:validation:Required
	Spec WayfinderRoleSpec `json:"spec,omitempty"`
	// Status is the status of the role
	// +kubebuilder:validation:Optional
	Status WayfinderRoleStatus `json:"status,omitempty"`
}

func (WayfinderRole) APIPath() string {
	return "wayfinderroles"
}
func (c *WayfinderRole) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *WayfinderRole) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*WayfinderRole))
}
func (WayfinderRole) ListType() corev1alpha1.ObjectList {
	return &WayfinderRoleList{}
}

// WayfinderRoleScope defines the scope of a role - i.e. whether it applies across Wayfinder or to
// only those objects in a single workspace
// +kubebuilder:validation:Enum=Global;Workspace
type WayfinderRoleScope string

const (
	// WayfinderRoleScopeGlobal provides Wayfinder-wide access to the specied resources
	WayfinderRoleScopeGlobal WayfinderRoleScope = "Global"
	// WayfinderRoleScopeWorkspace provides access to the specified resources in a workspace
	WayfinderRoleScopeWorkspace WayfinderRoleScope = "Workspace"
)

// WayfinderRoleSpec defines a set of RBAC rules to Wayfinder objects
// +k8s:openapi-gen=true
type WayfinderRoleSpec struct {
	// Scope indicates what this access role provides permission to. A 'Workspace' scoped role bound
	// to a subject will permit access to the specified resources in a workspace ONLY if the subject
	// is a member of that workspace.
	// +kubebuilder:validation:Required
	Scope WayfinderRoleScope `json:"scope,omitempty"`
	// Description is a human-readable description of the role
	// Should describe what the role would provide access to when bound to a user/group/robot
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
	// Rules is rules
	// +kubebuilder:validation:Required
	Rules []rbacv1.PolicyRule `json:"rules"`
}

// GetRules implements RBACRuleSource for WayfinderRole
func (c *WayfinderRole) GetRules() []rbacv1.PolicyRule {
	if c == nil {
		return []rbacv1.PolicyRule{}
	}
	return c.Spec.Rules
}

// WayfinderRoleStatus defines the status of a WayfinderRole
// +k8s:openapi-gen=true
type WayfinderRoleStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// GetCommonStatus implements CommonStatusAware
func (c *WayfinderRole) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WayfinderRoleList provides a list of global access roles
type WayfinderRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WayfinderRole `json:"items"`
}

func (WayfinderRoleList) ObjectType() corev1alpha1.Object {
	return &WayfinderRole{}
}
func (l *WayfinderRoleList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *WayfinderRoleList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*WayfinderRoleList))
}

func (l *WayfinderRoleList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *WayfinderRoleList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []WayfinderRole{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*WayfinderRole)))
	}
}
