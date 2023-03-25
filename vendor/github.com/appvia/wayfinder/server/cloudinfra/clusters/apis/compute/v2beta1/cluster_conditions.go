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
	// ConditionCloudAccountAvailable indicates the status of any cloud account claim which
	// is associated to this cluster
	ConditionCloudAccountAvailable corev1alpha1.ConditionType = "CloudAccountAvailable"

	// ConditionClusterProvisioned indicates if the cluster has been provision in
	// in the cloud vendor - for unmanaged cluster this is always true
	ConditionClusterProvisioned corev1alpha1.ConditionType = "ClusterProvisioned"

	// ConditionDeletionTimestampApplied indicates that the cluster Status has a
	// valid deletion timestamp
	ConditionDeletionTimestampApplied corev1alpha1.ConditionType = "ClusterTTLApplied"

	// ConditionClusterAvailable indicates the kubernetes cluster is available for
	// accessing - i.e we have a service account
	ConditionClusterAvailable corev1alpha1.ConditionType = "ClusterAvailable"

	// ConditionNetworkAvailable indicates the status of any underlying network infrastructure
	// associated to the cluster
	ConditionNetworkAvailable corev1alpha1.ConditionType = "NetworkAvailable"

	// ConditionNodePoolsHealthy indicates nodepool's are provisioned and healthy
	ConditionNodePoolsHealthy corev1alpha1.ConditionType = "NodePoolsHealthy"

	// ConditionClusterConfigured indicates the kubernetes component of the cluster
	// has been bootstrapped
	ConditionClusterConfigured corev1alpha1.ConditionType = "ClusterConfigured"

	// ConditionKubernetesAuthProxyReady indicates that the authentication proxy is
	// successfully deployed into the cluster, thus it is available for user access
	ConditionKubernetesAuthProxyReady corev1alpha1.ConditionType = "AuthProxyReady"

	// ConditionUpgradeScheduled indicates a available upgrade is available and scheduled
	// to upgrade within the maintenance period
	ConditionUpgradeScheduled corev1alpha1.ConditionType = "UpgradeScheduled"

	// ConditionHelmReady indicates that the infrastructure to deliver helm packages into this
	// cluster is deployed and ready to use
	ConditionHelmReady corev1alpha1.ConditionType = "HelmReady"

	// ConditionKognizerReady indicates that the infrastructure to query objects in clusters is deployed
	ConditionKognizerReady corev1alpha1.ConditionType = "KognizerReady"

	// ConditionDNSReady is owned by the DNS operator
	ConditionDNSReady corev1alpha1.ConditionType = "DNSReady"

	// ConditionCertManagerReady will be set when the controller validates cert-manager
	// has been properly installed in the cluster.
	ConditionCertManagerReady corev1alpha1.ConditionType = "CertManagerReady"

	// ConditionMetricsReady will be set when the operator validates the metrics server
	ConditionMetricsReady corev1alpha1.ConditionType = "MetricsReady"

	// ConditionObsoleteResourcesHandled indicates that any resources that have been marked
	// as obsolete have been handled appropriately.
	ConditionObsoleteResourcesHandled corev1alpha1.ConditionType = "ObsoleteResourcesHandled"
)

const (
	// ReasonNoUpgradeAvailable is used for the UpgradeAvailable condition
	ReasonNoUpgradeAvailable = "NoUpgradeAvailable"
)

// DefaultClusterConditions are the default conditions for all clusters
var DefaultClusterConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionCloudAccountAvailable, Name: "Cloud account available"},
	{Type: ConditionNetworkAvailable, Name: "Network available"},
	{Type: ConditionClusterProvisioned, Name: "Cluster provisioned"},
	{Type: ConditionDeletionTimestampApplied, Name: "Cluster TTL applied"},
	{Type: ConditionClusterAvailable, Name: "Service account ready"},
	{Type: ConditionNodePoolsHealthy, Name: "Node pools healthy"},
	{Type: ConditionClusterConfigured, Name: "Cluster configured"},
	{Type: corev1alpha1.ConditionReady, Name: "Cluster healthy"},
	{Type: ConditionObsoleteResourcesHandled, Name: "Obsolete resources handled"},

	{Type: ConditionUpgradeScheduled, Name: "Upgrade scheduled"},
}

// DefaultNodePoolConditions are the default conditions for all clusters
var DefaultNodePoolConditions = []corev1alpha1.ConditionSpec{
	{Type: corev1alpha1.ConditionReady, Name: "Node pools ready"},
}

var DNSConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionDNSReady, Name: "DNS ready"},
}

// CertConditions will be set to monitor the ongoing condition of a cluster's
// configuration and cert-manager
var CertConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionCertManagerReady, Name: "Certificate Manager ready"},
}

// MetricsConditions will be set to monitor the ongoing condition of a cluster's
// Metrics Server
var MetricsConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionMetricsReady, Name: "Metrics Server ready"},
}

const (
	// ConditionPackageUpdateReady is set when a package upgrade is applied to the cluster
	ConditionPackageUpdateReady corev1alpha1.ConditionType = "PackageUpdateReady"
	// ConditionClusterUpdateReady is set when a control plane or node pool upgrade is
	// applied to the cluster
	ConditionClusterUpdateReady corev1alpha1.ConditionType = "ClusterUpdateReady"
)

// PackageUpdateConditions are the conditions on the cluster relating to status of any
// package-related updates
var PackageUpdateConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionPackageUpdateReady, Name: "Package Update ready"},
}

// ClusterUpdateConditions are the conditions on the cluster relating to status of any
// cluster-related updates
var ClusterUpdateConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionClusterUpdateReady, Name: "Cluster Update ready"},
}

const (
	// ConditionClusterAllocationsReady is set by the cluster-allocator operator on the cluster when
	// the cluster allocations are in-sync for the cluster
	ConditionClusterAllocationsReady corev1alpha1.ConditionType = "ClusterAllocationsReady"
)

// ClusterAllocationConditions is the list of extra cluster conditions contributed to the cluster by
// the cluster-allocator operator
var ClusterAllocationConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionClusterAllocationsReady, Name: "Cluster allocations ready"},
}

// ClusterFeatureConditions is the list of extra cluster conditions contributed to the cluster by
// the clusterfeatures operator
var ClusterFeatureConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionKubernetesAuthProxyReady, Name: "Authentication Proxy deployed"},
	{Type: ConditionHelmReady, Name: "Helm controller ready"},
	{Type: ConditionKognizerReady, Name: "Kognizer agent ready"},
}
