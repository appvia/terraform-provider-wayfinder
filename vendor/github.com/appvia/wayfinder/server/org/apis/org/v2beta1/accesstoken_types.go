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

// +kubebuilder:webhook:name=workspaceaccesstokens.org.appvia.io,mutating=false,path=/validate/org.appvia.io/workspaceaccesstokens,verbs=create;update;delete,groups="org.appvia.io",resources=workspaceaccesstokens,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=platformaccesstokens.org.appvia.io,mutating=false,path=/validate/org.appvia.io/platformaccesstokens,verbs=create;update;delete,groups="org.appvia.io",resources=platformaccesstokens,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=workspaceaccesstokens.org.appvia.io,mutating=true,path=/mutate/org.appvia.io/workspaceaccesstokens,verbs=create;update,groups="org.appvia.io",resources=workspaceaccesstokens,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1
// +kubebuilder:webhook:name=platformaccesstokens.org.appvia.io,mutating=true,path=/mutate/org.appvia.io/platformaccesstokens,verbs=create;update,groups="org.appvia.io",resources=platformaccesstokens,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// WorkspaceAccessTokenKind is the workspace access token crd name
const WorkspaceAccessTokenKind = "WorkspaceAccessToken"

// PlatformAccessTokenKind is the platform access token crd name
const PlatformAccessTokenKind = "PlatformAccessToken"

// WorkspaceAccessTokenGVK is the GVK for workspace access tokens
var WorkspaceAccessTokenGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    WorkspaceAccessTokenKind,
}

// PlatformAccessTokenGVK is the GVK for wayfinder global access tokens
var PlatformAccessTokenGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    PlatformAccessTokenKind,
}

// +k8s:openapi-gen=false
// +kubebuilder:object:generate=false
type PlatformOrWorkspaceAccessToken interface {
	corev1alpha1.Object
	GetAccessTokenSpec() *AccessTokenSpec
	GetAccessTokenStatus() *AccessTokenStatus
}

// IsPlatformToken returns true if the provided value represents a platform token
func IsPlatformToken(t PlatformOrWorkspaceAccessToken) bool {
	return t.GetNamespace() == ""
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceAccessToken is an access token can automate Wayfinder features solely within a specified
// workspace.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=workspaceaccesstokens,scope=Namespaced,categories={wayfinder}
type WorkspaceAccessToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccessTokenSpec   `json:"spec,omitempty"`
	Status AccessTokenStatus `json:"status,omitempty"`
}

func (WorkspaceAccessToken) GroupVersionKind() schema.GroupVersionKind {
	return WorkspaceAccessTokenGVK
}

func (WorkspaceAccessToken) APIPath() string {
	return "workspaceaccesstokens"
}
func (r *WorkspaceAccessToken) Clone() corev1alpha1.Object {
	return r.DeepCopy()
}
func (r *WorkspaceAccessToken) CloneInto(o corev1alpha1.Object) {
	r.DeepCopyInto(o.(*WorkspaceAccessToken))
}
func (WorkspaceAccessToken) ListType() corev1alpha1.ObjectList {
	return &WorkspaceAccessTokenList{}
}

func (r *WorkspaceAccessToken) GetCommonStatus() *corev1alpha1.CommonStatus {
	r.Status.SetStatusIfNotSet(corev1alpha1.SuccessStatus)
	return &r.Status.CommonStatus
}

func (r *WorkspaceAccessToken) ToSubject() Subject {
	return Subject{
		Kind:      SubjectKindWorkspaceAccessToken,
		Name:      r.Name,
		Workspace: corev1alpha1.Workspace(r),
	}
}

func (r *WorkspaceAccessToken) GetAccessTokenSpec() *AccessTokenSpec {
	return &r.Spec
}
func (r *WorkspaceAccessToken) GetAccessTokenStatus() *AccessTokenStatus {
	return &r.Status
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PlatformAccessToken is an access token able to be used to automate all Wayfinder features, both
// inside and outside of workspaces.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=platformaccesstokens,scope=Cluster,categories={wayfinder}
type PlatformAccessToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccessTokenSpec   `json:"spec,omitempty"`
	Status AccessTokenStatus `json:"status,omitempty"`
}

func (PlatformAccessToken) GroupVersionKind() schema.GroupVersionKind {
	return PlatformAccessTokenGVK
}

func (PlatformAccessToken) APIPath() string {
	return "platformaccesstokens"
}
func (r *PlatformAccessToken) Clone() corev1alpha1.Object {
	return r.DeepCopy()
}
func (r *PlatformAccessToken) CloneInto(o corev1alpha1.Object) {
	r.DeepCopyInto(o.(*PlatformAccessToken))
}
func (PlatformAccessToken) ListType() corev1alpha1.ObjectList {
	return &PlatformAccessTokenList{}
}

func (r *PlatformAccessToken) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &r.Status.CommonStatus
}

func (r *PlatformAccessToken) ToSubject() Subject {
	return Subject{
		Kind: SubjectKindPlaformAccessToken,
		Name: r.Name,
	}
}

func (r *PlatformAccessToken) GetAccessTokenSpec() *AccessTokenSpec {
	return &r.Spec
}

func (r *PlatformAccessToken) GetAccessTokenStatus() *AccessTokenStatus {
	return &r.Status
}

// AccessTokenSpec defines the desired state of policy
type AccessTokenSpec struct {
	// Description provides a short summary on the use of the access token
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
	// TokenIDs is the set of signed token IDs which are currently valid for this access token. This
	// will be populated by Wayfinder, always leave blank on definition.
	// +kubebuilder:validation:Optional
	TokenIDs []string `json:"tokenIDs,omitempty"`
}

// AccessTokenStatus defines the observed state of status on a policy
// +k8s:openapi-gen=true
type AccessTokenStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// WorkspaceAccessTokenList contains a list of workspace-scoped access tokens
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkspaceAccessTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkspaceAccessToken `json:"items"`
}

func (WorkspaceAccessTokenList) ObjectType() corev1alpha1.Object {
	return &WorkspaceAccessToken{}
}
func (l *WorkspaceAccessTokenList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *WorkspaceAccessTokenList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*WorkspaceAccessTokenList))
}

func (l *WorkspaceAccessTokenList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *WorkspaceAccessTokenList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []WorkspaceAccessToken{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*WorkspaceAccessToken)))
	}
}

// PlatformAccessTokenList contains a list of platform-scoped access tokens
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PlatformAccessTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlatformAccessToken `json:"items"`
}

func (PlatformAccessTokenList) ObjectType() corev1alpha1.Object {
	return &PlatformAccessToken{}
}
func (l *PlatformAccessTokenList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *PlatformAccessTokenList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*PlatformAccessTokenList))
}

func (l *PlatformAccessTokenList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *PlatformAccessTokenList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []PlatformAccessToken{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*PlatformAccessToken)))
	}
}
