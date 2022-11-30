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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	cloudaccessv1alpha1 "github.com/appvia/wayfinder/server/cloudaccess/apis/cloudaccess/v1alpha1"
)

const (
	// LabelNetworkPeering is label which is automatically added on managed networkfabrics - and
	// used to target peering rule
	LabelNetworkPeering = "networking.appvia.io/peering"
)

// +kubebuilder:webhook:name=networkfabrics.networking.appvia.io,mutating=false,path=/validate/networking.appvia.io/networkfabrics,verbs=create;update,groups="networking.appvia.io",resources=networkfabric,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// NetworkFabricKind is the kind for a NetworkFabric
const NetworkFabricKind = "NetworkFabric"

// NetworkFabricGVK is the GVK for a NetworkFabric
var NetworkFabricGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    NetworkFabricKind,
}

// NewNetworkFabric creates and returns a empty resource
func NewNetworkFabric(name, namespace string) *NetworkFabric {
	return &NetworkFabric{
		TypeMeta: metav1.TypeMeta{
			Kind:       NetworkFabricKind,
			APIVersion: GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// NetworkFabricProviderType represents the concrete type of a network fabric provider
// +kubebuilder:validation:Enum=aws;azure;gcp
type NetworkFabricProviderType string

const (
	// NetworkFabricProviderTypeAWS is for AWS network fabric provider
	NetworkFabricProviderTypeAWS NetworkFabricProviderType = "aws"
	// NetworkFabricProviderTypeAzure is for Azure network fabric provider
	NetworkFabricProviderTypeAzure NetworkFabricProviderType = "azure"
	// NetworkFabricProviderTypeGCP is for GCP network fabric provider
	NetworkFabricProviderTypeGCP NetworkFabricProviderType = "gcp"
)

// NetworkFabricLayoutMode represents the network fabric layout mode
// +kubebuilder:validation:Enum=auto;manual
type NetworkFabricLayoutMode string

// Strings returns the string representation of the mode
func (n NetworkFabricLayoutMode) String() string {
	return string(n)
}

const (
	// NetworkFabricLayoutModeAuto is for network fabric layout mode auto
	NetworkFabricLayoutModeAuto NetworkFabricLayoutMode = "auto"
	// NetworkFabricLayoutModeManual is for network fabric layout mode manual
	NetworkFabricLayoutModeManual NetworkFabricLayoutMode = "manual"
)

// NetworkFabricSubnetType represents the network fabric subnet type
// +kubebuilder:validation:Enum=public;private
type NetworkFabricSubnetType string

const (
	// NetworkFabricSubnetTypePrivate is for network fabric subnet type private
	NetworkFabricSubnetTypePrivate NetworkFabricSubnetType = "private"
	// NetworkFabricSubnetTypePublic is for network fabric subnet type public
	NetworkFabricSubnetTypePublic NetworkFabricSubnetType = "public"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkFabric is the schema for NetworkFabric
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=networkfabrics,scope=Namespaced,categories={wayfinder}
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
type NetworkFabric struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkFabricSpec   `json:"spec,omitempty"`
	Status NetworkFabricStatus `json:"status,omitempty"`
}

func (NetworkFabric) APIPath() string {
	return "networkfabrics"
}
func (c *NetworkFabric) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *NetworkFabric) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*NetworkFabric))
}
func (NetworkFabric) ListType() corev1alpha1.ObjectList {
	return &NetworkFabricList{}
}

// NetworkFabricPrivateOptions are options related to private networking
type NetworkFabricPrivateOptions struct {
	// Enabled indicates we are expecting the predefined network layout to
	// generate a private network.
	Enabled *bool `json:"enabled,omitempty"`
	// Gateways provides the options around cloud NAT gateways
	// +kubebuilder:validation:Optional
	Gateways []*NetworkFabricGateway `json:"gateways,omitempty"`
}

// NetworkFabricGateway are cloud agnostic settings for nat gateway
type NetworkFabricGateway struct {
	// Associate is the location of the gateway
	// +kubebuilder:validation:Required
	Associate NetworkFabricGatewayAssociation `json:"associate"`
	// IPv4 defines the ip allocation options of the gateway
	// +kubebuilder:validation:Required
	IPV4 NetworkFabricGatewayIPV4 `json:"ipv4"`
	// NAT provides cloud agnostic settings for the NAT gateway itself
	// +kubebuilder:validation:Optional
	NAT *NetworkFabricGatewayNATOptions `json:"nat,omitempty"`
}

