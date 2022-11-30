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
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	cloudaccessv2 "github.com/appvia/wayfinder/server/cloudaccess/apis/cloudaccess/v2beta1"
	networkingv2 "github.com/appvia/wayfinder/server/cloudinfra/networks/apis/networking/v2beta1"
)

// ClusterKind is the kind for a Cluster
const ClusterKind = "Cluster"

// ClusterGVK is the GVK for a Cluster
var ClusterGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    ClusterKind,
}

// +kubebuilder:webhook:name=clusters.compute.appvia.io,mutating=false,path=/validate/compute.appvia.io/clusters,verbs=create;update;delete,groups="compute.appvia.io",resources=clusters,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=clusters.compute.appvia.io,mutating=true,path=/mutate/compute.appvia.io/clusters,verbs=create;update,groups="compute.appvia.io",resources=clusters,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

const (
	MaintenanceWindowLayout          = "15:04"
	MaintenanceWindowDefaultDuration = time.Hour * 1
)

const (
	// PreviousVersionAnnotation provides the controller with the previous version
	PreviousVersionAnnotation = "clusters.appvia.io/previous"
	// UpgradeAnnotation provides the controller the indicate we should update
	UpgradeAnnotation = "clusters.appvia.io/upgrade"
	// UpgradeAnnotationFullVersion is the full patch version to upgrade to
	UpgradeAnnotationFullVersion = "clusters.appvia.io/full-upgrade"
	// LabelKubernetesVersion is current kubernetes version
	LabelKubernetesVersion = "clusters.appvia.io/kubernetes-version"
)

// Authentication provides options to the authentication module
type Authentication struct {
	// DisableInline indicates if we disable inline authorization
	// +kubebuilder:validation:Optional
	DisableInline *bool `json:"disableInline,omitempty"`
}

// AuthorizedNetwork provides a definition for the authorized networks
type AuthorizedNetwork struct {
	// Name provides a descriptive name for this network
	Name string `json:"name"`
	// CIDR is the network range associated to this network
	CIDR string `json:"cidr"`
}

const (
	// NetworkAssignmentAutoMode indicates the address space should be auto assigned
	NetworkAssignmentAutoMode = "auto"
)

// Networking defines the structure for all the internal network parameters
// These are not used to create a network, but to configure the cluster
type Networking struct {
	// AuthorizedNetworks is a collection of authorized networks which is
	// permitted to speak to our authentication proxy
	// +kubebuilder:validation:Optional
	AuthorizedNetworks []*AuthorizedNetwork `json:"authorizedNetworks,omitempty"`
	// AuthorizedMasterNetworks is a collection of authorized networks which is
	// permitted to speak to the cloud kubernetes API, default to all if not provided.
	// +kubebuilder:validation:Optional
	AuthorizedMasterNetworks []*AuthorizedNetwork `json:"authorizedMasterNetworks,omitempty"`
	// NetworkProvider defines a network cni provider for the cluster
	// +kubebuilder:validation:Optional
	NetworkProvider string `json:"networkProvider,omitempty"`
	// NetworkPlan optionally defines the network plan for the cluster
	// If the network plan is not provided, the default network plan will be used from the cluster plan
	// +kubebuilder:validation:Optional
	NetworkPlan string `json:"networkPlan,omitempty"`
	// NetworkRef is reference to a network where the cluster should reside
	// +kubebuilder:validation:Optional
	NetworkRef *corev1alpha1.Ownership `json:"networkRef,omitempty"`
}

// AuthorizedMasterNetworksList returns a list of all the cidrs
func (n *Networking) AuthorizedMasterNetworksList() []string {
	var list []string

	for i := 0; i < len(n.AuthorizedMasterNetworks); i++ {
		list = append(list, n.AuthorizedMasterNetworks[i].CIDR)
	}

	return list
}

// AuthorizedNetworksList returns a list of all the cidrs
func (n *Networking) AuthorizedNetworksList() []string {
	var list []string

	for i := 0; i < len(n.AuthorizedNetworks); i++ {
		list = append(list, n.AuthorizedNetworks[i].CIDR)
	}

	return list
}

// SupportedTaints is a collection of supported taints
var SupportedTaints = []string{
	"NoExecute",
	"NoSchedule",
	"PreferNoSchedule",
}

// ProviderType represents the concrete type of account that a CloudAccount represents
// +kubebuilder:validation:Enum=AKS;EKS;GKE;Unmanaged
type ProviderType string

