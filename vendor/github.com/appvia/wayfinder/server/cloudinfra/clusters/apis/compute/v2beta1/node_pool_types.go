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
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// NodePoolKind is the kind for a NodePool
const NodePoolKind = "NodePool"

// NodePoolGVK is the GVK for a Cluster node pool
var NodePoolGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    NodePoolKind,
}

// +kubebuilder:webhook:name=nodepools.compute.appvia.io,mutating=false,path=/validate/compute.appvia.io/nodepools,verbs=create;update;delete,groups="compute.appvia.io",resources=nodepools,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=nodepools.compute.appvia.io,mutating=true,path=/mutate/compute.appvia.io/nodepools,verbs=create;update,groups="compute.appvia.io",resources=nodepools,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Nodepool is the schema for cluster nodepool definitions in Wayfinder
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodepools,scope=Namespaced,categories={wayfinder}
type NodePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodePoolSpec   `json:"spec,omitempty"`
	Status NodePoolStatus `json:"status,omitempty"`
}

// NodePool defines a node pool in kubernetes
// +k8s:deepcopy-gen=false
type NodePoolSpec struct {
	// LogicalName is the internally unique name of the NodePool (not in cloud)
	// - Should always be consistent with a single name entry in a Cluster or NodePoolUpdate
	// - Has to be unique across all nodepools on the Cluster spec
	// - Will NOT always match the name of the NodePool in cloud
	// - Should be treated as an immutable field
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	LogicalName string `json:"logicalName"`
	// CloudResourceName is the name of the cloud resource
	// - should not be set by the client
	// - set by the NodePool mutate if empty
	// - set by the NodePoolUpdate controller depending on the update type
	// +kubebuilder:validation:Optional
	CloudResourceName string `json:"cloudResourceName,omitempty"`
	// Description provides an optional description to the node pool
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Description string `json:"description,omitempty"`
	// DiskSize is the amount of disk space to assign to the nodes in MBs
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=10
	DiskSize int64 `json:"diskSize"`
	// Autoscaling indicates the node pool should autoscale
	// +kubebuilder:validation:Optional
	Autoscaling *AutoScalingOptions `json:"autoscaling,omitempty"`
	// Expires provides a time for automatic expiration of the cluster
	// +kubebuilder:validation:Optional
	Expires *metav1.Time `json:"expires,omitempty"`
	// Image is the image we should use on the instances of this pool
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`
	// Labels is a collection of label placed on to the nodepool
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`
	// MaxPodsPerNode controls how many pods can be scheduled onto each node in this pool
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	MaxPodsPerNode int64 `json:"maxPodsPerNode,omitempty"`
	// Machine is the instance type of the nodes in the poll
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Machine string `json:"machine"`
	// ProviderDetails provides any cloud specific options for this nodepool
	// +kubebuilder:validation:Optional
	ProviderDetails *NodePoolProviderDetails `json:"providerDetails,omitempty"`
	// Size is initial size if autoscaling defined - or the effective size
	// if no autoscaling is enabled
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	Size int64 `json:"size"`
	// Spot if defines enables nodepool to use spot instances
	// +kubebuilder:validation:Optional
	Spot *SpotInstancesOptions `json:"spot,omitempty"`
	// Taints defines a collection scheduling taints placed on the nodepool
	// +kubebuilder:validation:Optional
	Taints []NodeTaint `json:"taints,omitempty"`
	// Channel describes the channel a NodePool follows
	// If specified on a cluster, must be identical to the channel on the cluster
	// +kubebuilder:validation:Optional
	Channel string `json:"channel,omitempty"`
	// Version is the specified version of kubernetes on the node pool
	// This cannot be set by a client if a channel is also specified
	// Will be set by the NodePoolUpdate controller on actual nodepools if a channel is specified
	// +kubebuilder:validation:Optional
	Version string `json:"version,omitempty"`
	// Zones defines a list of cloud specific availability zones where the
	// nodes are permitted to run
	// +kubebuilder:validation:Optional
	Zones []string `json:"zones,omitempty"`
	// DefaultUpdateOptions defines how this nodepool can be upgraded
	// - Used by the cluster controller to generate NodePoolUpdates based on spec changes
	// - Ignored when a NodePoolUpdate is created directly
	// +kubebuilder:validation:Optional
	DefaultUpdateOptions *NodePoolUpdateOptions `json:"defaultUpdateOptions,omitempty"`
}

// GetNodePoolMetaName returns the name of the nodepool as it will be stored in the metadata
func (n *NodePoolSpec) GetNodePoolMetaName(cluster Cluster) string {
	return cluster.Name + "-" + n.LogicalName
}

// GenerateCloudResourceName generates the CloudResourceName when creating a new nodepool
// - In AKS this is done before the nodepool CRD is created and the webhook is called
func (n *NodePoolSpec) GenerateCloudResourceName() string {
	return n.LogicalName
}

