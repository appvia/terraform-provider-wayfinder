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

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// AuditEventSpec defines the desired state of User
// +k8s:openapi-gen=false
type AuditEventSpec struct {
	// ID is the unique identifier of this audit event.
	ID int `json:"id,omitempty"`
	// CreatedAt is the timestamp of record creation
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
	// Resource is the area of the API accessed in this audit operation (e.g. workspaces, etc).
	Resource string `json:"resource,omitempty"`
	// ResourceURI is the identifier of the resource in question.
	ResourceURI string `json:"resourceURI,omitempty"`
	// APIVersion is the version of the API used for this operation.
	APIVersion string `json:"apiVersion,omitempty"`
	// Verb is the type of action performed (e.g. PUT, GET, etc)
	Verb string `json:"verb,omitempty"`
	// Operation is the operation performed (e.g. UpdateCluster, CreateCluster, etc).
	Operation string `json:"operation,omitempty"`
	// Workspace is the workspace whom event may be associated to
	Workspace corev1alpha1.WorkspaceKey `json:"workspace,omitempty"`
	// User is the user which the event is related
	User string `json:"user,omitempty"`
	// StartedAt is the timestamp the operation was initiated
	StartedAt metav1.Time `json:"startedAt,omitempty"`
	// CompletedAt is the timestamp the operation completed
	CompletedAt metav1.Time `json:"completedAt,omitempty"`
	// ResponseCode indicates the HTTP status code of the operation (e.g. 200, 404, etc).
	ResponseCode int `json:"responseCode,omitempty"`
	// Message is event message itself
	Message string `json:"message,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AuditEvent is the Schema for the audit API
// +k8s:openapi-gen=false
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=auditevents
type AuditEvent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AuditEventSpec `json:"spec,omitempty"`
}

func (AuditEvent) APIPath() string {
	return "auditevents"
}
func (c *AuditEvent) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *AuditEvent) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*AuditEvent))
}
func (AuditEvent) ListType() corev1alpha1.ObjectList {
	return &AuditEventList{}
}

func (AuditEvent) GetCommonStatus() *corev1alpha1.CommonStatus {
	return nil
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AuditEventList contains a list of audit event
type AuditEventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuditEvent `json:"items"`
}

func (AuditEventList) ObjectType() corev1alpha1.Object {
	return &AuditEvent{}
}
func (l *AuditEventList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *AuditEventList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*AuditEventList))
}

func (l *AuditEventList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *AuditEventList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []AuditEvent{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*AuditEvent)))
	}
}
