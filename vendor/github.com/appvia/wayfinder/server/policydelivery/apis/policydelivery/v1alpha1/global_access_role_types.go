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

// +kubebuilder:webhook:name=globalaccessroles.policydelivery.appvia.io,mutating=false,path=/validate/policydelivery.appvia.io/globalaccessroles,verbs=create;update;delete,groups="policydelivery.appvia.io",resources=globalaccessroles,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// GlobalAccessRoleKind is the kind for a Global Access Role (a collection of Kubernetes policy rules)
// Dictates what access can will exist in what workspaces
const GlobalAccessRoleKind = "GlobalAccessRole"

// GlobalAccessRoleGVK is the GVK for an AccessRole
var GlobalAccessRoleGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    GlobalAccessRoleKind,
}

// NewGlobalAccessRole creates a new GlobalAccessRole with the specified name
func NewGlobalAccessRole(name, namespace string) *GlobalAccessRole {
	return &GlobalAccessRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalAccessRole represents a global set of Kubernetes RBAC rules that can use used for user access
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=globalaccessroles,scope=Cluster,categories={wayfinder}
type GlobalAccessRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the specification of the global access role
	// +kubebuilder:validation:Required
	Spec GlobalAccessRoleSpec `json:"spec,omitempty"`
	// Status is the status of the global access roles
	// +kubebuilder:validation:Optional
	Status GlobalAccessRoleStatus `json:"status,omitempty"`
}

func (GlobalAccessRole) APIPath() string {
	return "globalaccessroles"
}
func (c *GlobalAccessRole) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *GlobalAccessRole) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*GlobalAccessRole))
}
func (GlobalAccessRole) ListType() corev1alpha1.ObjectList {
	return &GlobalAccessRoleList{}
}

func (c GlobalAccessRole) GetAccessRoleSpec() AccessRoleSpec {
	return c.Spec.AccessRoleTemplate
}

// GlobalAccessRoleSpec defines the Kubernetes RBAC rules for user access to clusters across all workspaces
// Defines the rules for which workspaces will get cluster access role updates
// +k8s:openapi-gen=true
type GlobalAccessRoleSpec struct {
	// AccessRoleTemplate is the template for any AccessRole created
	// +kubebuilder:validation:Required
	AccessRoleTemplate AccessRoleSpec `json:"accessRoleTemplate,omitempty"`
	// WorkspaceSelector is a selector for Workspaces that will get this template
	// +kubebuilder:validation:Required
	WorkspaceSelector metav1.LabelSelector `json:"workspaceSelector,omitempty"`
}

// GlobalAccessRoleStatus defines the status of a GlobalAccessRole
// +k8s:openapi-gen=true
type GlobalAccessRoleStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// GetCommonStatus implements CommonStatusAware
func (c *GlobalAccessRole) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalAccessRoleList provides a list of global access roles
type GlobalAccessRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GlobalAccessRole `json:"items"`
}

func (GlobalAccessRoleList) ObjectType() corev1alpha1.Object {
	return &GlobalAccessRole{}
}
func (l *GlobalAccessRoleList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *GlobalAccessRoleList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*GlobalAccessRoleList))
}

func (l *GlobalAccessRoleList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *GlobalAccessRoleList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []GlobalAccessRole{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*GlobalAccessRole)))
	}
}
