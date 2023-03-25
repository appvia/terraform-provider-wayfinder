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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	cloudaccessv2 "github.com/appvia/wayfinder/server/cloudaccess/apis/cloudaccess/v2beta1"
)

// PeeringRuleKind is the kind for a PeeringRule
const PeeringRuleKind = "PeeringRule"

// PeeringRuleGVK is the GVK for a Peering
var PeeringRuleGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    PeeringRuleKind,
}

// NewPeeringRule is called to return an empty policy
func NewPeeringRule() *PeeringRule {
	return &PeeringRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       PeeringRuleKind,
		},
	}
}

// +kubebuilder:webhook:name=peeringrules.networking.appvia.io,mutating=false,path=/validate/networking.appvia.io/peeringrules,verbs=create;update;delete,groups="networking.appvia.io",resources=peeringrules,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=peeringrules.networking.appvia.io,mutating=true,path=/mutate/networking.appvia.io/peeringrules,verbs=create;update,groups="networking.appvia.io",resources=peeringrules,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PeeringRule providing a policy definition for peering
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=peeringrules,scope=Cluster,categories={wayfinder}
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
type PeeringRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PeeringRuleSpec   `json:"spec,omitempty"`
	Status PeeringRuleStatus `json:"status,omitempty"`
}

func (PeeringRule) APIPath() string {
	return "peeringrules"
}
func (p *PeeringRule) Clone() corev1alpha1.Object {
	return p.DeepCopy()
}
func (p *PeeringRule) CloneInto(o corev1alpha1.Object) {
	p.DeepCopyInto(o.(*PeeringRule))
}
func (PeeringRule) ListType() corev1alpha1.ObjectList {
	return &PeeringRuleList{}
}

func (p *PeeringRule) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &p.Status.CommonStatus
}

// PeeringRuleConnectionType represents the concrete type for configuration
// +kubebuilder:validation:Enum=peering;gateway
type PeeringRuleConnectionType string

const (
	// RuleConnectionTypeGateway indicates we are using a gateway
	RuleConnectionTypeGateway = "gateway"
	// RuleConnectionTypePeering indicates we are using a network peering mode
	RuleConnectionTypePeering = "peering"
)

// SupportedRuleConnectionTypes are the support types
var SupportedRuleConnectionTypes = []string{
	RuleConnectionTypeGateway,
	RuleConnectionTypePeering,
}

// PeeringSubnetFilter provides a means to filter down which subnets in the
// virtual network we are about to attach should be pushed down the peer. One
// example would be to only push internal subnets.
type PeeringSubnetFilter map[string]string

// ExternalNetworkPeer defines the definition when attached to the network
// to an external network
type ExternalNetworkPeer struct {
	// Account is a cloud agnostic name of the account, subscription or project
	// where the network we are peering to exists
	// +kubebuilder:validation:Required
	Account string `json:"account"`
	// Location is the region where the network exists. For AWS this might be
	// eu-west-2, for GCP europe-west2 and so forth
	// +kubebuilder:validation:Required
	Location string `json:"location"`
	// Identifier is the full resource identity of the virtual network which
	// we are peering to. This AWS and GCP this would be the virtual network name.
	// For Azure this would be the resource group plus the virtual network name.
	// +kubebuilder:validation:Required
	Identifier string `json:"identifier"`
	// Routes is a collection of network ranges which we want to expose to
	// the peered networks. The route tables of the source networks are automatically
	// amended to push these subnets down the peered connection
	// +kubebuilder:validation:Optional
	Routes IPv4CIDRs `json:"routes,omitempty" type:"[]string"`
	// RouteTableSelectors is required when enableAutoApproval is enabled. The field
	// provides a collection of cloud tags which is used to filter on which routing tables
	// in the external network need updating to include the source network routes.
	// Note, this field is not required for GCP or Azure as the route propagation
	// is performed automatically for you.
	// +kubebuilder:validation:Optional
	RouteTableSelectors map[string]string `json:"routeTableSelectors,omitempty"`
}

// PeeringRuleConnection provides the definitions for the connection details related
// to peering
type PeeringRuleConnection struct {
	// Type is the type of peering we are configuring
	// +unionDiscriminator
	// +kubebuilder:validation:Required
	Type PeeringRuleConnectionType `json:"type"`
	// Geteway provides the configuration for working with gateways and peering
	// connections via a network backbone such as Transit Gateway.
	// +kubebuilder:validation:Optional
	Gateway *PeeringRuleGateway `json:"gateway,omitempty"`
	// Peering provides the configuration for direct peering between two networks.
	// By default the peering assumes the management network, though this can be
	// overridden if required
	// +kubebuilder:validation:Optional
	Peering *PeeringRuleDirect `json:"peering,omitempty"`
}

// IsPeering returns true is the peering a vpc network peering
func (p *PeeringRuleConnection) IsPeering() bool {
	return p.Type == RuleConnectionTypePeering && p.Peering != nil
}

// IsGatewayAttachment returns true of the peering a gateway attachment
func (p *PeeringRuleConnection) IsGatewayAttachment() bool {
	return p.Type == RuleConnectionTypeGateway && p.Gateway != nil
}

