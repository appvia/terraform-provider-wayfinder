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

// +kubebuilder:webhook:name=workspaceinvitations.org.appvia.io,mutating=false,path=/validate/org.appvia.io/workspaceinvitations,verbs=create;update;delete,groups="org.appvia.io",resources=workspaceinvitations,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=workspaceinvitations.org.appvia.io,mutating=true,path=/mutate/org.appvia.io/workspaceinvitations,verbs=create;update,groups="org.appvia.io",resources=workspaceinvitations,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// WorkspaceInvitationKind is the kind for a workspace invitation CRD
const WorkspaceInvitationKind = "WorkspaceInvitation"

// WorkspaceInvitationGVK is the GVK for an invite
var WorkspaceInvitationGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    WorkspaceInvitationKind,
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceInvitation is the Schema for the workspace invitation API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=workspaceinvitations,scope=Namespaced,categories={wayfinder},shortName=invite
type WorkspaceInvitation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkspaceInvitationSpec   `json:"spec,omitempty"`
	Status WorkspaceInvitationStatus `json:"status,omitempty"`
}

func (WorkspaceInvitation) GroupVersionKind() schema.GroupVersionKind {
	return WorkspaceInvitationGVK
}
func (WorkspaceInvitation) APIPath() string {
	return "workspaceinvitations"
}
func (c *WorkspaceInvitation) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *WorkspaceInvitation) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*WorkspaceInvitation))
}
func (WorkspaceInvitation) ListType() corev1alpha1.ObjectList {
	return &WorkspaceInvitationList{}
}
func (*WorkspaceInvitation) GetCommonStatus() *corev1alpha1.CommonStatus {
	// No status on an invite, so nothing to return
	return nil
}

// WorkspaceInvitationSpec defines the desired state of a workspace invitation
// +k8s:openapi-gen=true
type WorkspaceInvitationSpec struct {
	// Username is the user being bound to the workspace
	// +kubebuilder:validation:Required
	Username string `json:"username"`
	// Workspace is the name of the workspace being invitied to
	// +kubebuilder:validation:Required
	Workspace corev1alpha1.WorkspaceKey `json:"workspace"`
}

// WorkspaceInvitationStatus defines the observed state of a workspace invite
// +k8s:openapi-gen=true
type WorkspaceInvitationStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// WorkspaceInvitationList contains a list of Workspace Invitations
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkspaceInvitationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkspaceInvitation `json:"items"`
}

func (WorkspaceInvitationList) ObjectType() corev1alpha1.Object {
	return &WorkspaceInvitation{}
}
func (l *WorkspaceInvitationList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *WorkspaceInvitationList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*WorkspaceInvitationList))
}

func (l *WorkspaceInvitationList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *WorkspaceInvitationList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []WorkspaceInvitation{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*WorkspaceInvitation)))
	}
}
