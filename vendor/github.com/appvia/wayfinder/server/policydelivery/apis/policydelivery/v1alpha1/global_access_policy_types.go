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

// +kubebuilder:webhook:name=globalaccesspolicies.policydelivery.appvia.io,mutating=false,path=/validate/policydelivery.appvia.io/globalaccesspolicies,verbs=create;update;delete,groups="policydelivery.appvia.io",resources=globalaccesspolicies,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// GlobalAccessPolicyKind is the kind for an Access Policy Plan (templates for who could access a policy and where)
// - Allows us to ship built in, read only Access Policies for all workspaces
// - Allows platform admins to define a set of Access Policies for targeted workspaces
const GlobalAccessPolicyKind = "GlobalAccessPolicy"

// GlobalAccessPolicyGVK is the GVK for an AccessPolicy
var GlobalAccessPolicyGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    GlobalAccessPolicyKind,
}

// NewGlobalAccessPolicy creates a new AccessPolicy with the specified name and namespace
func NewGlobalAccessPolicy(name string) *GlobalAccessPolicy {
	return &GlobalAccessPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "",
		},
	}
}

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalAccessPolicy represents a Global Access Policy to define Shared Access Policies
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=globalaccesspolicies,scope=Cluster,categories={wayfinder}
type GlobalAccessPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the specification of the global access policy
	// +kubebuilder:validation:Required
	Spec GlobalAccessPolicySpec `json:"spec,omitempty"`
	// Status is the status of the global access policy
	// +kubebuilder:validation:Optional
	Status GlobalAccessPolicyStatus `json:"status,omitempty"`
}

func (GlobalAccessPolicy) APIPath() string {
	return "globalaccesspolicies"
}
func (c *GlobalAccessPolicy) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *GlobalAccessPolicy) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*GlobalAccessPolicy))
}
func (GlobalAccessPolicy) ListType() corev1alpha1.ObjectList {
	return &GlobalAccessPolicyList{}
}

func (c *GlobalAccessPolicy) GetAccessPolicySpec() AccessPolicySpec {
	return c.Spec.AccessPolicyTemplate
}

// GlobalAccessPolicySpec defines the desired state of a GlobalAccessPolicy
// +k8s:openapi-gen=true
type GlobalAccessPolicySpec struct {
	// AccessPolicyTemplate is the spec of a resulting AccessPolicy
	// +kubebuilder:validation:Required
	AccessPolicyTemplate AccessPolicySpec `json:"accessPolicyTemplate,omitempty"`
	// WorkspaceSelector is a selector for Workspaces that will get this template
	// +kubebuilder:validation:Required
	WorkspaceSelector metav1.LabelSelector `json:"workspaceSelector,omitempty"`
}

// GlobalAccessPolicyStatus defines the status of a GlobalAccessPolicy
// +k8s:openapi-gen=true
type GlobalAccessPolicyStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// GetCommonStatus implements CommonStatusAware
func (c *GlobalAccessPolicy) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalAccessPolicyList provides a list of global access policies
type GlobalAccessPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GlobalAccessPolicy `json:"items"`
}

func (GlobalAccessPolicyList) ObjectType() corev1alpha1.Object {
	return &GlobalAccessPolicy{}
}
func (l *GlobalAccessPolicyList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *GlobalAccessPolicyList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*GlobalAccessPolicyList))
}

func (l *GlobalAccessPolicyList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *GlobalAccessPolicyList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []GlobalAccessPolicy{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*GlobalAccessPolicy)))
	}
}
