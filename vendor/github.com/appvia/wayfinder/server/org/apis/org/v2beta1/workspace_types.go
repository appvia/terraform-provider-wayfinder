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

// +kubebuilder:webhook:name=workspaces.org.appvia.io,mutating=false,path=/validate/org.appvia.io/workspaces,verbs=create;update;delete,groups="org.appvia.io",resources=workspaces,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=workspaces.org.appvia.io,mutating=true,path=/mutate/org.appvia.io/workspaces,verbs=create;update,groups="org.appvia.io",resources=workspaces,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// WorkspaceKind is the kind for a workspace CRD
const WorkspaceKind = "Workspace"

// WorkspaceGVK is the GVK for a workspace
var WorkspaceGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    WorkspaceKind,
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Workspace is the Schema for the workspace API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=workspaces,scope=Cluster,categories={wayfinder},shortName=ws
type Workspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkspaceSpec   `json:"spec,omitempty"`
	Status WorkspaceStatus `json:"status,omitempty"`
}

func (Workspace) GroupVersionKind() schema.GroupVersionKind {
	return WorkspaceGVK
}
func (Workspace) APIPath() string {
	return "workspaces"
}
func (t *Workspace) Clone() corev1alpha1.Object {
	return t.DeepCopy()
}
func (t *Workspace) CloneInto(o corev1alpha1.Object) {
	t.DeepCopyInto(o.(*Workspace))
}
func (Workspace) ListType() corev1alpha1.ObjectList {
	return &WorkspaceList{}
}

// GetCommonStatus implements CommonStatusAware
func (t *Workspace) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &t.Status.CommonStatus
}

// Key returns the unique key for this workspace
func (t Workspace) Key() corev1alpha1.WorkspaceKey {
	return corev1alpha1.WorkspaceKey(t.ObjectMeta.Name)
}

// WorkspaceType identifies the type of a workspace
// +kubebuilder:validation:Enum=standard;infrastructure;admin
type WorkspaceType string

const (
	// WorkspaceTypeStandard is the default workspace type for normal, application-team-oriented
	// workspaces
	WorkspaceTypeStandard WorkspaceType = "standard"
	// WorkspaceTypeInfrastructure identifies that this workspace is a container for infrastructure,
	// for example to contain multi-tenant clusters allocated to multiple standard workspaces
	WorkspaceTypeInfrastructure WorkspaceType = "infrastructure"
	// WorkspaceTypeAdmin is for internal, Wayfinder managed workspaces used by the system. These
	// can only be created by Wayfinder itself.
	WorkspaceTypeAdmin WorkspaceType = "admin"
)

// WorkspaceSpec defines the desired state of workspace
// +k8s:openapi-gen=true
type WorkspaceSpec struct {
	// Key is the unique identifier for this workspace
	// +kubebuilder:validation:Required
	Key corev1alpha1.WorkspaceKey `json:"key"`
	// Type is an optional type for this workspace. If unspecified, 'standard' is assumed.
	// +kubebuilder:validtion:Optional
	Type WorkspaceType `json:"type,omitempty"`
	// Summary is a short human-readable name for this workspace
	// +kubebuilder:validation:Required
	Summary string `json:"summary"`
	// Description is an extended description for the workspace
	// +kubebuilder:validation:Optional
	Description string `json:"description"`
	// ResourceNamespace will be set by Wayfinder on creation and cannot be modified or defined by
	// users. This is the namespace within Wayfinder in which this workspace's resources will exist.
	// +kubebuilder:validation:Required
	ResourceNamespace string `json:"resourceNamespace"`
	// Owners is the set of users who own this workspace (i.e. can control access to the
	// workspace and its resources). If unpopulated on creation, Wayfinder will set this to the
	// user who created the workspace.
	// +kubebuilder:validation:Optional
	Owners []Subject `json:"owners,omitempty"`
}

// WorkspaceStatus defines the observed state of workspace
// +k8s:openapi-gen=true
type WorkspaceStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// WorkspaceList contains a list of workspaces
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkspaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workspace `json:"items"`
}

func (WorkspaceList) ObjectType() corev1alpha1.Object {
	return &Workspace{}
}
func (l *WorkspaceList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *WorkspaceList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*WorkspaceList))
}

func (l *WorkspaceList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *WorkspaceList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []Workspace{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*Workspace)))
	}
}
