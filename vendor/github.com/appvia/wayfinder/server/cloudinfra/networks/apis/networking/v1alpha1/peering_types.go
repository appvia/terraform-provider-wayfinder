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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	cloudaccessv1alpha1 "github.com/appvia/wayfinder/server/cloudaccess/apis/cloudaccess/v1alpha1"
)

// +kubebuilder:webhook:name=peerings.networking.appvia.io,mutating=false,path=/validate/networking.appvia.io/peerings,verbs=create;update,groups="networking.appvia.io",resources=peering,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// PeeringKind is the kind for a Peering
const PeeringKind = "Peering"

const (
	// PeeringRuleAnnotation is added to peering connections to track which rule
	// created the connection
	PeeringRuleAnnotation = "peering.appvia.io/rule"
)

// PeeringGVK is the GVK for a Peering
var PeeringGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    PeeringKind,
}

// NewPeering is called to return an empty peering
func NewPeering() *Peering {
	return &Peering{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       PeeringKind,
		},
	}
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Peering is tracking resource which is used to create a connection between the
// source network and an external network
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=peerings,scope=Namespaced,categories={wayfinder}
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
type Peering struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PeeringSpec   `json:"spec,omitempty"`
	Status PeeringStatus `json:"status,omitempty"`
}

func (Peering) APIPath() string {
	return "peerings"
}
func (p *Peering) Clone() corev1alpha1.Object {
	return p.DeepCopy()
}
func (p *Peering) CloneInto(o corev1alpha1.Object) {
	p.DeepCopyInto(o.(*Peering))
}
func (Peering) ListType() corev1alpha1.ObjectList {
	return &PeeringList{}
}

// PeeringSpec defines the definition for the peering connection resource
type PeeringSpec struct {
	// CloudAccount provides an optional reference to a cloudaccount which has the
	// permission to carry out the tasks required to fulfil peering or gateway
	// attachments
	// +kubebuilder:validation:Optional
	CloudAccount *cloudaccessv1alpha1.CloudAccountReference `json:"cloudAccount,omitempty"`
	// Connection provides the policy details around how the peering should
	// be achieved
	// +kubebuilder:validation:Required
	Connection PeeringRuleConnection `json:"connection"`
	// NetworkRef is the network which we are creating a connection from
	// +kubebuilder:validation:Required
	NetworkRef corev1alpha1.Ownership `json:"networkRef"`
}

// PeeringStatus defines the observed state of a peering setup
// +k8s:openapi-gen=true
type PeeringStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// PeeringID is the cloud agnostic peering identifier
	PeeringID string `json:"peeringID,omitempty"`
}

// CloudResourceName returns the cloud resource name
func (p *Peering) CloudResourceName() string {
	return fmt.Sprintf("%s-%s", p.Namespace, p.Name)
}

// GetCommonStatus implements CommonStatusAware for CloudAccount
func (p *Peering) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &p.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PeeringList provides a list of resources
type PeeringList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Peering `json:"items"`
}

func (PeeringList) ObjectType() corev1alpha1.Object {
	return &Peering{}
}
func (l *PeeringList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *PeeringList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*PeeringList))
}

// GetItems returns the items
func (l *PeeringList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

// SetItems set them
func (l *PeeringList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []Peering{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*Peering)))
	}
}
