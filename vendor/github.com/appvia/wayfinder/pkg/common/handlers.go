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

// These handlers are used to construct API paths in API server.
// Make sure they match what is defines in controllers and handlers code.
// In example if Type is 'GlobalRepository' then handler should be `global repository`
// so it will get translated to `api/v1alpha1/globalrepositories`

const (
	HandlerNodePools               = "nodepools"
	HandlerNodePoolUpdates         = "nodepoolupdates"
	HandlerCostLimits              = "cost limits"
	HandlerGlobalCostLimit         = "global cost limit"
	HandlerClusterPlans            = "plans"
	HandlerClusterAllocations      = "cluster allocation"
	HandlerQuotaLimit              = "quota limits"
	HandlerNamespaceClaims         = "namespace claims"
	HandlerWorkloadIdentity        = "workload identity"
	HandlerCostImports             = "cost imports"
	HandlerServiceAccess           = "service access"
	HandlerAssumePolicy            = "assume policy"
	HandlerAccessRoleBinding       = "access role binding"
	HandlerAccessPolicy            = "access policy"
	HandlerGlobalAccessPolicy      = "global access policy"
	HandlerAccessRole              = "access role"
	HandlerGlobalAccessRole        = "global access role"
	HandlerNetworkFabric           = "networkfabric"
	HandlerNetworkFabricPlan       = "networkfabricplan"
	HandlerAssignableNetworks      = "assignable networks"
	HandlerDNSZone                 = "dns zone"
	HandlerGlobalDNSZone           = "global dns zone"
	HandlerContainerRegistry       = "container registry"
	HandlerGlobalContainerRegistry = "global container registry"
	HandlerRepository              = "repository"
	HandlerRepositoryRelease       = "repository release"
	HandlerPackageRelease          = "package release"
	HandlerKubernetesUpdate        = "kubernetes update"
	HandlerPeeringRule             = "peering rule"
	HandlerPeering                 = "peering"
	HandlerStage                   = "stage"
	HandlerPackage                 = "package"
	HandlerPackageUpdate           = "packageupdate"
)
