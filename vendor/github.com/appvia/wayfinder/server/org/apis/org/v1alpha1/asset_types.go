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

// AssetType defines the type of a workspace asset
type AssetType string

// @note: IMPORTANT - ensure any updates here are also reflected in the persistence model
const (
	// AssetTypeCluster identifies a cluster asset
	AssetTypeCluster AssetType = "Cluster"
	// AssetTypeNodePool identifies a node pool asset
	AssetTypeNodePool AssetType = "NodePool"
	// AssetTypeNamespace identifies a namespace asset
	AssetTypeNamespace AssetType = "Namespace"
	// AssetTypeCloudService identifies a cloud service (e.g. S3 bucket, RDS instance) asset
	AssetTypeCloudService AssetType = "CloudService"
	// AssetTypeCloudAccount identifies a cloud account asset
	AssetTypeCloudAccount AssetType = "CloudAccount"
)

func (t AssetType) String() string {
	return string(t)
}