// NetworkFabricGatewayPrefix is used as a prefix to the gateway name
const NetworkFabricGatewayPrefix = "wf"

// NetworkFabricGatewayAssociation is used to define where the gateway should reside
type NetworkFabricGatewayAssociation struct {
	// Subnet is reference to the subnet the gateway should reside. Note the subnet must
	// be defined in the subnets sections below. Note, GCP does not require this settings,
	// CloudNAT is associated to a network
	// +kubebuilder:validation:Optional
	Subnet *string `json:"subnet,omitempty"`
	// Location defines the regional the gateway should reside in. This is only required
	// for GCP where cloud gateways are regional rather than designated to an availability
	// zone
	// +kubebuilder:validation:Optional
	Location *string `json:"location,omitempty"`
}

// NetworkFabricGatewayNATOptions provides the ability to configure cloud agnostic settings
// on the gateway
type NetworkFabricGatewayNATOptions struct {
	// Subnet defines the subnets which should be associated and natted through the nat gateway.
	// These must be defined in the subnets section within the network fabric spec.
	// +kubebuilder:validation:Optional
	Subnets []string `json:"subnets,omitempty"`
}

// NetworkFabricGatewayIPV4Mode defines the mode for ipv4 allocation on the gateway
type NetworkFabricGatewayIPV4Mode string

const (
	// NetworkFabricGatewayIPV4ModeAuto indicates the address is automatically assigned to the gateway
	NetworkFabricGatewayIPV4ModeAuto = "auto"
	// NetworkFabricGatewayIPV4ModeStatic indicates the gateway should be associated to a
	// statically created address
	NetworkFabricGatewayIPV4ModeStatic = "static"
	// NetworkFabricGatewayIPV4ModeAllocate indicates we should provision a static external address
	// for the nat gateway
	NetworkFabricGatewayIPV4ModeAllocate = "allocate"
)

// NetworkFabricGatewayIPV4 are the options related to ipv4 settings on a nat gateway
type NetworkFabricGatewayIPV4 struct {
	// Mode defines the mode of how to allocate external address or address pools to the
	// NAT gateway
	// +kubebuilder:validation:Required
	Mode NetworkFabricGatewayIPV4Mode `json:"mode"`
	// Addresses provides a collection of external addresses which should be associated to the
	// nat gateway, assuming the mode is static. In AWS is the an EIP which has been allocated
	// in the correct region and account. In GCP you can define mutiple external addresses which
	// the CloudNAT will manage
	// +kubebuilder:validation:Optional
	Addresses []string `json:"addresses,omitempty"`
}

// NetworkFabricSpec defines the desired state of a network
// +k8s:openapi-gen=true
type NetworkFabricSpec struct {
	// Layout refers to the layout of the network. It controls whether the
	// various parts - subnets, routes, etc. - are set up manually (i.e. specified
	// in the spec) or automatically (i.e. generated by the provider).
	// +kubebuilder:validation:Required
	Layout NetworkFabricLayout `json:"layout"`
	// CloudAccountRef is a reference to the cloud account that should be used to
	// create the network.
	// +kubebuilder:validation:Optional
	CloudAccountRef *cloudaccessv1alpha1.CloudAccountReference `json:"cloudAccountRef,omitempty"`
	// IPv4 contains the IPv4 configuration associated with the network.
	// +kubebuilder:validation:Required
	IPv4 NetworkFabricIPv4 `json:"ipv4"`
	// Location is the region the network should be created in.
	// [AWS] Region
	// [GCP] Networks do not have an associated region, so this will be set to "global"
	// [Azure] Region
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Location string `json:"location"`
	// Name is used to identify the network object in the cloud provider
	// [AWS] Names are not supported, so is a "Name" tag on the VPC
	// [GCP] Name of the Network
	// [Azure] Name of the Virtual Network (VNet)
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=40
	Name string `json:"name,omitempty"`
	// Plan currently refers to a pre-defined "flavour" of network, which is not
	// configurable by the caller. For example, when `.Spec.Layout.Mode="auto"`
	// and this is set to "EKS", a network suitable for use by EKS will be
	// generated automatically, with minimal input from the caller.
	//
	// In future, when "plans" in their traditional sense are supported for
	// NetworkFabrics, this will refer to the original plan that the network was
	// created from.
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=EKS;AKS;GKE
	Plan string `json:"plan,omitempty"`
	// Provider refers to the cloud provider.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=aws;gcp;azure
	Provider string `json:"provider"`
	// ProviderDetails defines cloud-specific network options
	// +kubebuilder:validation:Optional
	ProviderDetails *ProviderDetails `json:"providerDetails,omitempty"`
	// Private are options related to private networking
	// +kubebuilder:validation:Optional
	Private *NetworkFabricPrivateOptions `json:"private,omitempty"`
	// Routes is the list of routes within the network. Mode must be set to
	// "manual" for this field to be valid.
	// +kubebuilder:validation:Optional
	Routes []NetworkFabricRoute `json:"routes,omitempty"`
	// Stage is the name of the stage for the network.
	// +kubebuilder:validation:Required
	Stage string `json:"stage"`
	// Subnets is the list of subnets within the network. Mode must be set to
	// "manual" for this field to be valid.
	// +kubebuilder:validation:Optional
	Subnets []NetworkFabricSubnet `json:"subnets,omitempty"`
	// Tags is a collection of tags to apply to the resources associated with the
	// network, if applicable.
	// +kubebuilder:validation:Optional
	Tags map[string]string `json:"tags,omitempty"`
}