// PodSecurityStandard defines the Pod Security Standard options
type PodSecurityStandard struct {
	// Enabled indicates the addons is enabled
	// +kubebuilder:validation:Optional
	Enabled bool `json:"enabled,omitempty"`

	// DefaultProfile is the default profile to use for the cluster
	// wayfinder managed namespaces
	DefaultProfile string `json:"defaultProfile,omitempty"`

	// AllowedProfiles is a list of profiles that are allowed to be used
	// in the cluster
	AllowedProfiles []string `json:"allowed,omitempty"`
}

// ClusterNodePool is alias to NodePoolSpec
type ClusterNodePool = NodePoolSpec

// ClusterSpec defines the desired state of a cluster
// +k8s:openapi-gen=true
type ClusterSpec struct {
	// Allocation describes which workspaces can use this cluster.
	// +kubebuilder:validation:Optional
	Allocation *corev1alpha1.ResourceAllocation `json:"allocation,omitempty"`
	// Channel describes the channel a cluster follows. On create, you must populate either channel
	// or version, but not both.
	// +kubebuilder:validation:Optional
	Channel string `json:"channel,omitempty"`
	// CloudAccessConfigRef is a reference to the cloud access configuration to use when building the cluster
	// +kubebuilder:validation:Optional
	CloudAccessConfigRef *cloudaccessv2.CloudAccessConfigReference `json:"cloudAccessConfigRef,omitempty"`
	// CloudResourceName specifies the name of the cluster in the cloud
	// Can be left blank so that the name is derived from the wayfinder workspace and resource name
	CloudResourceName string `json:"cloudResourceName,omitempty"`
	// Description provides a short summary to the use of the cluster
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
	// EnableAutoUpgrade indicates wayfinder should handle any upgrades for the clusters
	// +kubebuilder:validation:Optional
	EnableAutoUpgrade *bool `json:"enableAutoUpgrade,omitempty"`
	// EnablePrivateCluster indicates the cluster should be made private
	// +kubebuilder:validation:Optional
	EnablePrivateCluster *bool `json:"enablePrivateCluster,omitempty"`
	// Expires provides the duration (TTL) before automatic expiration of the cluster
	// +kubebuilder:validation:Optional
	Expires *metav1.Duration `json:"expires,omitempty"`
	// Maintenance provides the details around the maintenance window
	// +kubebuilder:validation:Optional
	Maintenance *MaintenanceWindow `json:"maintenance,omitempty"`
	// Networking provides the details around the cluster networking options
	// +kubebuilder:validation:Optional
	Networking *Networking `json:"networking,omitempty"`
	// NodePools is a collection of node pools associated to the cluster
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	NodePools []*ClusterNodePool `json:"nodePools,omitempty"`
	// Provider refers to the cluster type (e.g. AKS, GKE, EKS)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Provider string `json:"provider"`
	// Plan refers to the original plan the cluster was created from
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Plan string `json:"plan,omitempty"`
	// ProviderDetails defines cloud specific cluster options
	// +kubebuilder:validation:Optional
	ProviderDetails *ClusterProviderDetails `json:"providerDetails,omitempty"`
	// SecretRef is a reference to an existing secret containing an administrative
	// access token for this cluster. Required only for clusters which are not
	// created by Wayfinder.
	// +kubebuilder:validation:Optional
	SecretRef *v1.SecretReference `json:"secretRef,omitempty"`
	// Stage is the name of the stage for this cluster
	// +kubebuilder:validation:Required
	Stage string `json:"stage"`
	// Security contains security settings for the cluster
	// +kubebuilder:validation:Optional
	Security *Security `json:"security,omitempty"`
	// Region is the region you want the cluster to reside
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Region string `json:"region,omitempty"`
	// QuotaLimits provide defaults for quotas and limits which are critical for multi-tenancy
	// +kubebuilder:validation:Optional
	QuotaLimits *ClusterQuotaLimits `json:"quotaLimits,omitempty"`
	// Version is the kubernetes version to use. On create, you must populate either Channel or
	// Version, but not both.
	// +kubebuilder:validation:Optional
	Version string `json:"version,omitempty"`
}

