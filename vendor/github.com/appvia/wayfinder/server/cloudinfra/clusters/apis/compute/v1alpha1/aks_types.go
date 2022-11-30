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

// AKSSpec defines the desired state of an AKS cluster
// +k8s:openapi-gen=true
type AKSSpec struct {
	// LinuxProfile is the configuration for Linux VMs
	// +kubebuilder:validation:Optional
	LinuxProfile *LinuxProfile `json:"linuxProfile,omitempty"`
	// SKUTier is the Uptime SLA that should be used for the AKS cluster. "Free" or "Paid"
	// +kubebuilder:validation:Enum=Free;Paid
	// +kubebuilder:validation:Optional
	SKUTier string `json:"skuTier,omitempty"`
	// WindowsProfile is the configuration for Windows VMs
	// +kubebuilder:validation:Optional
	WindowsProfile *WindowsProfile `json:"windowsProfile,omitempty"`
	// ResourceGroupOverride will specify a resource group when required
	ResourceGroupOverride string `json:"resourceGroupOverride,omitempty"`
	// InfraResourceGroupOverride will specify a resource group to use for nodes
	// useful when we want to reclaim a cluster
	InfraResourceGroupOverride string `json:"infraResourceGroupOverride,omitempty"`
}

// AKSNodePoolSpotInstances are the options for spot instances in Azure
type AKSNodePoolSpotInstances struct {
	// MaxSpotPrice is the maximum price willing to pay for an spot instance billed in
	// microdollars. The figure is optional and needless to say only relevant when the
	// nodepool is made of spot instances
	// +kubebuilder:validation:Pattern=`^[1-9][0-9]*\.?[0-9]{1,5}|0|-1$`
	// +kubebuilder:validation:Optional
	MaxSpotPrice string `json:"maxSpotPrice,omitempty"`
}

// AKSNodePoolSpec represents a node pool within a AKS cluster
type AKSNodePoolSpec struct {
	// Mode is the type of the node pool.
	// System node pools serve the primary purpose of hosting critical system pods such
	// as CoreDNS and tunnelfront. User node pools serve the primary purpose of hosting
	// your application pods.
	// +kubebuilder:validation:Optional
	Mode string `json:"mode,omitempty"`
}

// LinuxProfile is the configuration for Linux VMs
// +k8s:openapi-gen=true
type LinuxProfile struct {
	// AdminUsername is the admin username for Linux VMs
	// +kubebuilder:validation:Optional
	AdminUsername string `json:"adminUsername,omitempty"`
	// SSHPublicKeys is a list of public SSH keys to allow to connect to the Linux VMs
	// +kubebuilder:validation:Optional
	SSHPublicKeys []string `json:"sshPublicKeys,omitempty"`
}

// WindowsProfile is the configuration for Windows VMs
// +k8s:openapi-gen=true
type WindowsProfile struct {
	// AdminPassword is the admin password for Windows VMs
	// +kubebuilder:validation:Optional
	AdminPassword string `json:"adminPassword,omitempty"`
	// AdminUsername is the admin username for Windows VMs
	// +kubebuilder:validation:Optional
	AdminUsername string `json:"adminUsername,omitempty"`
}

// AKSClusterStatus is used to hold any thing related to the AKS clusters
type AKSClusterStatus struct {
	// InfraResourceGroupName is the resource group that AKS manages
	// +kubebuilder:validation:Optional
	InfraResourceGroupName string `json:"infraResourceGroupName,omitempty"`

	// ClusterPrincipalID is the cluster principal in Azure
	// This is required for assigning permission to the AKS cluster in Azure
	// +kubebuilder:validation:Optional
	ClusterPrincipalID string `json:"clusterPrincipalId,omitempty"`
}
