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
	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

const (
	// ConditionCloudAccountAvailable indicates the cloudaccount is available for use
	ConditionCloudAccountAvailable corev1alpha1.ConditionType = "CloudAccountAvailable"
	// ConditionNetworkAvailable indicates the networks has been provisioned and is ready for use
	ConditionNetworkAvailable corev1alpha1.ConditionType = "NetworkAvailable"
	// ConditionSubnetsAvailable indicates the subnets have been provisioned and is ready for use
	ConditionSubnetsAvailable corev1alpha1.ConditionType = "SubnetsAvailable"
	// ConditionConnectivityEstablished indicates connectivity has been provisioned
	ConditionConnectivityEstablished corev1alpha1.ConditionType = "ConnectivityEstablished"
)

// DefaultConditions are the default conditions for all networks
var DefaultConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionCloudAccountAvailable, Name: "Cloud account available"},
	{Type: ConditionNetworkAvailable, Name: "Network available"},
	{Type: ConditionSubnetsAvailable, Name: "Subnets available"},
	{Type: ConditionConnectivityEstablished, Name: "Connectivity established"},
	{Type: corev1alpha1.ConditionReady, Name: "Network healthy"},
}