// AllocatedToWorkspace implements the Allocatable interface for Cluster, it will return true if the
// specified workspace is allowed to use this cluster according to the allocation settings on the
// spec.
func (c *Cluster) AllocatedToWorkspace(ws corev1alpha1.WorkspaceKey) bool {
	if c.Spec.Allocation == nil {
		// if no allocation then allowed only if in same workspace
		return c.Namespace == ws.Namespace()
	}

	return c.Spec.Allocation.Allowed(c.Namespace, ws)
}

// UniqueName returns the unique name for the cluster within a workspace
func (c *Cluster) UniqueName(tenantWorkspace corev1alpha1.WorkspaceKey) string {
	if c.Namespace != tenantWorkspace.Namespace() {
		// get the cluster workspace key from the namespace of the cluster
		return corev1alpha1.Workspace(c).Key() + "." + c.Name
	} else {
		return c.Name
	}
}

// GetNodePool returns the named node pool from the cluster spec, or nil if not present
func (c *Cluster) GetNodePool(name string) *ClusterNodePool {
	for i := range c.Spec.NodePools {
		if c.Spec.NodePools[i] != nil && c.Spec.NodePools[i].LogicalName == name {
			return c.Spec.NodePools[i]
		}
	}
	return nil
}

// Security defines the security options for a cluster
type Security struct {
	// PodSecurityStandard indicates we want to use the pod security standard
	PodSecurityStandard *PodSecurityStandard `json:"podSecurityStandard,omitempty"`
}

// MaintenanceWindow defines the maintenance window
type MaintenanceWindow struct {
	// WindowStart is the time we can perform updates and upgrades, represented
	// in 24h format e.g. "15:00"
	// +kubebuilder:validation:Optional
	WindowStart string `json:"windowStart,omitempty"`
	// +kubebuilder:validation:Optional
	// Duration is the length of the maintenance window
	WindowDuration string `json:"windowDuration,omitempty"`
}

// StartTime returns the start time of a maintenance window
func (m *MaintenanceWindow) MaintenanceWindowStart() (time.Time, error) {
	return time.Parse(MaintenanceWindowLayout, m.WindowStart)
}

// WindowDuration returns the duration of a maintenance window
func (m *MaintenanceWindow) MaintenanceWindowDuration() (time.Duration, error) {
	if m.WindowDuration == "" {
		return MaintenanceWindowDefaultDuration, nil
	}

	return time.ParseDuration(m.WindowDuration)
}

// ClusterQuotaLimits defines templates for QuotaLimits for a cluster
type ClusterQuotaLimits struct {
	// Templates provide initial defaults for Quotas and Limits for allocated namespaces
	// used when a cluster is allocated to other workspaces to control ResourceQuotas and LimitRanges
	// These values will result in a one time creation of:
	// - a new cluster workspace QuotaLimit object for "constraints" per workspace (with a namespace claim)
	// - a tenant workspace QuotaLimit object for namespace defaults per workspace
	// Required if a cluster is allocated for any tenant workspaces
	// +kubebuilder:validation:Optional
	Templates []QuotaLimitTemplate `json:"templates,omitempty"`
	// Default specifies the default template name to use e.g. "Small"
	// A specific template can be specified with labels on NamespaceClaim objects
	// Required if a cluster is allocated for any tenant workspaces
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Default string `json:"default,omitempty"`
}

// GetTemplateOrDefault will obtain a QuotaLimit template or the default or just nil if none defined
func (q *ClusterQuotaLimits) GetTemplateOrDefault(nsClaim *NamespaceClaim, templateLabelName string) (*QuotaLimitTemplate, bool, error) {
	if q == nil {
		return nil, false, nil
	}

	// resolve if a QuotaLimit template name is specified
	labels := nsClaim.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	name := labels[templateLabelName]

	// now see if the template exists on the cluster spec
	var defaultTemplate *QuotaLimitTemplate
	for _, t := range q.Templates {
		// grab this - we may need it
		if t.Name == q.Default {
			// ensure we have new variable for pointer
			new := t
			defaultTemplate = &new
		}
		if len(name) > 0 && t.Name == name {
			return &t, false, nil
		}
	}
	if defaultTemplate == nil {
		return nil, false, fmt.Errorf("invalid cluster, unable to find any QuotaLimit template (or default)")
	}

	return defaultTemplate, true, nil
}