// IsNodePoolEquivalentForChannel returns true if the nodepool is equivalent to the spec
// will ignore version if channel is specified
func (n *NodePoolSpec) IsNodePoolEquivalentForChannel(u *NodePoolSpec) bool {
	nc := n.DeepCopy()
	uc := u.DeepCopy()
	if nc.Channel != "" {
		// channel specified - ignore version
		nc.Version = ""
		uc.Version = ""
	}
	if nc.CloudResourceName != "" {
		// cloud resource name is set - ignore
		nc.CloudResourceName = ""
		uc.CloudResourceName = ""
	}
	if reflect.DeepEqual(nc, uc) {
		// nodepool is equal to spec - no op - move onto next nodepool
		return true
	}
	return false
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodePoolList contains a list of nodepools
type NodePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodePool `json:"items"`
}

func (n *NodePoolList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(n.Items))
	for i, item := range n.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (n *NodePoolList) SetItems(objects []corev1alpha1.Object) {
	n.Items = []NodePool{}
	for _, o := range objects {
		n.Items = append(n.Items, *(o.(*NodePool)))
	}
}

// AutoScalingOptions define the options per cloud provider for
// autoscaling
type AutoScalingOptions struct {
	// Enabled indicates the node pool to use autoscaling
	// +kubebuilder:validation:Optional
	Enabled bool `json:"enabled,omitempty"`
	// MinSize is minimum number of nodes if autoscaling is enabled
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	MinSize int64 `json:"minSize,omitempty"`
	// MaxSize is the maximum number of nodes if autoscaling is enabled
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	MaxSize int64 `json:"maxSize,omitempty"`
}

// SpotInstancesOptions defines the options for spot instances
type SpotInstancesOptions struct {
	// Enabled indicates the node pool should use spots
	// +kubebuilder:validation:Optional
	Enabled bool `json:"enabled,omitempty"`
	// AKS defines the options for AKS spot instances
	// +kubebuilder:validation:Optional
	AKS *AKSNodePoolSpotInstances `json:"aks,omitempty"`
	// EKS provides additional options for EKS
	// +kubebuilder:validation:Optional
	EKS *EKSNodePoolSpotInstances `json:"eks,omitempty"`
}

// NodePoolProviderDetails defines the parameters for cloud specific options - i.e
// options which cannot be consolidated as they are too specific to the
// chosen cloud vendor
// +union
type NodePoolProviderDetails struct {
	// +unionDiscriminator
	// +kubebuilder:validation:Required
	// +required
	Type ProviderType `json:"type"`
	// AKS defines the cloud specific options for AKS clusters
	// +kubebuilder:validation:Optional
	AKS *AKSNodePoolSpec `json:"aks,omitempty"`
	// EKS defines a cloud specific options for EKS clusters
	// +kubebuilder:validation:Optional
	EKS *EKSNodePoolSpec `json:"eks,omitempty"`
	// GKE is the provider specification for their clusters
	// +kubebuilder:validation:Optional
	GKE *GKENodePoolSpec `json:"gke,omitempty"`
	// Unmanaged provides the specification for unmanaged cluster
	// +kubebuilder:validation:Optional
	Unmanaged *UnmanagedNodePoolSpec `json:"unmanaged,omitempty"`
}

type NodePoolStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`

	// Name is the name of the node pool
	Name string `json:"name,omitempty"`
	// Details is message from the provider about the node pool
	// +kubebuilder:validation:Optional
	Details string `json:"details,omitempty"`
	// ProvisioningState is provisioning state of the nodePool
	// +kubebuilder:validation:Optional
	ProvisioningState ProvisioningState `json:"providerStatus,omitempty"`
	// NodeCount is the number of nodes in the node pool
	// +kubebuilder:validation:Optional
	NodeCount int `json:"nodeCount,omitempty"`
	// NodeCountReady is the number of nodes in the node pool which are ready
	// +kubebuilder:validation:Optional
	NodeCountReady int `json:"nodeCountReady,omitempty"`
}

// NodeTaint is the structure of a taint on a nodepool
// https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/
type NodeTaint struct {
	// Key provides the key definition for this tainer
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key,omitempty"`
	// Value is arbitrary value for this taint to compare
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Value string `json:"value,omitempty"`
	// Effect is desired action on the taint
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Effect string `json:"effect,omitempty"`
}

// GetCommonStatus implements CommonStatusAware for Cluster
func (n *NodePool) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &n.Status.CommonStatus
}

// IsReady checks if NodePool is ready
func (n *NodePool) IsReady() bool {
	return n.Status.InCondition(corev1alpha1.ConditionReady)
}

// IsError checks if NodePool is in error status
func (n *NodePool) IsError() bool {
	return n.Status.Status.IsError()
}

func (NodePool) APIPath() string {
	return "nodepools"
}
func (n *NodePool) Clone() corev1alpha1.Object {
	return n.DeepCopy()
}
func (n *NodePool) CloneInto(o corev1alpha1.Object) {
	n.DeepCopyInto(o.(*NodePool))
}
func (NodePool) ListType() corev1alpha1.ObjectList {
	return &NodePoolList{}
}

func (NodePoolList) ObjectType() corev1alpha1.Object {
	return &NodePool{}
}

func (n *NodePoolList) Clone() corev1alpha1.ObjectList {
	return n.DeepCopy()
}
func (n *NodePoolList) CloneInto(o corev1alpha1.ObjectList) {
	n.DeepCopyInto(o.(*NodePoolList))
}
