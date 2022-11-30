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

const (
	// PolicyTypeKubeRBAC is for kubernetes rbac
	PolicyTypeKubeRBAC PolicyType = "KubeRBAC"
)

// +kubebuilder:webhook:name=clusterpolicies.policydelivery.appvia.io,mutating=false,path=/validate/policydelivery.appvia.io/clusterpolicies,verbs=create;update;delete,groups="policydelivery.appvia.io",resources=clusterpolicies,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// ClusterPolicyKind is the kind for a Policy
const ClusterPolicyKind = "ClusterPolicy"

// ClusterPolicyGVK is the GVK for a Policy
var ClusterPolicyGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    ClusterPolicyKind,
}

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterPolicy represents a cluster targeted policy
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=clusterpolicies,scope=Namespaced,categories={wayfinder}
type ClusterPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterPolicySpec   `json:"spec,omitempty"`
	Status ClusterPolicyStatus `json:"status,omitempty"`
}

func (ClusterPolicy) APIPath() string {
	return "clusterpolicies"
}
func (c *ClusterPolicy) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *ClusterPolicy) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*ClusterPolicy))
}
func (ClusterPolicy) ListType() corev1alpha1.ObjectList {
	return &ClusterPolicyList{}
}

// ClusterPolicyStatus defines the status of a cluster policy
// +k8s:openapi-gen=true
type ClusterPolicyStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// Targets shows the status of the distribution to the targets specified by this cluster policy
	// +kubebuilder:validation:Optional
	Targets []TargetStatus `json:"targets"`
}

func (s *ClusterPolicyStatus) SetTargetStatus(cluster, namespace string, ready bool, message string) {
	ts := TargetStatus{
		Cluster:   cluster,
		Namespace: namespace,
		Ready:     ready,
		Message:   message,
	}
	for i, t := range s.Targets {
		if t.Cluster == cluster && t.Namespace == namespace {
			s.Targets[i] = ts
			return
		}
	}
	s.Targets = append(s.Targets, ts)
}

func (s *ClusterPolicyStatus) RemoveTargetStatus(cluster, namespace string) {
	for i, t := range s.Targets {
		if t.Cluster == cluster && t.Namespace == namespace {
			s.Targets = append(s.Targets[:i], s.Targets[i+1:]...)
			return
		}
	}
}

func (s *ClusterPolicyStatus) GetTargetStatus(cluster, namespace string) bool {
	for _, t := range s.Targets {
		if t.Cluster == cluster && t.Namespace == namespace {
			return t.Ready
		}
	}

	return false
}

// TargetStatus describes the status of distribution to a specific target
// +k8s:openapi-gen=true
type TargetStatus struct {
	// Cluster is the name of a cluster in this workspace
	Cluster string `json:"cluster"`
	// Namespace is the name of a namespace claim against the target cluster
	Namespace string `json:"namespace,omitempty"`
	// Ready is true if the most recent attempt to apply this policy to the target cluster/namespace
	// was successful, false otherwise
	Ready bool `json:"ready,omitempty"`
	// Message gives additional information on the status
	Message string `json:"message,omitempty"`
}

// GetCommonStatus implements CommonStatusAware
func (c *ClusterPolicy) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// PolicyType represents the concrete type of a policy to provide
// +kubebuilder:validation:Enum=KubeRBAC
type PolicyType string

// ClusterPolicySpec defines the specification of a policy in a cluster or namespace
// +k8s:openapi-gen=true
type ClusterPolicySpec struct {
	// Target contains targeting information for this cluster policy
	// +kubebuilder:validation:Optional
	Target ClusterPolicyTarget `json:"target"`
	// Policy details
	// +kubebuilder:validation:Optional
	Policy *PolicyProviderDetails `json:"policy,omitempty"`
}

// ClusterPolicyTarget is the targeting specification for this cluster policy
// Not convinced this is required as a PolicyBinding can create one of these per cluster or namespace
// +k8s:openapi-gen=true
type ClusterPolicyTarget struct {
	// Cluster specifies labels for clusters
	Cluster metav1.LabelSelector `json:"cluster,omitempty"`
	// Namespace specifies labels for namespaces
	// +kubebuilder:validation:Optional
	Namespace metav1.LabelSelector `json:"namespace,omitempty"`
}

// IsClusterScoped will return if the object has cluster or namespace targets
func (t *ClusterPolicyTarget) IsClusterScoped() bool {
	if len(t.Namespace.MatchExpressions) == 0 && len(t.Namespace.MatchLabels) == 0 {
		return true
	}
	return false
}

// GetClusterTarget
func (c *ClusterPolicy) GetClusterTarget() *metav1.LabelSelector {
	return &c.Spec.Target.Cluster
}

// GetNamespaceTarget
func (c *ClusterPolicy) GetNamespaceTarget() *metav1.LabelSelector {
	return &c.Spec.Target.Namespace
}

// PolicyProviderDetails
// +union
type PolicyProviderDetails struct {
	// PolicyType is the specific type of policy
	// +unionDiscriminator
	// +kubebuilder:validation:Required
	// +required
	Type PolicyType `json:"type"`
	// KubeRBAC holds the kubernetes rbac details
	// +kubebuilder:validation:Optional
	KubeRBAC *KubeRBACParameters `json:"kuberbac,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterPolicyList provides a list of cluster policies
type ClusterPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterPolicy `json:"items"`
}

func (ClusterPolicyList) ObjectType() corev1alpha1.Object {
	return &ClusterPolicy{}
}
func (l *ClusterPolicyList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *ClusterPolicyList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*ClusterPolicyList))
}

func (l *ClusterPolicyList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *ClusterPolicyList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []ClusterPolicy{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*ClusterPolicy)))
	}
}
