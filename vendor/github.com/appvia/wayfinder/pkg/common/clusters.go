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
	"strings"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	computev1alpha1 "github.com/appvia/wayfinder/server/cloudinfra/clusters/apis/compute/v1alpha1"
)

const (
	// ClusterProviderAWS represents Amazon Web Services's Elastic Kubernetes Service (EKS)
	ClusterProviderAWS = "EKS"
	// ClusterProviderGCP represents Google Cloud Platform's Google Kubernetes Engine (GKE)
	ClusterProviderGCP = "GKE"
	// ClusterProviderAzure represents Microsoft Azure's Azure Kubernetes Service (AKS)
	ClusterProviderAzure = "AKS"
	// ClusterProviderWayfinder represents Wayfinder's cluster
	ClusterProviderWayfinder = "Wayfinder"
	// ClusterProviderUnmanaged represents an unmanaged cluster
	ClusterProviderUnmanaged = "Unmanaged"
	// ClusterNameManagement is the special name given to Wayfinder's management cluster
	ClusterNameManagement = "management"
)

var (
	// DummyProviders indicates whether we should use dummy or real cluster/network providers
	// This is set by the controller CLI option --dummy-cluster-providers
	DummyProviders = false
)

// ClusterProviders is the set of supported cluster providers in Wayfinder
var ClusterProviders = []string{
	ClusterProviderAWS,
	ClusterProviderGCP,
	ClusterProviderAzure,
}

var clusterCloudProviderMapping = map[string]string{
	ClusterProviderAWS:       CloudProviderAWS,
	ClusterProviderAzure:     CloudProviderAzure,
	ClusterProviderGCP:       CloudProviderGCP,
	ClusterProviderWayfinder: CloudProviderUnknown,
	ClusterProviderUnmanaged: CloudProviderUnknown,
}

// GetCloudProviderFromClusterProvider returns the cloud provider name for the given cluster provider
func GetCloudProviderFromClusterProvider(clusterProvider string) (string, bool) {
	// Special-case "Wayfinder" because it is not an upper-case acronym like the other real clouds.
	if strings.EqualFold(clusterProvider, ClusterProviderWayfinder) {
		return CloudProviderUnknown, true
	}

	// at some point we started calling this function with lower and upper case providers...
	// This means this check had to be bit more generic
	for clusterP, cloudP := range clusterCloudProviderMapping {
		if strings.EqualFold(clusterP, clusterProvider) {
			return cloudP, true
		}
	}

	return "", false
}

// GetClusterProviderFromCloudProvider returns the cluster provider name for a given cloud provider
func GetClusterProviderFromCloudProvider(cloudProvider string) (string, bool) {
	for k, v := range clusterCloudProviderMapping {
		if v == cloudProvider {
			return k, true
		}
	}
	return "", false
}

var clusterDNSProviderMapping = map[string]string{
	ClusterProviderAWS:   CloudProviderAWS,
	ClusterProviderAzure: CloudProviderAzure,
	ClusterProviderGCP:   CloudProviderGCP,
}

// GetClusterDNSProvider returns the DNS provider name for the given cluster provider
func GetClusterDNSProvider(clusterProvider string) (string, bool) {
	cloudProvider, ok := clusterDNSProviderMapping[clusterProvider]
	return cloudProvider, ok
}

// IsManagementCluster returns true if the referenced cluster represents Wayfinder's own management
// cluster
func IsManagementCluster(c *computev1alpha1.Cluster) bool {
	if c == nil {
		return false
	}
	return corev1alpha1.AdminWorkspace.Owns(c) && c.Name == ClusterNameManagement
}

// IsManagementClusterOwnership returns true if the ownership represents Wayfinder's own management
// cluster
func IsManagementClusterOwnership(o corev1alpha1.Ownership) bool {
	return o.Workspace() == corev1alpha1.AdminWorkspace && o.Name == ClusterNameManagement
}
