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

// ProviderDetailsGCP defines the GCP-specific NetworkFabric configuration
// +k8s:openapi-gen=true
type ProviderDetailsGCP struct{}

// NetworkFabricStatusGCP contains the GCP-specific attributes of the status block.
type NetworkFabricStatusGCP struct {
	// Name is the name of the network in GCP
	Name string `json:"name,omitempty"`
	// NetworkRef is the cloud provider reference
	NetworkRef string `json:"networkRef,omitempty"`
	// Gateways provides a status on the gateways and any external addresses
	Gateways []NetworkFabricStatusGCPGateway `json:"gateways,omitempty"`
}

// NetworkFabricStatusGCPGateway defines the status on the gateway
type NetworkFabricStatusGCPGateway struct {
	// Name is name of the gateway
	Name string `json:"name,omitempty"`
	// Location is location of where it resides
	Location string `json:"location,omitempty"`
	// Addresses is the external ip or self links references associated to the gateway
	Addresses []string `json:"addresses,omitempty"`
}