// PeeringRuleDirect defines the options around direct peering
type PeeringRuleDirect struct {
	// EnableAutoApproval indicates we should always accept the peering connection
	// on the other end of the peer. This requires the user provides a cloud access config
	// which the correct permissions to do so.
	// +kubebuilder:validation:Optional
	EnableAutoApproval *bool `json:"enableAutoApproval,omitempty"`
	// Network provides the ability to override the network which the peering
	// is created on. By default this is the management cluster.
	// +kubebuilder:validation:Optional
	Network *ExternalNetworkPeer `json:"network,omitempty"`
	// Subnets provides a means to filter which on the subnets you want to push
	// down the peering. By default we assume all the subnets attached to the virtual
	// network should be routed down the peer
	// +kubebuilder:validation:Optional
	Subnets PeeringSubnetFilter `json:"subnets,omitempty"`
	// ProviderDetails provides the cloud specific options when performing a gateway
	// attachment
	// +kubebuilder:validation:Optional
	ProviderDetails *PeeringProviderDetails `json:"providerDetails,omitempty"`
}

// PeeringRuleGateway defines the options for gateway attachments and peering.
type PeeringRuleGateway struct {
	// Identifier is the cloud specific identifier for the gateway - this could be
	// a transit gateway id in AWS.
	// +kubebuilder:validation:Required
	Identifier string `json:"identifier"`
	// EnableAutoApproval indicates we should always accept the peering connection
	// on the other end of the peer. This requires the user provides a cloud access config
	// which the correct permissions to do so.
	// +kubebuilder:validation:Optional
	EnableAutoApproval *bool `json:"enableAutoApproval,omitempty"`
	// Location is the cloud region which where the transit gateway resides
	// +kubebuilder:validation:Optional
	Location string `json:"location,omitempty"`
	// Routes is a collection of cidr blocks which we need to push down the transit
	// gateway.
	// +kubebuilder:validation:Required
	Routes IPv4CIDRs `json:"routes" type:"[]string"`
	// RouteTableSelectors is required when enableAutoApproval is enabled. The field
	// provides a collection of cloud tags which is used to filter on which routing tables
	// in the external network need updating to include the source network routes.
	// +kubebuilder:validation:Optional
	RouteTableSelectors map[string]string `json:"routeTableSelectors,omitempty"`
	// ProviderDetails provides the cloud specific options when performing a gateway
	// attachment
	// +kubebuilder:validation:Optional
	ProviderDetails *PeeringGatewayProviderDetails `json:"providerDetails,omitempty"`
}

// PeeringProviderDetails provides a means configure cloud specific options around
// gateway attachments
type PeeringProviderDetails struct {
	// Azure defines the cloud specifics for gateway options
	Azure *PeeringProviderDetailsAzure `json:"azure,omitempty"`
}

// PeeringGatewayProviderDetails provides a means configure cloud specific options around
// gateway attachments
type PeeringGatewayProviderDetails struct {
	// AWS defines the cloud specifics for gateway options
	// +kubebuilder:validation:Optional
	AWS *PeeringGatewayProviderDetailsAWS `json:"aws,omitempty"`
}

// PeeringProviderDetailsAzure are cloud specific options for Azure
type PeeringProviderDetailsAzure struct {
	// EnableUseRemoteGateway indicates if cluster peering should use remote gateway.
	// If set to true, local gateway will not be deployed and remote one configured
	// to be used
	// +kubebuilder:validation:Optional
	EnableUseRemoteGateway bool `json:"enableUseRemoteGateway,omitempty"`
}

// PeeringGatewayProviderDetailsAWS are cloud specific options for AWS
type PeeringGatewayProviderDetailsAWS struct {
	// EnableDNS indicates we should enable or disable dns support via the gateway. Note this
	// defaults to true unless defined.
	// +kubebuilder:validation:Optional
	EnableDNS *bool `json:"enableDNS,omitempty"`
}

// PeeringRuleFilters is used to filter down whom the policy should apply
type PeeringRuleFilters struct {
	// Allocation offers the ability to filter the peering policy down to a
	// collection of workspaces only
	// +kubebuilder:validation:Required
	Allocation corev1alpha1.ResourceAllocation `json:"allocation"`
	// Selectors offers the option to filter down which networks the peering
	// policy is applied based on the labels on the network fabric resource.
	// +kubebuilder:validation:Optional
	Selectors metav1.LabelSelector `json:"selectors,omitempty"`
}

// PeeringRuleSpec provides the definition for a peering rule. These are matched
// against one of more Network Fabric CRD and used to provision a Peer CRD which
// is used to connect up networks
type PeeringRuleSpec struct {
	// CloudAccessConfigRef provides an optional reference to a cloudaccount which has the
	// permission to carry out the tasks required to fulfil peering or gateway
	// attachments
	// +kubebuilder:validation:Optional
	CloudAccessConfigRef *cloudaccessv2.CloudAccessConfigReference `json:"cloudAccessConfigRef,omitempty"`
	// Connection provides the policy details around how the peering should
	// be achieved
	// +kubebuilder:validation:Required
	Connection PeeringRuleConnection `json:"connection"`
	// Filters provides the ability to apply a collection of filters to whom the
	// peering policy should apply to
	// +kubebuilder:validation:Required
	Filters *PeeringRuleFilters `json:"filters"`
}

// PeeringRuleStatus defines the observed state of a peering setup
// +k8s:openapi-gen=true
type PeeringRuleStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// ConnectionID is the ID of the peering or gateway attachment which has been
	// created
	ConnectionID *string `json:"connectionID,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PeeringRuleList provides a list of resources
type PeeringRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PeeringRule `json:"items"`
}

func (PeeringRuleList) ObjectType() corev1alpha1.Object {
	return &PeeringRule{}
}
func (l *PeeringRuleList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *PeeringRuleList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*PeeringRuleList))
}

// GetItems returns the items
func (l *PeeringRuleList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

// SetItems set them
func (l *PeeringRuleList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []PeeringRule{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*PeeringRule)))
	}
}
