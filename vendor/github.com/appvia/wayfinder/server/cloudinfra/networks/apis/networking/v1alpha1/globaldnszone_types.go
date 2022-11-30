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
)

// GlobalDNSZoneKind is the CRD resource kind
const GlobalDNSZoneKind = "GlobalDNSZone"

func GlobalDNSZoneGroupVersionKind() schema.GroupVersionKind {
	return schema.FromAPIVersionAndKind(GroupVersion.String(), GlobalDNSZoneKind)
}

// Check implementation:
var _ GlobalOrScopedDNSZone = &GlobalDNSZone{}

func NewGlobalDNSZone(name string) *GlobalDNSZone {
	return &GlobalDNSZone{
		TypeMeta: metav1.TypeMeta{
			Kind:       GlobalDNSZoneKind,
			APIVersion: GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// +kubebuilder:webhook:name=globaldnszones.networking.appvia.io,mutating=false,path=/validate/networking.appvia.io/globaldnszones,verbs=create;update;delete,groups="networking.appvia.io",resources=globaldnszone,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=globaldnszones.networking.appvia.io,mutating=true,path=/mutate/networking.appvia.io/globaldnszones,verbs=create;update,groups="networking.appvia.io",resources=globaldnszone,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalDNSZone represents a DNS zone in a specific cloud provider DNS implementation
// cloud account, available system-wide
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=globaldnszone,scope=Cluster,categories={wayfinder}
type GlobalDNSZone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DNSZoneSpec   `json:"spec,omitempty"`
	Status DNSZoneStatus `json:"status,omitempty"`
}

func (GlobalDNSZone) APIPath() string {
	return "globaldnszones"
}
func (c *GlobalDNSZone) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *GlobalDNSZone) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*GlobalDNSZone))
}
func (GlobalDNSZone) ListType() corev1alpha1.ObjectList {
	return &GlobalDNSZoneList{}
}

// IsGlobal implements GlobalOrScopedDNSZone
func (c *GlobalDNSZone) IsGlobal() bool {
	return true
}

// GetZoneSpec implements GlobalOrScopedDNSZone
func (c *GlobalDNSZone) GetZoneSpec() *DNSZoneSpec {
	return &c.Spec
}

// GetZoneStatus implements GlobalOrScopedDNSZone
func (c *GlobalDNSZone) GetZoneStatus() *DNSZoneStatus {
	return &c.Status
}

// GetRef implements GlobalOrScopedDNSZone
func (c *GlobalDNSZone) GetRef() *DNSZoneRef {
	return &DNSZoneRef{
		Namespace: "",
		Name:      c.Name,
	}
}

// GetCommonStatus implements CommonStatusAware
func (c *GlobalDNSZone) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// GetStatus returns the current overall status
func (c GlobalDNSZone) GetStatus() corev1alpha1.Status {
	return c.Status.Status
}

// AllocatedToWorkspace implements the Allocatable interface for GlobalDNSZone, it will return true
// if the specified workspace is allowed to use this GlobalDNSZone according to the allocation
// settings on the spec.
func (c *GlobalDNSZone) AllocatedToWorkspace(ws corev1alpha1.WorkspaceKey) bool {
	return c.Spec.Allocation != nil && c.Spec.Allocation.Allowed(c.Namespace, ws)
}

// CloudResourceName returns the DNS provider name for this DNS zone
func (c *GlobalDNSZone) CloudResourceName() string {
	if c.Spec.CloudResourceName != "" {
		return c.Spec.CloudResourceName
	}
	return fmt.Sprintf("global-%s", c.Name)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DNSZoneList provides a list of workload identities
type GlobalDNSZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GlobalDNSZone `json:"items"`
}

func (GlobalDNSZoneList) ObjectType() corev1alpha1.Object {
	return &GlobalDNSZone{}
}
func (l *GlobalDNSZoneList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *GlobalDNSZoneList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*GlobalDNSZoneList))
}

func (l *GlobalDNSZoneList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *GlobalDNSZoneList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []GlobalDNSZone{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*GlobalDNSZone)))
	}
}
