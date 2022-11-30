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

// +kubebuilder:webhook:name=wayfindergroups.org.appvia.io,mutating=false,path=/validate/org.appvia.io/wayfindergroups,verbs=create;update;delete,groups="org.appvia.io",resources=wayfindergroups,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=wayfindergroups.org.appvia.io,mutating=true,path=/mutate/org.appvia.io/wayfindergroups,verbs=create;update,groups="org.appvia.io",resources=wayfindergroups,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// WayfinderGroupKind is the kind for a Wayfinder group (a collection of Wayfinder users)
const WayfinderGroupKind = "WayfinderGroup"

// WayfinderGroupGVK is the GVK for a WayfinderGroup
var WayfinderGroupGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    WayfinderGroupKind,
}

// NewWayfinderGroup creates a new WayfinderGroup with the specified name
func NewWayfinderGroup(name string) *WayfinderGroup {
	return &WayfinderGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WayfinderGroup represents a collection of Wayfinder users
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=wayfindergroups,scope=Cluster,categories={wayfinder},shortName=wfgroup
type WayfinderGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the specification of the group
	// +kubebuilder:validation:Required
	Spec GroupSpec `json:"spec,omitempty"`
	// Status is the status of the group
	// +kubebuilder:validation:Optional
	Status WayfinderGroupStatus `json:"status,omitempty"`
}

func (WayfinderGroup) GroupVersionKind() schema.GroupVersionKind {
	return WayfinderGroupGVK
}

func (WayfinderGroup) APIPath() string {
	return "wayfindergroups"
}
func (g *WayfinderGroup) Clone() corev1alpha1.Object {
	return g.DeepCopy()
}
func (g *WayfinderGroup) CloneInto(o corev1alpha1.Object) {
	g.DeepCopyInto(o.(*WayfinderGroup))
}
func (WayfinderGroup) ListType() corev1alpha1.ObjectList {
	return &WayfinderGroupList{}
}
func (g *WayfinderGroup) HasUserName(name string) bool {
	for _, u := range g.Spec.Members {
		if u.Kind == SubjectKindUser && u.Name == name {
			return true
		}
	}
	return false
}
func (g *WayfinderGroup) ToSubject() Subject {
	return Subject{
		Kind: SubjectKindWayfinderGroup,
		Name: g.Name,
	}
}

// GroupSpec defines the specification for a group
// +k8s:openapi-gen=true
type GroupSpec struct {
	// Description provides a human-readable description of this group
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Description string `json:"description"`
	// Members are the members of the group
	// +kubebuilder:validation:Optional
	Members []Subject `json:"members,omitempty"`
}

// HasMember returns true if the specified name is present in the subjects. Will not find
// workspace-scoped subjects (e.g. WorkspaceAccessToken) - use HasWorkspaceScopedMember.
func (g GroupSpec) HasMember(kind SubjectKind, name string) bool {
	return g.HasSubject(Subject{Kind: kind, Name: name, Workspace: ""})
}

// HasSubject returns true if the group contains the subject
func (g GroupSpec) HasSubject(subject Subject) bool {
	for _, s := range g.Members {
		if s.Equals(subject) {
			return true
		}
	}

	return false
}

// HasWorkspaceScopedMember returns true if the specified name is present in the subjects with the
// specified workspace key. Suitable ONLY for workspace-scoped subjects (e.g. WorkspaceAccessToken) - use
// HasMember for non-workspace-scoped subjects.
func (g GroupSpec) HasWorkspaceScopedMember(kind SubjectKind, name string, ws corev1alpha1.WorkspaceKey) bool {
	return g.HasSubject(Subject{Kind: kind, Name: name, Workspace: ws})
}

// WayfinderGroupStatus defines the status of a group
// +k8s:openapi-gen=true
type WayfinderGroupStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// GetCommonStatus implements CommonStatusAware
func (g *WayfinderGroup) GetCommonStatus() *corev1alpha1.CommonStatus {
	g.Status.SetStatusIfNotSet(corev1alpha1.SuccessStatus)
	return &g.Status.CommonStatus
}

// WayfinderGroupList provides a list of WayfinderGroups
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WayfinderGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WayfinderGroup `json:"items"`
}

func (WayfinderGroupList) ObjectType() corev1alpha1.Object {
	return &WayfinderGroup{}
}
func (l *WayfinderGroupList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *WayfinderGroupList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*WayfinderGroupList))
}

func (l *WayfinderGroupList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *WayfinderGroupList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []WayfinderGroup{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*WayfinderGroup)))
	}
}
