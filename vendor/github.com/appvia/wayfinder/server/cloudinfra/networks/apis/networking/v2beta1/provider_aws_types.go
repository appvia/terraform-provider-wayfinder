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

// ProviderDetailsAWS defines the AWS-specific NetworkFabric configuration
// +k8s:openapi-gen=true
type ProviderDetailsAWS struct {
	// AvailabilityZones is the number of availability zones used for the network.
	// Must be 2 or 3 or not specified. If specified, must equal the number of privateSubnets.
	// Each subnet (public or private) will be created in a different availability zone.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=3
	AvailabilityZones int `json:"availabilityZones,omitempty"`
	// NATGateways is the number of NAT gateways to create.
	// Must be 1 or 2 or not specified. If specified, must equal the number of privateSubnets.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=3
	NATGateways int `json:"natGateways,omitempty"`
}

// NetworkFabricStatusAWS contains the AWS-specific attributes of the status block.
type NetworkFabricStatusAWS struct {
	// AccoundID is the aws account id
	AccountID string `json:"accountID,omitempty"`
	// AvailabilityZoneIDs is the list of AZ ids
	AvailabilityZoneIDs []string `json:"availabilityZoneIDs,omitempty"`
	// AvailabilityZoneIDs is the list of AZ names
	AvailabilityZoneNames []string `json:"availabilityZoneNames,omitempty"`
	// PrivateIPV4Addresses provides the list of private subnet addresses
	PrivateIPV4Addresses []string `json:"privateIPV4Addresses,omitempty"`
	// PrivateSubnetIDs is a list of subnet IDs to use for the worker nodes
	PrivateSubnetIDs []string `json:"privateSubnetIDs,omitempty"`
	// PublicIPV4Addresses provides the list of public subnet addresses
	PublicIPV4Addresses []string `json:"publicIPV4Addresses,omitempty"`
	// PublicIPV4EgressAddresses provides the source addresses for traffic coming from the cluster
	PublicIPV4EgressAddresses []string `json:"ipv4EgressAddresses,omitempty"`
	// PublicSubnetIDs is a list of subnet IDs to use for resources that need a public IP (e.g. load balancers)
	PublicSubnetIDs []string `json:"publicSubnetIDs,omitempty"`
	// SecurityGroupIDs is a list of security group IDs to use for a cluster
	SecurityGroupIDs []string `json:"securityGroupIDs,omitempty"`
	// VpcID is the identifier of the VPC
	VpcID string `json:"vpcID,omitempty"`
}
