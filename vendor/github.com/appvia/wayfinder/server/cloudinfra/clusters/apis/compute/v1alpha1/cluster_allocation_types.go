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

// ClusterAllocationKind is the kind for a ClusterAllocation
const ClusterAllocationKind = "ClusterAllocation"

// ClusterAllocationGVK is the GVK for a Cluster
var ClusterAllocationGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    ClusterAllocationKind,
}

// AnnotationClusterAllocation should be set ("true") on a cluster that is returned from a ClusterAllocation
// to signal that it is not in fact the immediate parent object to anything referencing it (e.g. a
// namespaceclaim)
const AnnotationClusterAllocation = "appvia.io/allocated"

// +kubebuilder:webhook:name=clusterallocations.compute.appvia.io,mutating=false,path=/validate/compute.appvia.io/clusterallocations,verbs=create;update;delete,groups="compute.appvia.io",resources=clusterallocations,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterAllocation is the schema for cluster allocationn definitions in Wayfinder
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=clusterallocations,scope=Namespaced,categories={wayfinder}
type ClusterAllocation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterAllocationSpec   `json:"spec,omitempty"`
	Status ClusterAllocationStatus `json:"status,omitempty"`
}

// ClusterAllocationStatus defines the observed state of a cluster allocation
// +k8s:openapi-gen=true
type ClusterAllocationStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// AuthProxyEndpoint is the endpoint of the authentication proxy for this cluster
	// +kubebuilder:validation:Optional
	AuthProxyEndpoint string `json:"authProxyEndpoint,omitempty"`
	// AuthProxyCertificate is the certificate of the auth proxy endpoint
	// +kubebuilder:validation:Optional
	AuthProxyCertificate string `json:"authProxyCertificate,omitempty"`
	// CloudAccountRef is a reference to the cloud account to use to retrieve credentials for
	// this cluster. Will be populated if the spec specifies a CloudAccount (org or shared) as
	// the credential object.
	// +kubebuilder:validation:Optional
	CloudAccountRef *cloudaccessv1alpha1.CloudAccountReference `json:"cloudAccount,omitempty"`
	// Provider refers to the cluster type (e.g. AKS, GKE, EKS)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Provider string `json:"provider"`
	// Plan refers to the original plan the cluster was created from
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Plan string `json:"plan,omitempty"`
	// Stage is the name of the stage for the cluster allocated
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Stage string `json:"stage,omitempty"`
	// QuotaLimitTemplatesNames provides a list of teamplates that can be used
	// The value is placed on a namespaceclaim to indicate the desired constraints / size
	QuotaLimitTemplateNames []string `json:"quotalimitTemplateNames,omitempty"`
	// QuotaLimitDefault is the name of the default used if no name is specified
	QuotaLimitDefault string `json:"quotalimitDefault,omitempty"`
	// PodSecurityStandard describes what PSS restrictions exist on this cluster
	PodSecurityStandard *PodSecurityStandard `json:"podSecurityStandard,omitempty"`
}

// ClusterAllocationSpec defines the desired state of a cluster allocation
// +k8s:openapi-gen=true
type ClusterAllocationSpec struct {
	// Cluster is the cluster the clusterallocation references
	// +kubebuilder:validation:Required
	Cluster corev1alpha1.Ownership `json:"cluster"`
}

// GetCommonStatus implements CommonStatusAware for ClusterAllocatiion
func (c *ClusterAllocation) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

func (ClusterAllocation) APIPath() string {
	return "clusterallocations"
}
func (c *ClusterAllocation) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *ClusterAllocation) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*ClusterAllocation))
}
func (ClusterAllocation) ListType() corev1alpha1.ObjectList {
	return &ClusterAllocationList{}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterAllocationList contains a list of cluster allocations
type ClusterAllocationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterAllocation `json:"items"`
}

func (ClusterAllocationList) ObjectType() corev1alpha1.Object {
	return &ClusterAllocation{}
}
func (l *ClusterAllocationList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *ClusterAllocationList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*ClusterAllocationList))
}

// Owner implements the Ownable interface
func (c ClusterAllocation) Owner() corev1alpha1.Ownership {
	return c.Spec.Cluster
}

// Ownership returns a ownership resource from ClusterAllocation definition
func (c *ClusterAllocation) Ownership() corev1alpha1.Ownership {
	return corev1alpha1.Ownership{
		Group:     GroupVersion.Group,
		Version:   GroupVersion.Version,
		Kind:      "ClusterAllocation",
		Namespace: c.Namespace,
		Name:      c.Name,
	}
}

// AsCluster converts a clusterallocation object to a cluster for listing etc.
func (c ClusterAllocation) AsCluster(tenantNamespace bool) Cluster {
	cluster := Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       c.Spec.Cluster.Kind,
			APIVersion: fmt.Sprintf("%s/%s", c.Spec.Cluster.Group, c.Spec.Cluster.Version),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              c.Name,
			Namespace:         c.Spec.Cluster.Namespace,
			CreationTimestamp: c.ObjectMeta.CreationTimestamp,
		},
		Spec: ClusterSpec{
			Plan:     c.Status.Plan,
			Provider: c.Status.Provider,
		},
		Status: ClusterStatus{
			AuthProxyEndpoint:    c.Status.AuthProxyEndpoint,
			AuthProxyCertificate: c.Status.AuthProxyCertificate,
			CommonStatus: corev1alpha1.CommonStatus{
				Status:     c.Status.Status,
				Conditions: c.Status.Conditions,
			},
		},
	}
	if tenantNamespace {
		cluster.Namespace = c.Namespace
	}
	return cluster
}

func (l *ClusterAllocationList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *ClusterAllocationList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []ClusterAllocation{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*ClusterAllocation)))
	}
}
