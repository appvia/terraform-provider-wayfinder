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

// +kubebuilder:webhook:name=nodepoolupdates.compute.appvia.io,mutating=false,path=/validate/compute.appvia.io/nodepoolupdates,verbs=create;update;delete,groups="compute.appvia.io",resources=nodepoolupdates,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=nodepoolupdates.compute.appvia.io,mutating=true,path=/mutate/compute.appvia.io/nodepoolupdates,verbs=create;update,groups="compute.appvia.io",resources=nodepoolupdates,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// NodePoolUpdateKind is the kind for a NodePoolUpdate
const NodePoolUpdateKind = "NodePoolUpdate"

// NodePoolUpdateGVK is the GVK for a Cluster node pool
var NodePoolUpdateGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    NodePoolUpdateKind,
}

// NewNodePoolUpdate creates a new NodePoolReplacement with the specified name and namespace
func NewNodePoolUpdate(name, namespace string) *NodePoolUpdate {
	return &NodePoolUpdate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodePoolUpdate is the schema for cluster nodepool replacement definitions in Wayfinder
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodepoolupdates,scope=Namespaced,categories={wayfinder}
type NodePoolUpdate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodePoolUpdateSpec   `json:"spec,omitempty"`
	Status NodePoolUpdateStatus `json:"status,omitempty"`
}

func (NodePoolUpdate) APIPath() string {
	return "nodepoolupdates"
}
func (np *NodePoolUpdate) Clone() corev1alpha1.Object {
	return np.DeepCopy()
}
func (np *NodePoolUpdate) CloneInto(o corev1alpha1.Object) {
	np.DeepCopyInto(o.(*NodePoolUpdate))
}
func (NodePoolUpdate) ListType() corev1alpha1.ObjectList {
	return &NodePoolUpdateList{}
}
func (np *NodePoolUpdate) IsReady() bool {
	return np.Status.InCondition(corev1alpha1.ConditionReady)
}

// IsComplete implements the IsComplete method for the UpdateObject interface
func (np *NodePoolUpdate) IsComplete() bool {
	return np.Status.EstimatedPercentageComplete == 100
}

// EstimatePercentageComplete implements the EstimatePercentageComplete method for the UpdateObject interface
func (np *NodePoolUpdate) EstimatePercentageComplete() int {
	return np.Status.EstimatedPercentageComplete
}

// IsAwaitingApproval implements the IsAwaitingApproval method for the UpdateObject interface
func (np *NodePoolUpdate) IsAwaitingApproval() bool {
	return np.Spec.AwaitingApproval
}

// GetPreRequisite implements the GetPreRequisite method for the UpdateObject interface
func (np *NodePoolUpdate) GetPreRequisite() *corev1alpha1.UpdateStep {
	return np.Spec.PreRequisite
}

// NodePoolReplacementStrategy is the update strategy type for a NodePoolUpdate
// Only "None" is supported now
// https://appviakore.atlassian.net/browse/WF-2131 will introduce NewReplacement
// +kubebuilder:validation:Enum=None
type NodePoolReplacementStrategy string

// NodePoolReplacementStrategy is the CloudSupported NodePoolUpdate update in place strategy
// Only "CloudDefault" is supported now
// https://appviakore.atlassian.net/browse/WF-2131 will introduce None (e.g. only replacement)
// +kubebuilder:validation:Enum=CloudDefault
type NodePoolCloudUpdateStrategy string

const (
	// NodePool Replacement Strategies

	// NodePoolReplacementStrategyNewReplacement will create a new node pool and migrate workloads to it
	// - will drain and delete the old NodePool nodes, one at a time
	NodePoolReplacementStrategyNewReplacement NodePoolReplacementStrategy = "NewReplacement"
	// NodePoolReplacementStrategyNone signifies we do not manage replacement NodePools
	// We need to validate fields that are not supported by the cloud provider
	NodePoolReplacementStrategyNone NodePoolReplacementStrategy = "None"

	// NodePool Cloud Update Strategies

	// NodePoolCloudUpdateStrategyNone will disable cloud provider updates
	// Only the replacement strategy options are then used
	NodePoolCloudUpdateStrategyNone NodePoolCloudUpdateStrategy = "None"
	// NodePoolCloudUpdateStrategyCloudDefault will defer to the cloud provider mechanism for updating
	NodePoolCloudUpdateStrategyCloudDefault NodePoolCloudUpdateStrategy = "CloudDefault"
)

// NodePoolUpdateNewReplacementConfig are the options for a replacement strategy
// Not used yet - see https://appviakore.atlassian.net/browse/WF-2131
type NodePoolUpdateNewReplacementConfig struct {
	// NodeDrainTimeOut is the delay to wait for a node to drain before force deleting it
	// After this delay and then the node will be force deleted
	// +kubebuilder:validation:Optional
	NodeDrainTimeOut *metav1.Duration `json:"nodeDrainTimeOut,omitempty"`
}

