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

// CloudWatchLogging defines the control plane logging options
// +k8s:openapi-gen=true
type CloudWatchLogging struct {
	// API will enable logging for the Kubernetes API server
	// +kubebuilder:validation:Optional
	API bool `json:"api,omitempty"`
	// Audit will enable logging for the Kubernetes audit
	// +kubebuilder:validation:Optional
	Audit bool `json:"audit,omitempty"`
	// Authenticator will enable logging for the Kubernetes authentication
	// +kubebuilder:validation:Optional
	Authenticator bool `json:"authenticator,omitempty"`
	// ControllerManager will enable logging for the Kubernetes controller manager
	// +kubebuilder:validation:Optional
	ControllerManager bool `json:"controllerManager,omitempty"`
	// Scheduler will enable logging for the Kubernetes scheduler component
	// +kubebuilder:validation:Optional
	Scheduler bool `json:"scheduler,omitempty"`
}

// EKSSpec defines the desired state of EKSCluster
// +k8s:openapi-gen=true
type EKSSpec struct {
	// CloudWatchLogging allows all control plane logging to be enabled
	// +kubebuilder:validation:Optional
	CloudWatchLogging CloudWatchLogging `json:"cloudWatchLogging,omitempty"`
	// EnableEndpointPrivate indicates the EKS endpoint should be private and
	// non-public facing
	// +kubebuilder:validation:Optional
	EnableEndpointPrivate bool `json:"enableEndpointPrivate,omitempty"`
	// SecurityGroupIds is a list of security group IDs
	// +kubebuilder:validation:Optional
	SecurityGroupIDs []string `json:"securityGroupIDs,omitempty"`
	// AdminARNs is the list of roles or users to be granted access to the EKS K8s API
	// +kubebuilder:validation:Optional
	AdminARNs []string `json:"adminARNs,omitempty"`
}

// EKSNodePoolSpotInstances defined the options for EKS spot instances
type EKSNodePoolSpotInstances struct {
	// AdditionalInstanceTypes provides an addotional list of instances to use when spot
	// instances is enabled
	// +kubebuilder:validation:Optional
	AdditionalInstanceTypes []string `json:"additionalInstanceTypes,omitempty"`
}

// EKSNodePoolSpec defines the desired state of EKSCluster
// +k8s:openapi-gen=true
type EKSNodePoolSpec struct {
	// EC2SSHKey is the Amazon EC2 SSH key that provides access for SSH communication with
	// the worker nodes in the managed node group
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html
	// +kubebuilder:validation:Optional
	EC2SSHKey string `json:"eC2SSHKey,omitempty"`
	// SSHSourceSecurityGroups is the security groups that are allowed SSH access (port 22) to the worker nodes
	// +kubebuilder:validation:Optional
	SSHSourceSecurityGroups []string `json:"sshSourceSecurityGroups,omitempty"`
}

// EKSClusterStatus is used to hold any thing related to the EKS clusters
type EKSClusterStatus struct {
	// OIDC is the url for the OIDC endpoint
	// +kubebuilder:validation:Optional
	OIDC string `json:"oidc,omitempty"`
}
