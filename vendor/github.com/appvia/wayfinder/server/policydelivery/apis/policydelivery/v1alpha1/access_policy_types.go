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
	"github.com/appvia/wayfinder/pkg/common"
	"github.com/appvia/wayfinder/pkg/utils"
)

// +kubebuilder:webhook:name=accesspolicies.policydelivery.appvia.io,mutating=false,path=/validate/policydelivery.appvia.io/accesspolicies,verbs=create;update;delete,groups="policydelivery.appvia.io",resources=accesspolicies,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// AccessPolicyKind is the kind for an Access Policy (who could access a policy and where)
// - Dictates who can access what and when
// - Gives permission for a set of subjects to create AccessRoleBindings
// - Specifies where Roles should be distributed to
const AccessPolicyKind = "AccessPolicy"

// AccessPolicyGVK is the GVK for an AccessPolicy
var AccessPolicyGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    AccessPolicyKind,
}

// GlobalOrScopedAccessPolicy represents either type of access policy in a consistent way
// +kubebuilder:object:generate=false
type GlobalOrScopedAccessPolicy interface {
	corev1alpha1.Object
	// GetAccessPolicySpec gives the applicable spec for this policy
	GetAccessPolicySpec() AccessPolicySpec
}

// NewAccessPolicy creates a new AccessPolicy with the specified name and namespace
func NewAccessPolicy(name, namespace string) *AccessPolicy {
	return &AccessPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       AccessPolicyKind,
			APIVersion: GroupVersion.String(),
		},
	}
}

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AccessPolicy represents a targeted policy
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=accesspolicies,scope=Namespaced,categories={wayfinder}
type AccessPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the specification of the access policy
	// +kubebuilder:validation:Required
	Spec AccessPolicySpec `json:"spec,omitempty"`
	// Status is the status of the access policy
	// +kubebuilder:validation:Optional
	Status AccessPolicyStatus `json:"status,omitempty"`
}

func (AccessPolicy) APIPath() string {
	return "accesspolicies"
}
func (c *AccessPolicy) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *AccessPolicy) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*AccessPolicy))
}
func (AccessPolicy) ListType() corev1alpha1.ObjectList {
	return &AccessPolicyList{}
}

func (c *AccessPolicy) GetAccessPolicySpec() AccessPolicySpec {
	return c.Spec
}

// Enabled checks that this policy is enabled
func (c *AccessPolicy) Enabled() bool {
	// It's important to return true unless explicitly disabled
	return common.IsEnabled(c)
}

// Targets checks if this policy targets a given role
func (c *AccessPolicy) Targets(role string) bool {
	return utils.Contains(role, c.Spec.AccessRoleRefs)
}

// AccessPolicySpec defines the desired state of AccessPolicy
// +k8s:openapi-gen=true
type AccessPolicySpec struct {
	// AccessRoleRefs is the reference to AccessRoles (in this workspace)
	// grants access to create AccessRoleBindings for the exact roles only
	// This is the "what" exactly is granted access to
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	AccessRoleRefs []string `json:"accessRoleRefs"`
	// Description is a human-readable description of the policy
	// Should describe what sort of function the policy provides (e.g. support for production clusters)
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
	// Groups are the list of Wayfinder groups that apply this AccessRoleBindings
	// This is the who in the access question
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Groups []string `json:"groups"`
	// Constraints are the collection of constraints which control access to the roles
	// This is a combination of when and where
	// +kubebuilder:validation:Required
	Constraints Constraints `json:"constraints"`
	// SubjectKind is the type of the subject for this policy
	// +kubebuilder:validation:Enum=User;Robot
	// +kubebuilder:validation:Required
	SubjectKind string `json:"subjectType"`
}

// AccessPolicyStatus defines the status of an AccessPolicy
// +k8s:openapi-gen=true
type AccessPolicyStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// GetCommonStatus implements CommonStatusAware
func (c *AccessPolicy) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AccessPolicyList provides a list of access policies
type AccessPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AccessPolicy `json:"items"`
}

func (AccessPolicyList) ObjectType() corev1alpha1.Object {
	return &AccessPolicy{}
}
func (l *AccessPolicyList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *AccessPolicyList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*AccessPolicyList))
}

func (l *AccessPolicyList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *AccessPolicyList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []AccessPolicy{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*AccessPolicy)))
	}
}

// HasTargetForType helps to filter roles for ui/cli
func (l *AccessPolicyList) HasTargetForType(role string, t SubjectKind) bool {
	for _, policy := range l.Items {
		if policy.Enabled() && policy.Spec.SubjectKind == string(t) && policy.Targets(role) {
			return true
		}
	}
	return false
}