// ClusterProviderDetails defines the parameters for cloud specific options - i.e
// options which cannot be consolidated as they are too specific to the
// chosen cloud vendor
// +union
type ClusterProviderDetails struct {
	// +unionDiscriminator
	// +kubebuilder:validation:Required
	// +required
	Type ProviderType `json:"type"`
	// AKS defines the cloud specific options for AKS clusters
	// +kubebuilder:validation:Optional
	AKS *AKSSpec `json:"aks,omitempty"`
	// EKS defines a cloud specific options for EKS clusters
	// +kubebuilder:validation:Optional
	EKS *EKSSpec `json:"eks,omitempty"`
	// GKE is the provider specification for their clusters
	// +kubebuilder:validation:Optional
	GKE *GKESpec `json:"gke,omitempty"`
	// Unmanaged provides the specification for unmanagement cluster
	// +kubebuilder:validation:Optional
	Unmanaged *UnmanagedSpec `json:"unmanaged,omitempty"`
}

// NetworkRange defines a network block
type NetworkRange struct {
	// Type defines the type of network
	Type string `json:"type,omitempty"`
	// CIDR defines the network range
	CIDR string `json:"cidr,omitempty"`
}

// ClusterStatus defines the observed state of a cluster
// +k8s:openapi-gen=true
type ClusterStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// AuthProxyEndpoint is the endpoint of the authentication proxy for this cluster
	// +kubebuilder:validation:Optional
	AuthProxyEndpoint string `json:"authProxyEndpoint,omitempty"`
	// AuthProxyCertificate is the certificate of the auth proxy endpoint
	// +kubebuilder:validation:Optional
	AuthProxyCertificate string `json:"authProxyCertificate,omitempty"`
	// CaCertificate is the base64 encoded cluster certificate
	// +kubebuilder:validation:Optional
	CaCertificate string `json:"caCertificate,omitempty"`
	// KubeAPIEndpoint is the kubernetes API endpoint url
	// +kubebuilder:validation:Optional
	KubeAPIEndpoint string `json:"kubeApiEndpoint,omitempty"`
	// Network contains the network configuration used by this cluster
	// +kubebuilder:validation:Optional
	NetworkFabric *networkingv2.NetworkFabricStatus `json:"networkFabric,omitempty"`
	// ProviderStatus is provider specific data with types
	// +kubebuilder:validation:Optional
	ProviderStatus ProviderStatus `json:"providerStatus,omitempty"`
	// Version is the kubernetes version of the cluster
	// +kubebuilder:validation:Optional
	Version string `json:"version,omitempty"`
	// DNSZones is a list of Wayfinder-managed DNS zones which are currently available in this
	// cluster with ExternalDNS. This will be populated if the DNSReady condition is true.
	DNSZones []string `json:"dnsZones,omitempty"`
	// Expires is the timestamp after which the cluster should automatically be removed.
	// The populated value should be time at which the cluster is created, plus duration defined in
	// Spec.Expires
	// +kubebuilder:validation:Optional
	Expires *metav1.Time `json:"expires,omitempty"`
	// DeployedChartsImage will be set, once the cluster is in condition HelmReady, to indicate
	// which version of the charts image is deployed and active in this cluster
	// +kubebuilder:validation:Optional
	DeployedChartsImage string `json:"deployedChartsImage,omitempty"`
	// DeployedAuthProxyImage will be set, once the cluster is in condition AuthProxyReady, to
	// indicate which version of the auth proxy image is deployed and active in this cluster
	// +kubebuilder:validation:Optional
	DeployedAuthProxyImage string `json:"deployedAuthProxyImage,omitempty"`
	// DeployedResourceHashes is set to a hash of the resources deployed into the cluster so we can
	// re-apply if the resources change but quickly no-op if they have not.
	// +kubebuilder:validation:Optional
	DeployedResourceHashes []DeployedResourceHash `json:"deployedResourceHashes,omitempty"`
	// Metrics is a list of metrics for this cluster
	// +kubebuilder:validation:Optional
	Metrics Metrics `json:"metrics,omitempty"`
}

type Metrics struct {
	// LastChecked identifies when the metrics of the cluster was last checked.
	// +kubebuilder:validation:Optional
	LastChecked metav1.Time `json:"lastChecked,omitempty"`
	// Metrics are the metrics of the cluster
	// +kubebuilder:validation:Optional
	Metrics []ClusterMetric `json:"metrics,omitempty"`
}