// GetSubnets returns a list of all the subnets on the network fabric
func (c *NetworkFabricSpec) GetSubnets() []string {
	var list []string

	if len(c.IPv4.CIDRBlocks.Strings()) > 0 {
		list = append(list, c.IPv4.CIDRBlocks.Strings()...)
	}

	for _, subnet := range c.Subnets {
		if subnet.IPv4.CIDRBlock.String() != "" {
			list = append(list, subnet.IPv4.CIDRBlock.String())
		}

		if subnet.IPv4.ProviderDetails != nil && subnet.IPv4.ProviderDetails.GCP != nil {
			for _, secondary := range subnet.IPv4.ProviderDetails.GCP.SecondaryIPRanges {
				list = append(list, secondary.CIDRBlock.String())
			}
		}
	}

	return list
}

// FindSubnetByName searches for an matching subnets in the spec
func (c *NetworkFabricSpec) FindSubnetByName(name string) (NetworkFabricSubnet, bool) {
	for _, subnet := range c.Subnets {
		if subnet.Name == name {
			return subnet, true
		}
	}

	return NetworkFabricSubnet{}, false
}

// FindSubnetByCIDR searches for an matching subnets in the spec
func (c *NetworkFabricSpec) FindSubnetByCIDR(cidr IPv4CIDR) (NetworkFabricSubnet, bool) {
	for _, subnet := range c.Subnets {
		if subnet.IPv4.CIDRBlock.String() == cidr.String() {
			return subnet, true
		}
	}

	return NetworkFabricSubnet{}, false
}

// FindGCPSecondaryByCIDR search the subnets for a range
func (c *NetworkFabricSpec) FindGCPSecondaryByCIDR(cidr IPv4CIDR, mode GCPSubnetDefaultSubnetName) (GCPSecondarySubnetIPRanges, bool) {
	if len(c.Subnets) == 0 {
		return GCPSecondarySubnetIPRanges{}, false
	}

	for _, subnet := range c.Subnets {
		if sub, found := subnet.FindGCPSecondaryByCIDR(cidr, mode); found {
			return sub, true
		}
	}

	return GCPSecondarySubnetIPRanges{}, false
}

// HasSecondaryIPRanges returns if found the subnet inside spec.subnets which matches the cidr
// range and the designation
func (c *NetworkFabricSubnet) HasSecondaryIPRanges() bool {
	switch {
	case c.IPv4.ProviderDetails == nil:
		return false
	case c.IPv4.ProviderDetails.GCP == nil:
		return false
	}

	return len(c.IPv4.ProviderDetails.GCP.SecondaryIPRanges) > 0
}

// FindGCPSecondaryByCIDR checks if the range exists in the subnet
func (c *NetworkFabricSubnet) FindGCPSecondaryByCIDR(cidr IPv4CIDR, mode GCPSubnetDefaultSubnetName) (GCPSecondarySubnetIPRanges, bool) {
	if !c.HasSecondaryIPRanges() {
		return GCPSecondarySubnetIPRanges{}, false
	}

	for _, x := range c.IPv4.ProviderDetails.GCP.SecondaryIPRanges {
		if x.CIDRBlock.String() == cidr.String() {
			return x, true
		}
	}

	return GCPSecondarySubnetIPRanges{}, false
}

// Locations returns a list of locations we subnets
func (c *NetworkFabricSpec) Locations() []string {
	if len(c.Subnets) <= 0 {
		return nil
	}
	var list []string

	maker := make(map[string]bool)

	for i := 0; i < len(c.Subnets); i++ {
		if !maker[c.Subnets[i].Location] {
			list = append(list, c.Subnets[i].Location)
			maker[c.Subnets[i].Location] = true
		}
	}

	return list
}

