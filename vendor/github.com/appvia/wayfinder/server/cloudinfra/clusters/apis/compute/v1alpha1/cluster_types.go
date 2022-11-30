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

import (
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	cloudaccessv1alpha1 "github.com/appvia/wayfinder/server/cloudaccess/apis/cloudaccess/v1alpha1"
	networkingv1alpha1 "github.com/appvia/wayfinder/server/cloudinfra/networks/apis/networking/v1alpha1"
)

// ClusterKind is the kind for a Cluster
const ClusterKind = "Cluster"

// ClusterGVK is the GVK for a Cluster
var ClusterGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    ClusterKind,
}

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

// AutoScalingOptions define the options per cloud provider for
// autoscaling
type AutoScalingOptions struct {
	// Enabled indicates the node pool to use autoscaling
	// +kubebuilder:validation:Optional
	Enabled bool `json:"enabled,omitempty"`
	// MinSize is minimum number of nodes if autoscaling is enabled
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	MinSize int64 `json:"minSize,omitempty"`
	// MaxSize is the maximum numer of nodes if autoscaling is enabled
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	MaxSize int64 `json:"maxSize,omitempty"`
}

// SpotInstancesOptions defines the options for spot instances
type SpotInstancesOptions struct {
	// Enabled indicates the node pool should use spots
	// +kubebuilder:validation:Optional
	Enabled bool `json:"enabled,omitempty"`
	// AKS defines the options for AKS spot instances
	// +kubebuilder:validation:Optional
	AKS *AKSNodePoolSpotInstances `json:"aks,omitempty"`
	// EKS provides additional options for EKS
	// +kubebuilder:validation:Optional
	EKS *EKSNodePoolSpotInstances `json:"eks,omitempty"`
}

// NodePool defines a node pool in kubernetes
// +k8s:openapi-gen=true
type NodePool struct {
	// Name is the name of the nodepool
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Description provides an optional description to the node pool
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Description string `json:"description,omitempty"`
	// DiskSize is the amount of disk space to assign to the nodes in MBs
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=10
	DiskSize int64 `json:"diskSize"`
	// Autocaling indicates the node pool should autoscale
	// +kubebuilder:validation:Optional
	Autoscaling *AutoScalingOptions `json:"autoscaling,omitempty"`
	// Expires provides a time for automatic expiration of the cluster
	// +kubebuilder:validation:Optional
	Expires *metav1.Time `json:"expires,omitempty"`
	// Image is the image we should use on the instances of this pool
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`
	// Labels is a collection of label placed on to the nodepool
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`
	// MaxPodsPerNode controls how many pods can be scheduled onto each node in this pool
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	MaxPodsPerNode int64 `json:"maxPodsPerNode,omitempty"`
	// Machine is the instance type of the nodes in the poll
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Machine string `json:"machine"`
	// ProviderDetails provides any cloud specific options for this nodepool
	// +kubebuilder:validation:Optional
	ProviderDetails *NodePoolProviderDetails `json:"providerDetails,omitempty"`
	// Size is initial size if autoscaling defined - or the effective size
	// if no autoscaling is enabled
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	Size int64 `json:"size"`
	// Spot if defines enables nodepool to use spot instances
	// +kubebuilder:validation:Optional
	Spot *SpotInstancesOptions `json:"spot,omitempty"`
	// Taints defines a collection scheduling taints placed on the nodepool
	// +kubebuilder:validation:Optional
	Taints []NodeTaint `json:"taints,omitempty"`
	// Version is the initial version of kubernetes on the node pool
	// +kubebuilder:validation:Optional
	Version string `json:"version,omitempty"`
	// Zones defines a list of cloud specific availability zones where the
	// nodes are permitted to run
	// +kubebuilder:validation:Optional
	Zones []string `json:"zones,omitempty"`
}

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

