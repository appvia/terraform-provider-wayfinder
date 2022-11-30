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
)

// Label returns a wayfinder label on a resource
func Label(tag string) string {
	return fmt.Sprintf("appvia.io/%s", tag)
}

// NamedLabel returns a wayfinder named label on a resource
func NamedLabel(name, tag string) string {
	return fmt.Sprintf("%s.appvia.io/%s", name, tag)
}

const (
	LabelValueTrue  = "true"
	LabelValueFalse = "false"

	LabelValueDevelopment = "development"
	LabelValueProduction  = "production"
)

const (
	CapabilitiesPrefixName = "capabilities"
	PackagesPrefixName     = "packages"
)

// LabelWayfinderIdentifier is the label for the immutable unique identifier for this instance
// of Wayfinder. This will always be set by Wayfinder and any user-provided values will be ignored.
var LabelWayfinderIdentifier = Label("wayfinderid")

// LabelClusterIdentifier is the label for the immutable unique identifier for a cluster.
// This should be left blank for new clusters (Wayfinder will auto-assign a new value) or
// populated with the identifier of a previously-deleted cluster which this cluster
// replaces. No other values will be permitted, and read-only after the cluster is created.
var LabelClusterIdentifier = Label("clusterid")

// LabelNetworkIdentifier is the label for the immutable unique identifier for a network.
// This should be left blank for new network (Wayfinder will auto-assign a new value) or
// populated with the identifier of a previously-deleted network which this network
// replaces. No other values will be permitted, and read-only after the network is created.
var LabelNetworkIdentifier = Label("networkid")

// LabelWorkspaceIdentifier is the label for the immutable unique identifier for a workspace that
// owns this resource. This should be left blank (Wayfinder will auto-populate with the correct
// identifier for the workspace) or populated with the workspace's identifier for a resource. No
// other values permitted, and read-only after a cluster or workspace is created.
var LabelWorkspaceIdentifier = Label("workspaceid")

// LegacyLabelName is used various k8s resources
var LegacyLabelName = "name"

// LabelName is used various k8s resources
var LabelName = Label("name")

// LabelManaged is used to indicates the resource is managed by us
var LabelManaged = Label("managed")

// LabelManagementCluster is used to indicates a cluster is the Wayfinder management cluster
var LabelManagementCluster = Label("managementcluster")

// LabelAccountIdentifier is used to identify an account
var LabelAccountIdentifier = Label("accountid")

// LabelPlatform is used to identify the platform for an object, e.g. for a service kind
var LabelPlatform = Label("platform")

// LabelProvider is used to identify the provider
var LabelProvider = Label("provider")

// LabelWorkspace is used to categorize objects
var LabelWorkspace = Label("workspace")

// LabelCategory is used to categorize objects
var LabelCategory = Label("category")

// LabelOwnerWorkspace is used to reference the workspace owning this resource in multitenancy
var LabelOwnerWorkspace = Label("ownerworkspace")

// LabelOwnerCluster is used to reference the cluster containing this resource in multitenancy
var LabelOwnerCluster = Label("ownercluster")

// LabelWayfinderCreated is used to record that Wayfinder created this item
var LabelWayfinderCreated = Label("wayfindercreated")

// LabelProduction is used to record whether an object belongs to Production or not
var LabelProduction = Label("production")

// LabelEnvironment is used to record which environment an object belongs to
var LabelEnvironment = Label("environment")

// LabelStage is used to record which stage an object belongs to
var LabelStage = Label("stage")

// LabelNodepool is used to record which nodepool an object belongs to
var LabelNodepool = Label("nodepool")

// LabelQuotaLimit is used to record the name of a quotalimit a namespaceclaim refers to
var LabelQuotaLimit = Label("quotalimit")

// LabelQuotaLimitIsDefault is used to record the fact a namespaceclaim is using default quotalimit values
// The default values are used when LabelQuotaLimit is not present or the templet does not exist on a cluster
var LabelQuotaLimitIsDefault = Label("quotalimitdefault")

// LabelKubernetesVersion is current kubernetes version
var LabelKubernetesVersion = Label("kubernetesversion")

// LabelCapabilitiesName is used to record the name of the capability
var LabelCapabilitiesName = NamedLabel(CapabilitiesPrefixName, "name")

// LabelCapabilitiesDescription is used to record the description of the capability
var LabelCapabilitiesDescription = NamedLabel(CapabilitiesPrefixName, "description")

// Labels are the list of known labels (excluding internally used ones)
var Labels = []string{
	LabelAccountIdentifier,
	LabelCategory,
	LabelOwnerWorkspace,
	LabelOwnerCluster,
	LabelClusterIdentifier,
	LabelEnvironment,
	LabelName,
	LabelNetworkIdentifier,
	LabelPlatform,
	LabelPlatform,
	LabelProduction,
	LabelProvider,
	LabelQuotaLimit,
	LabelQuotaLimitIsDefault,
	LabelStage,
	LabelWayfinderCreated,
	LabelWayfinderIdentifier,
	LabelWorkspace,
	LabelKubernetesVersion,
	LabelCapabilitiesName,
	LabelCapabilitiesDescription,
}