// CloudResourceName returns a cloud specific name
func (c *NetworkFabric) CloudResourceName() string {
	if c.Spec.Name != "" {
		return c.Spec.Name
	}

	return fmt.Sprintf("%s-%s", c.Namespace, c.Name)
}

// IsPrivateNetwork checks if the network is designated as private
func (c *NetworkFabric) IsPrivateNetwork() bool {
	if c.Spec.Private != nil && c.Spec.Private.Enabled != nil && *c.Spec.Private.Enabled {
		return true
	}

	return false
}

// ProviderDetails defines the parameters for cloud specific options - i.e
// options which cannot be consolidated as they are too specific to the
// chosen cloud vendor
// +union
type ProviderDetails struct {
	// Type represents the cloud the NetworkFabric belongs to
	// +unionDiscriminator
	// +kubebuilder:validation:Required
	// +required
	Type NetworkFabricProviderType `json:"type"`
	// AWS is the provider specification for AWS networks
	// +kubebuilder:validation:Optional
	AWS *ProviderDetailsAWS `json:"aws,omitempty"`
	// Azure is the provider specification for Azure networks
	// +kubebuilder:validation:Optional
	Azure *ProviderDetailsAzure `json:"azure,omitempty"`
	// GCP is the provider specification for GCP networks
	// +kubebuilder:validation:Optional
	GCP *ProviderDetailsGCP `json:"gcp,omitempty"`
}

// NetworkFabricLayout defines the options for wayfinder prescribed network topology
type NetworkFabricLayout struct {
	// Mode dictates whether the layout of the network should be set up manually
	// or automatically. If automatic, no other fields should be specified. If
	// manual, fields should be specified manually by the caller.
	//
	// Currently, only "auto" is supported. In future, "manual" will be
	// implemented as required.
	//
	// +kubebuilder:validation:Required
	Mode NetworkFabricLayoutMode `json:"mode"`
	// PrivateSubnets is the number of private subnets that should be generated
	// in the network. Mode must be set to "auto" for this field to be valid.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=3
	PrivateSubnets int `json:"privateSubnets,omitempty"`
	// PublicSubnets is the number of public subnets that should be generated
	// in the network. Mode must be set to "auto" for this field to be valid.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=3
	PublicSubnets int `json:"publicSubnets,omitempty"`
}

// NetworkFabricIPv4 is a collection of network ipv4 ranges
type NetworkFabricIPv4 struct {
	// CIDRBlocks is a list of CIDR blocks that should be associated with the network.
	// [AWS] Multiple IP ranges. AWS only supports a single IP range is supported on
	//       *creation*, but can be updated afterwards to specify additional IP ranges.
	// [GCP] Unsupported.
	// [Azure] Multiple IP ranges.
	// +kubebuilder:validation:Optional
	CIDRBlocks IPv4CIDRBlocks `json:"cidrBlocks"`
}

// NetworkFabricRoute is a cloud agnostic definition for a route
type NetworkFabricRoute struct {
	// Name is the name of the route.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Description is an optional description of the route.
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`

	// CIDRBlock represents the range of destination IP addresses that this route
	// applies to.
	// +kubebuilder:validation:Required
	CIDRBlock IPv4CIDR `json:"cidrBlock"`
	// Target is the destination that the traffic bound for IP addresses
	// within CIDRBlock range will be sent to. This target may be a gateway, network
	// interface, or connection through which to send the destination traffic;
	// for example, an internet gateway.
	// +kubebuilder:validation:Required
	Target NetworkFabricRouteTarget `json:"target"`
}

// NetworkFabricRouteTarget contains the information necessary to determine the
// destination that network traffic should be sent to.
//
// TODO: Determine what goes here (IPv4 vs IPv6 routing, local, internet gateway, NAT gateway)
type NetworkFabricRouteTarget struct{}

// NetworkFabricSubnet defines the options for a virtual subnet
type NetworkFabricSubnet struct {
	// Name is the name of the subnet.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Description is an optional description of the subnet.
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`

	// Location is the zone or region associated with the subnet.
	// [AWS] Zone
	// [GCP] Region
	// [Azure] Region (same as the Virtual Network)
	// +kubebuilder:validation:Required
	Location string `json:"location"`
	// IPv4 is the Internet Protocol (version 4) configuration for the subnet.
	// +kubebuilder:validation:Required
	IPv4 NetworkFabricSubnetIPv4 `json:"ipv4"`
}