// NodePoolUpdateOptions are all the options for updating or replacing a NodePool
type NodePoolUpdateOptions struct {
	// ReplacementStrategy is how node pools can get replaced when required
	// +unionDiscriminator
	// +kubebuilder:validation:Required
	// +kubebuilder:default:=None
	// +required
	ReplacementStrategy NodePoolReplacementStrategy `json:"replacementStrategy"`

	// CloudUpdateStrategy is how the cloud provider can update the node pool (if at all)
	// +unionDiscriminator
	// +kubebuilder:validation:Required
	// +required
	CloudUpdateStrategy NodePoolCloudUpdateStrategy `json:"cloudUpdateStrategy"`

	// NewReplacementConfig specifies how to create a new node pool and migrate workloads to it
	// +kubebuilder:validation:Optional
	NewReplacementConfig *NodePoolUpdateNewReplacementConfig `json:"newReplacementConfig,omitempty"`
}

// NodePoolUpdateSpec defines a node pool update or replacement in kubernetes
type NodePoolUpdateSpec struct {
	corev1alpha1.UpdateSpec `json:",inline"`
	// NewSpec is the desired spec of the existing (if updated) or new node pool (if replaced)
	// +kubebuilder:validation:Required
	NewSpec NodePoolSpec `json:"newSpec"`
	// Options are the options for the update or replacement strategy
	// +kubebuilder:validation:Required
	Options NodePoolUpdateOptions `json:"options"`
	// NodePoolToUpdate is the reference to the NodePool metadata.Name to be updated / replaced
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	NodePoolToUpdate string `json:"nodePoolToUpdate"`
	// TargetKubernetesVersion is the specific Kubernetes version to use for the new node pool spec
	// - Only valid when a channel is also specified
	// - Normally be an increment of the current, existing NodePool version
	// - Not be greater than the cluster version
	// +kubebuilder:validation:Optional
	TargetKubernetesVersion string `json:"targetKubernetesVersion,omitempty"`
}

// NodeStatus is the migration status of a node in a node pool
type NodeStatus struct {
	// Name is the name of the node
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// DrainStartTime is the time that the node draining was initiated
	// This is used along with the NodeDrainTimeOut to determine if the node should be force deleted
	// +kubebuilder:validation:Optional
	DrainStartTime metav1.Time `json:"drainStartTime,omitempty"`
	// Drained is true if the node has been successfully drained (ready for deletion)
	// +kubebuilder:validation:Optional
	Drained bool `json:"drained,omitempty"`
	// Migrated is true if the node has been successfully migrated to the new node pool
	// +kubebuilder:validation:Optional
	Migrated bool `json:"migrated,omitempty"`
	// KubeletVersion is the K8s version of the node
	// +kubebuilder:validation:Optional
	KubeletVersion string `json:"kubeletVersion"`
	// MachineID is the K8s version of the node
	// +kubebuilder:validation:Optional
	MachineID string `json:"machineID"`
}

// OriginalNodePoolStatus is the status of a node pool being updated / replaced
type OriginalNodePoolStatus struct {
	// Nodes provide the information about the nodes to migrate stored by name
	// +kubebuilder:validation:Optional
	Nodes map[string]*NodeStatus `json:"nodes,omitempty"`
	// OriginalAutoScalingOptions are the auto scaling options for the node pool before we started migrating
	// +kubebuilder:validation:Optional
	OriginalAutoScalingOptions *AutoScalingOptions `json:"originalAutoScalingOptions,omitempty"`
	// AutoscalingDisabled is true if the autoscaling has been disabled
	// +kubebuilder:validation:Optional
	AutoscalingDisabled bool `json:"autoscalingDisabled"`
	// Deleted indicates if the old node pool has been deleted
	// +kubebuilder:validation:Optional
	Deleted bool `json:"deleted"`
	// Cordoned indicates if the NodePool has been cordoned yet
	// +kubebuilder:validation:Optional
	Cordoned bool `json:"cordoned"`
	// DrainStartTime is the time the node pool started draining
	// +kubebuilder:validation:Optional
	DrainStartTime metav1.Time `json:"drainStartTime,omitempty"`
	// MigratedNodes is the number of old nodes migrated to the new node pool
	// +kubebuilder:validation:Optional
	MigratedNodes int `json:"migratedNodes"`
	// Migrated indicates if the Workloads have been migrated to the new node pool
	// +kubebuilder:validation:Optional
	Migrated bool `json:"migrated"`
	// Size is the size of the old NodePool
	// +kubebuilder:validation:Optional
	Size int64 `json:"size"`
}

// NodePoolUpdateStatus defines the observed state of NodePoolUpdate
type NodePoolUpdateStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// OriginalNodePool is the status of the node pool being updated / replaced
	// +kubebuilder:validation:Optional
	OriginalNodePool *OriginalNodePoolStatus `json:"originalNodePool,omitempty"`
	// EstimatedPercentageComplete is the estimated percentage complete of the node pool update
	// This will vary per provider and update strategy and cluster usage
	// +kubebuilder:validation:Optional
	EstimatedPercentageComplete int `json:"estimatedPercentageComplete,omitempty"`
}

// GetCommonStatus implements CommonStatusAware
func (np *NodePoolUpdate) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &np.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodePoolUpdateList provides a list of NodePoolReplacement objects
type NodePoolUpdateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodePoolUpdate `json:"items"`
}

func (NodePoolUpdateList) ObjectType() corev1alpha1.Object {
	return &NodePoolUpdate{}
}
func (l *NodePoolUpdateList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *NodePoolUpdateList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*NodePoolUpdateList))
}

func (l *NodePoolUpdateList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *NodePoolUpdateList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []NodePoolUpdate{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*NodePoolUpdate)))
	}
}