// Networking defines the structure for a
type Networking struct {
	// AuthorizedNetworks is a collection of authorized networks which is
	// permitted to speak to authentication proxy
	// +kubebuilder:validation:Optional
	AuthorizedNetworks []*AuthorizedNetwork `json:"authorizedNetworks,omitempty"`
	// AuthorizedMasterNetworks is a collection of authorized networks which is
	// permitted to speak to the cloud kubernetes API, default to all if not provided.
	// +kubebuilder:validation:Optional
	AuthorizedMasterNetworks []*AuthorizedNetwork `json:"authorizedMasterNetworks,omitempty"`
	// ClusterIPV4Cidr is an optional network CIDR which is used to place the
	// node network on
	// +kubebuilder:validation:Optional
	ClusterIPV4Cidr string `json:"clusterIPV4Cidr,omitempty"`
	// NetworkProvider defines a network cni provider for the cluster
	// +kubebuilder:validation:Optional
	NetworkProvider string `json:"networkProvider,omitempty"`
	// NetworkRef is reference to a network where the cluster should reside
	// +kubebuilder:validation:Optional
	NetworkRef *corev1alpha1.Ownership `json:"networkRef,omitempty"`
	// ServicesIPV4Cidr is an optional network cidr configured for the cluster services. If set to 'auto' - cidr will be allocated for you
	// +kubebuilder:validation:Optional
	ServicesIPV4Cidr string `json:"servicesIPV4Cidr,omitempty"`
	// PodsIPV4Cidr is an optional network cidr configured for the pods networks
	// +kubebuilder:validation:Optional
	PodsIPV4Cidr string `json:"podsIPV4Cidr,omitempty"`
}

