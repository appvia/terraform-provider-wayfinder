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

// AssignableNetworkKind is the kind for assignable networks
const AssignableNetworkKind = "Peering"

// AssignableNetworkGVK is the GVK for a the assignable networks
var AssignableNetworkGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    AssignableNetworkKind,
}

// NewAssignableNetwork is called to return an empty resource
func NewAssignableNetwork() *AssignableNetwork {
	return &AssignableNetwork{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       AssignableNetworkKind,
		},
	}
}

// AssignableNetworkType represents the network type of an assignable network
// +kubebuilder:validation:Enum=node;pod;service
type AssignableNetworkType string

const (
	// NodeNetwork defines node network - the range provided to the pools
	NodeNetwork AssignableNetworkType = "node"
	// PodsNetwork defines the range for pods
	PodsNetwork AssignableNetworkType = "pod"
	// ServicesNetwork defines the service / cluster range
	ServicesNetwork AssignableNetworkType = "service"
)

var (
	// AssignableNetworkTypes is list of supported types
	AssignableNetworkTypes = []AssignableNetworkType{
		NodeNetwork,
		PodsNetwork,
		ServicesNetwork,
	}
)

// AssignableNetworkRange defines a assignable network range
type AssignableNetworkRange struct {
	// DefaultMask is the default block to assign from the range
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Min=1
	DefaultMask int `json:"defaultMask"`
	// MinMask defines the *largest* size of network that can be allocated (in CIDR notation).
	// The smaller this number, the fewer bits of the network address are reserved, and the
	// larger a network allocation can be. Typical values would be /21, /22 or /23.
	// +kubebuilder:validation:Required
	MinMask int `json:"minMask"`
	// MaxMask defined the *smallest* size of network that can be allocated (in CIDR notation).
	// The larger this number, the smaller a network allocation can be created. Typical values would be
	// /24 or /25.
	// +kubebuilder:validation:Required
	MaxMask int `json:"maxMask"`
	// Range is the CIDR range of the network
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=9
	Range string `json:"range"`
	// Type is the network type being defined - i.e. pods, clusters, services or node
	// +kubebuilder:validation:Required
	Type AssignableNetworkType `json:"type"`
}

// AssignableNetworkSpec define the definitions for network ranges
type AssignableNetworkSpec struct {
	// Providers the provider the range is assigned
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=3
	Provider string `json:"provider"`
	// ExcludeWorkspaces is a collection of workspaces whom are excluded from the requirement.
	// +kubebuilder:validation:Optional
	ExcludeWorkspaces []string `json:"excludeWorkspaces,omitempty"`
	// IncludeWorkspaces is a collection of workspaces who are included as part of the rule - if
	// specified the requirement is only applied to those workspaces - by default we assume
	// this as a wildcard all workspaces
	// +kubebuilder:validation:Optional
	IncludeWorkspaces []string `json:"includeWorkspaces,omitempty"`
	// Networks is a collection of network assignment for a particular provider
	// +kubebuilder:validation:Optional
	Networks []*AssignableNetworkRange `json:"networks,omitempty"`
	// Plans is a optional list of plans to associate the range to
	// +kubebuilder:validation:Optional
	Plans []string `json:"plans,omitempty"`
}

// GetNetworkRangeByType returns all the ranges by type
func (a *AssignableNetworkSpec) GetNetworkRangeByType(nType AssignableNetworkType) []*AssignableNetworkRange {
	var list []*AssignableNetworkRange

	for i := 0; i < len(a.Networks); i++ {
		if a.Networks[i].Type == nType {
			list = append(list, a.Networks[i])
		}
	}

	return list
}

// AssignableNetworkStatus defines the observed state of status on a policy
// +k8s:openapi-gen=true
type AssignableNetworkStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// +kubebuilder:webhook:name=assignablenetworks.networking.appvia.io,mutating=false,path=/validate/networking.appvia.io/assignablenetworks,verbs=create;update,groups="networking.appvia.io",resources=assignablenetworks,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AssignableNetwork is the definition for an assignable network range
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={wayfinder}
type AssignableNetwork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AssignableNetworkSpec   `json:"spec,omitempty"`
	Status AssignableNetworkStatus `json:"status,omitempty"`
}

func (AssignableNetwork) APIPath() string {
	return "assignablenetworks"
}
func (a *AssignableNetwork) Clone() corev1alpha1.Object {
	return a.DeepCopy()
}
func (a *AssignableNetwork) CloneInto(o corev1alpha1.Object) {
	a.DeepCopyInto(o.(*AssignableNetwork))
}
func (AssignableNetwork) ListType() corev1alpha1.ObjectList {
	return &AssignableNetworkList{}
}

func (a *AssignableNetwork) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &a.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AssignableNetworkList contains a list of policies
type AssignableNetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AssignableNetwork `json:"items"`
}

func (AssignableNetworkList) ObjectType() corev1alpha1.Object {
	return &AssignableNetwork{}
}
func (l *AssignableNetworkList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *AssignableNetworkList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*AssignableNetworkList))
}

// GetItems returns the items
func (l *AssignableNetworkList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

// SetItems sets the items
func (l *AssignableNetworkList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []AssignableNetwork{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*AssignableNetwork)))
	}
}