// ClusterMetric defines a metric for a cluster
type ClusterMetric struct {
	// Name is the name of the metric
	// +kubebuilder:validation:Optional
	Name string `json:"name"`
	// CPU is the CPU usage
	// +kubebuilder:validation:Optional
	CPU string `json:"cpu,omitempty"`
	// Memory is the memory usage
	// +kubebuilder:validation:Optional
	Memory string `json:"memory,omitempty"`
	// Pods is the number of pods
	// +kubebuilder:validation:Optional
	Pods string `json:"pods,omitempty"`
}

type ProvisioningState string

const (
	// ProvisioningStateCreating is the state when the node pool is being created
	ProvisioningStateCreating ProvisioningState = "Provisioning"
	// ProvisioningStateUpdating is the state when the node pool is being updated
	ProvisioningStateUpdating ProvisioningState = "Updating"
	// ProvisioningStateDeleting is the state when the node pool is being deleted
	ProvisioningStateDeleting ProvisioningState = "Deleting"
	// ProvisioningStateSucceeded is the state when the node pool is created
	ProvisioningStateSucceeded ProvisioningState = "Succeeded"
	// ProvisioningStateFailed is the state when the node pool is failed
	ProvisioningStateFailed ProvisioningState = "Failed"
	// ProvisioningStateUnknown is the state when the node pool is unknown
	ProvisioningStateUnknown ProvisioningState = "Unknown"
	// ProvisioningStateDegraded is the state when the node pool is degraded
	ProvisioningStateDegraded ProvisioningState = "Degraded"
)

// DeployedResourceHash provides a hash for a deployed resource (e.g. the auth proxy or helm
// controller) so we can quickly no-op if the hash is unchanged
type DeployedResourceHash struct {
	// Name is the name of the resource
	Name string `json:"name,omitempty"`
	// Hash is the computed hash of the resources last applied
	Hash string `json:"hash,omitempty"`
}

// GetDeployedResourceHash returns the deployed hash for resource resName - will return the empty
// string if no hash is found for the deployed resource.
func (c *Cluster) GetDeployedResourceHash(resName string) string {
	for _, r := range c.Status.DeployedResourceHashes {
		if r.Name == resName {
			return r.Hash
		}
	}

	return ""
}

// SetDeployedResourceHash sets the deployed hash for resource resName
func (c *Cluster) SetDeployedResourceHash(resName string, hash string) {
	for i, r := range c.Status.DeployedResourceHashes {
		if r.Name == resName {
			c.Status.DeployedResourceHashes[i].Hash = hash
			return
		}
	}

	// If we're still here, need to add it
	c.Status.DeployedResourceHashes = append(c.Status.DeployedResourceHashes, DeployedResourceHash{Name: resName, Hash: hash})
}

// ProviderStatus is a broken down status per provider for the cluster - i.e. outputs from the
// clusters which are specific to the clouds
type ProviderStatus struct {
	// EKS is the provider status for AWS
	// +kubebuilder:validation:Optional
	EKS *EKSClusterStatus `json:"eks,omitempty"`

	// AKS is the provider status for Azure
	// +kubebuilder:validation:Optional
	AKS *AKSClusterStatus `json:"aks,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is the schema for cluster definitions in Wayfinder
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=clusters,scope=Namespaced,categories={wayfinder}
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

func (Cluster) APIPath() string {
	return "clusters"
}

func (c *Cluster) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}

func (c *Cluster) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*Cluster))
}

func (Cluster) ListType() corev1alpha1.ObjectList {
	return &ClusterList{}
}

// GetServiceAccountRef returns the service account reference else we default
// to the name and namespace of the cluster
func (c *Cluster) GetServiceAccountRef() *v1.SecretReference {
	if c.Spec.SecretRef == nil {
		return &v1.SecretReference{
			Namespace: c.Namespace,
			Name:      c.Name,
		}
	}

	return c.Spec.SecretRef
}

// GetCommonStatus implements CommonStatusAware for Cluster
func (c *Cluster) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// Ownership returns a ownership resource from cluster definition
func (c *Cluster) Ownership() corev1alpha1.Ownership {
	return corev1alpha1.Ownership{
		Group:     GroupVersion.Group,
		Version:   GroupVersion.Version,
		Kind:      "Cluster",
		Namespace: c.Namespace,
		Name:      c.Name,
	}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList contains a list of clusters
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func (ClusterList) ObjectType() corev1alpha1.Object {
	return &Cluster{}
}

func (l *ClusterList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}

func (l *ClusterList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*ClusterList))
}

func (l *ClusterList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *ClusterList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []Cluster{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*Cluster)))
	}
}
