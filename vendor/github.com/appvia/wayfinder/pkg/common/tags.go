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

const (
	// ResourceTagWayfinderInstanceID tag stores the Wayfinder instance ID on external resources
	ResourceTagWayfinderInstanceID = "wayfinder-instance"
	// ResourceTagWayfinderWorkspaceID tag stores the Wayfinder workspace ID on external resources
	ResourceTagWayfinderWorkspaceID = "wayfinder-workspace"
	// ResourceTagWayfinderClusterID tag stores the Wayfinder cluster ID on external resources
	ResourceTagWayfinderClusterID = "wayfinder-cluster"
	// ResourceTagWayfinderNetworkID tag stores the Wayfinder network ID on external resources
	ResourceTagWayfinderNetworkID = "wayfinder-network"
	// ResourceTagWayfinderNodepoolID tag stores the Wayfinder cluster nodepool name on external resources
	ResourceTagWayfinderNodepoolID = "wayfinder-nodepool"

	// ResourceTagWayfinderCreatedAt identifies the time that wayfinder created this resource
	ResourceTagWayfinderCreatedAt = "wayfinder-created-at"
	// ResourceTagWayfinderManaged identifies a cluster currently managed by Wayfinder (or installer)
	ResourceTagWayfinderManaged = "wayfinder-managed"
	// ResourceTagWayfinderCreatedAtTimestampFormat defines the format of the value of the
	// ResourceTagWayfinderCreatedAt tag
	ResourceTagWayfinderCreatedAtTimestampFormat = "2006-01-02T15:04:05Z07:00"
)
