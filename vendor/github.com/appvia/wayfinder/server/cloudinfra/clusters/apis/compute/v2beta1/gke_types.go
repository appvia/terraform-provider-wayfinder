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

// GKESpec defines the additional options for a GKE cluster
// +k8s:openapi-gen=true
type GKESpec struct {
	// The Horizontal Pod Autoscaler changes the shape of your Kubernetes workload
	// by automatically increasing or decreasing the number of Pods in response to
	// the workload's CPU or memory consumption, or in response to custom metrics
	// reported from within Kubernetes or external metrics from sources outside of
	// your cluster.
	// +kubebuilder:validation:Optional
	EnableHorizontalPodAutoscaler *bool `json:"enableHorizontalPodAutoscaler,omitempty"`
	// EnableHTTPLoadBalancer indicates if the cluster should be configured with
	// the GKE ingress controller. When enabled GKE will autodiscover your
	// ingress resources and provision load balancer on your behalf.
	// +kubebuilder:validation:Optional
	EnableHTTPLoadBalancer *bool `json:"enableHTTPLoadBalancer,omitempty"`
	// EnableShieldedNodes indicates we should enable the shielded nodes options in GKE.
	// This protects against a variety of attacks by hardening the underlying GKE node
	// against rootkits and bootkits.
	EnableShieldedNodes *bool `json:"enableShieldedNodes,omitempty"`
	// EnableStackDriverLogging indicates if Stackdriver logging should be enabled
	// for the cluster
	// +kubebuilder:validation:Optional
	EnableStackDriverLogging *bool `json:"enableStackDriverLogging,omitempty"`
	// EnableStackDriverMetrics indicates if Stackdriver metrics should be enabled
	// for the cluster
	// +kubebuilder:validation:Optional
	EnableStackDriverMetrics *bool `json:"enableStackDriverMetrics,omitempty"`

	// TODO: always derive this off the network when it exists
	// MasterIPV4Cidr is network range used when private networking is enabled.
	// This is the peering subnet used to to GKE master api layer. Note, this must
	// be unique within the network.
	// +kubebuilder:validation:Optional
	MasterIPV4Cidr string `json:"masterIPV4Cidr,omitempty"`
}

// GKENodePoolSpec defines a the cloud specific options for a GKE nodepool
type GKENodePoolSpec struct {
	// EnableAutorepair indicates if the node pool should automatically repair failed nodes
	// +kubebuilder:validation:Optional
	EnableAutorepair *bool `json:"enableAutorepair,omitempty"`
	// EnableAutoUpgrade indicates if the node group should be configured with auto-upgrading
	// enabled. This must be true if the cluster has ReleaseChannel set.
	// +kubebuilder:validation:Optional
	EnableAutoupgrade *bool `json:"enableAutoupgrade,omitempty"`
}
