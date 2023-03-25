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

package common

import (
	"fmt"

	networkingv2 "github.com/appvia/wayfinder/server/cloudinfra/networks/apis/networking/v2beta1"
)

// GetNetworkFabricProviderDetailsAzureDefaults will return the default values for the Azure provider for network fabric
func GetNetworkFabricProviderDetailsAzureDefaults(network *networkingv2.NetworkFabric) *networkingv2.ProviderDetailsAzure {
	return &networkingv2.ProviderDetailsAzure{
		ResourceGroup: GetCloudPrefixedName(fmt.Sprintf("%s-%s", network.Spec.CloudResourceName, network.Spec.Location)),
	}
}

// GetNetworkFabricGatewayName returns the expected name of the gateway
func GetNetworkFabricGatewayName(network *networkingv2.NetworkFabric, gateway *networkingv2.NetworkFabricGateway) string {
	switch {
	case gateway.Associate.Location != nil:
		return fmt.Sprintf("wf-%s-%s", network.Spec.CloudResourceName, *gateway.Associate.Location)

	case gateway.Associate.Subnet != nil:
		return fmt.Sprintf("wf-%s-%s", network.Spec.CloudResourceName, *gateway.Associate.Subnet)
	}

	return ""
}