// HasAutoGeneratingNetworks indicates that one of more of the ranges to be
// auto generated
func (n *Networking) HasAutoGeneratingNetworks() bool {
	mode := networkingv1alpha1.NetworkFabricLayoutModeAuto.String()

	return n.ClusterIPV4Cidr == mode || n.PodsIPV4Cidr == mode || n.ServicesIPV4Cidr == mode
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

// NodeTaint is the structure of a taint on a nodepool
// https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/
type NodeTaint struct {
	// Key provides the key definition for this tainer
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key,omitempty"`
	// Value is arbitrary value for this taint to compare
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Value string `json:"value,omitempty"`
	// Effect is desired action on the taint
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Effect string `json:"effect,omitempty"`
}

// Addons defines the structure for addons and features on a cluster
type Addons struct {
	// TODO: Legacy addons for now so we can have time to migrate to new API
	// Delivery of those are already based on the new API

	// Ingress indicates we want the ingress service enabled
	// +kubebuilder:validation:Optional
	Ingress *IngressAddon `json:"ingress,omitempty"`
	// Terranetes indicates we want the the terranetes-controller enabled
	// +kubebuilder:validation:Optional
	Terranetes *TerranetesAddon `json:"terranetes,omitempty"`
}

// Capability represents a capability backed by a package
type Capability struct {
	// Enabled indicates if the addon is enabled
	Enabled bool `json:"enabled,omitempty"`

	// Description is a description of the addon
	Description string `json:"description,omitempty"`
}

// IngressAddon defines the ingress options
type IngressAddon struct {
	// Enabled indicates the addons is enabled
	// +kubebuilder:validation:Optional
	Enabled *bool `json:"enabled,omitempty"`
}

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

// TerranetesAddon defines the terranetes options
type TerranetesAddon struct {
	// Enabled indicates the addons is enabled
	// +kubebuilder:validation:Optional
	Enabled *bool `json:"enabled,omitempty"`
}

// ClusterSpec defines the desired state of a cluster
// +k8s:openapi-gen=true
type ClusterSpec struct {
	// Allocation describes which workspaces can use this cluster.
	// +kubebuilder:validation:Optional
	Allocation *corev1alpha1.ResourceAllocation `json:"allocation,omitempty"`
	// CloudAccountRef is a reference to the cloud account this cluster should reside in
	// +kubebuilder:validation:Optional
	CloudAccountRef *cloudaccessv1alpha1.CloudAccountReference `json:"cloudAccountRef,omitempty"`
	// CloudResourceName specifies the name of the cluster in the cloudaccount
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
	// Addons provide details on enabled addons sets of the cluster
	// +kubebuilder:validation:Optional
	Addons *Addons `json:"addons,omitempty"`
	// Capabilities provide details on enabled capabilities sets of the cluster
	// +kubebuilder:validation:Optional
	Capabilities map[string]Capability `json:"capabilities,omitempty"`
	// MaintenanceWindow is the time we can perform updates and upgrades
	// +kubebuilder:validation:Optional
	MaintenanceWindow string `json:"maintenanceWindow,omitempty"`
	// Networking provides the details around the cluster networking options
	// +kubebuilder:validation:Optional
	Networking *Networking `json:"networking,omitempty"`
	// NodePools is a collection of node pools associated to the cluster
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	NodePools []*NodePool `json:"nodePools,omitempty"`
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
	// Version is the kubernetes version to use, you can use 'latest' and allow
	// the cloud provider choose the latest release, or using a specific cloud
	// vendor version. Note, when enableAutoUpgrade is enabled you must specify
	// the version as latest
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Version string `json:"version,omitempty"`
}

// GetNodePool returns the named node pool from the cluster spec, or nil if not present
func (c *Cluster) GetNodePool(name string) *NodePool {
	for i := range c.Spec.NodePools {
		if c.Spec.NodePools[i] != nil && c.Spec.NodePools[i].Name == name {
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

// NodePoolProviderDetails defines the parameters for cloud specific options - i.e
// options which cannot be consolidated as they are too specific to the
// chosen cloud vendor
// +union
type NodePoolProviderDetails struct {
	// +unionDiscriminator
	// +kubebuilder:validation:Required
	// +required
	Type ProviderType `json:"type"`
	// AKS defines the cloud specific options for AKS clusters
	// +kubebuilder:validation:Optional
	AKS *AKSNodePoolSpec `json:"aks,omitempty"`
	// EKS defines a cloud specific options for EKS clusters
	// +kubebuilder:validation:Optional
	EKS *EKSNodePoolSpec `json:"eks,omitempty"`
	// GKE is the provider specification for their clusters
	// +kubebuilder:validation:Optional
	GKE *GKENodePoolSpec `json:"gke,omitempty"`
	// Unmanaged provides the specification for unmanaged cluster
	// +kubebuilder:validation:Optional
	Unmanaged *UnmanagedNodePoolSpec `json:"unmanaged,omitempty"`
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
	// CloudAccountRef is a reference to the cloud account to use to retrieve credentials for
	// this cluster. Will be populated if the spec specifies a CloudAccount (org or shared) as
	// the credential object.
	// +kubebuilder:validation:Optional
	CloudAccountRef *cloudaccessv1alpha1.CloudAccountReference `json:"cloudAccount,omitempty"`
	// KubeAPIEndpoint is the kubernetes API endpoint url
	// +kubebuilder:validation:Optional
	KubeAPIEndpoint string `json:"kubeApiEndpoint,omitempty"`
	// Network contains the network configuration used by this cluster
	// +kubebuilder:validation:Optional
	NetworkFabric *networkingv1alpha1.NetworkFabricStatus `json:"networkFabric,omitempty"`
	// ProviderData is provider specific data
	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Optional
	ProviderData *apiextv1.JSON `json:"providerData,omitempty"`
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
}

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

// GetProviderData unmarshals the provider data into the target object
func (c *ClusterStatus) GetProviderData(v interface{}) error {
	if c.ProviderData == nil {
		return nil
	}

	if err := json.Unmarshal(c.ProviderData.Raw, v); err != nil {
		return fmt.Errorf("failed to unmarshal cluster provider data: %w", err)
	}
	return nil
}

// SetProviderData marshals the given object as provider data
func (c *ClusterStatus) SetProviderData(v interface{}) error {
	if v == nil {
		c.ProviderData = nil

		return nil
	}

	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal cluster provider data: %w", err)
	}
	c.ProviderData = &apiextv1.JSON{Raw: raw}

	return nil
}

// +kubebuilder:webhook:name=clusters.compute.appvia.io,mutating=false,path=/validate/compute.appvia.io/clusters,verbs=create;update;delete,groups="compute.appvia.io",resources=clusters,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=clusters.compute.appvia.io,mutating=true,path=/mutate/compute.appvia.io/clusters,verbs=create;update,groups="compute.appvia.io",resources=clusters,versions=v1alpha1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1
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

// GetTargetAccount returns either the spec credentials reference (for legacy accounts) or
// the relevant cloud account reference (if using Cloud Accounts)
// If the cloud account hasn't yet been set, this may return an empty ownership.
func (c *Cluster) GetTargetAccount() corev1alpha1.Ownership {
	if c.Status.CloudAccountRef != nil {
		return c.Status.CloudAccountRef.GetOwnership()
	}

	return corev1alpha1.Ownership{}
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

// CloudResourceName returns the cloud provider name for this cluster
func (c *Cluster) CloudResourceName() string {
	if c.Spec.CloudResourceName != "" {
		return c.Spec.CloudResourceName
	}
	return fmt.Sprintf("%s-%s", c.Namespace, c.Name)
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