// NetworkFabricSubnetIPv4 defines the options for the subnet range
type NetworkFabricSubnetIPv4 struct {
	// CIDRBlock is the IP address range for the subnet.
	// +kubebuilder:validation:Required
	CIDRBlock IPv4CIDR `json:"cidrBlock"`
	// IPVersion is the Internet Protocol version of the subnet.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=4;6
	// +kubebuilder:default=4
	IPVersion uint16 `json:"ipVersion,omitempty"`
	// Type determines whether VMs launched into this subnet should have public or
	// private IP address. If `nat` is specified, this must be set to "private".
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=private
	Type NetworkFabricSubnetType `json:"type"`

	// ProviderDetails is is the cloud specific configuration for the subnet.
	// +kubebuilder:validation:Optional
	ProviderDetails *ProviderDetailsSubnetIPv4 `json:"providerDetails,omitempty"`
}

// ProviderDetailsSubnetIPv4 provides cloud provider specifics for the subnet
type ProviderDetailsSubnetIPv4 struct {
	// GCP is the provider specification for GCP subnets
	GCP *ProviderDetailsSubnetIPv4GCP `json:"gcp,omitempty"`
}

// ProviderDetailsSubnetIPv4GCP provides secondary ranges for gcp
type ProviderDetailsSubnetIPv4GCP struct {
	SecondaryIPRanges []GCPSecondarySubnetIPRanges `json:"secondaryIpRanges"`
}

// GCPSubnetDefaultSubnetName represents the default subnets names for GKE private clusters
// +kubebuilder:validation:Enum=pods;services
type GCPSubnetDefaultSubnetName string

// String returns the string representation of the type
func (g GCPSubnetDefaultSubnetName) String() string {
	return string(g)
}

const (
	// GCPSubnetDefaultSubnetNamePods represents the default subnet name for GCP pods networks
	GCPSubnetDefaultSubnetNamePods GCPSubnetDefaultSubnetName = "pods"
	// GCPSubnetDefaultSubnetNameServices represents the default subnet name for GCP services networks
	GCPSubnetDefaultSubnetNameServices GCPSubnetDefaultSubnetName = "services"
)

// GCPSecondarySubnetIPRanges represents the secondary subnet ranges for GCP when running in private cluster mode
type GCPSecondarySubnetIPRanges struct {
	// CIDRBlock is the IP address range for the subnet.
	// +kubebuilder:validation:Required
	CIDRBlock IPv4CIDR `json:"cidrBlock"`
	// Type determines what type of secondary range ip address this is.
	// +kubebuilder:validation:Required
	Type GCPSubnetDefaultSubnetName `json:"type"`
}

// NetworkFabricStatus defines the observed state of a network
// +k8s:openapi-gen=true
type NetworkFabricStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`

	// CloudAccountRef is a reference to the cloud account to use to retrieve
	// credentials for this network. Will be populated if the spec specifies a
	// CloudAccount as the credential object.
	// +kubebuilder:validation:Optional
	CloudAccountRef *cloudaccessv1alpha1.CloudAccountReference `json:"cloudAccount,omitempty"`

	// AWS contains the AWS-specific state for the network
	AWS *NetworkFabricStatusAWS `json:"aws,omitempty"`
	// Azure contains the Azure-specific state for the network
	Azure *NetworkFabricStatusAzure `json:"azure,omitempty"`
	// GCP contains the GCP-specific state for the network
	GCP *NetworkFabricStatusGCP `json:"gcp,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkFabricList contains a list of networks
type NetworkFabricList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkFabric `json:"items"`
}

func (NetworkFabricList) ObjectType() corev1alpha1.Object {
	return &NetworkFabric{}
}
func (l *NetworkFabricList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *NetworkFabricList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*NetworkFabricList))
}

// GetItems returns the items as abstract client.Object
func (l *NetworkFabricList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

// SetItems sets the items
func (l *NetworkFabricList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []NetworkFabric{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*NetworkFabric)))
	}
}

// Ownership returns a ownership resource from the network definition
func (c *NetworkFabric) Ownership() corev1alpha1.Ownership {
	return corev1alpha1.Ownership{
		Group:     GroupVersion.Group,
		Version:   GroupVersion.Version,
		Kind:      NetworkFabricKind,
		Namespace: c.Namespace,
		Name:      c.Name,
	}
}

// GetCommonStatus implements CommonStatusAware for NetworkFabric
func (c *NetworkFabric) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}
