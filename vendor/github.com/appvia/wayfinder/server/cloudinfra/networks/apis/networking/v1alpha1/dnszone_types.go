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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	cloudaccessv1alpha1 "github.com/appvia/wayfinder/server/cloudaccess/apis/cloudaccess/v1alpha1"
)

const DNSZoneKind = "DNSZone"

func DNSZoneGroupVersionKind() schema.GroupVersionKind {
	return schema.FromAPIVersionAndKind(GroupVersion.String(), DNSZoneKind)
}

func NewDNSZone(name, namespace string) *DNSZone {
	return &DNSZone{
		TypeMeta: metav1.TypeMeta{
			Kind:       DNSZoneKind,
			APIVersion: GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

// NewGlobalOrScopedDNSZone returns either a global or workspce-scoped DNSZone object depending on
// whether the namespace is populated in the DNSZoneRef
func NewGlobalOrScopedDNSZone(ref DNSZoneRef) GlobalOrScopedDNSZone {
	if ref.Namespace == "" {
		return NewGlobalDNSZone(ref.Name)
	}
	return NewDNSZone(ref.Name, ref.Namespace)
}

// GlobalOrScopedDNSZone provides a common interface for global or workspace-scoped DNS zones
// +kubebuilder:object:generate=false
type GlobalOrScopedDNSZone interface {
	corev1alpha1.Object
	corev1alpha1.Allocatable
	// IsGlobal returns true if this is a global zone (i.e. can safely be casted to a GlobalDNSZone)
	IsGlobal() bool
	// CloudResourceName defines the cloud-specific name of the resource created for this DNS zone,
	// if applicable to the provider
	CloudResourceName() string
	// GetZoneSpec gets the specification for this DNS zone
	GetZoneSpec() *DNSZoneSpec
	// GetZoneStatus gets the status for this DNS zone
	GetZoneStatus() *DNSZoneStatus
	// GetRef returns a reference to this DNS zone (for use as a parent zone reference)
	GetRef() *DNSZoneRef
}

// SortZonesByDomain is a list of global or scoped DNS zones which can be sorted by the spec domain
// +kubebuilder:object:generate=false
type SortZonesByDomain []GlobalOrScopedDNSZone

func (s SortZonesByDomain) Len() int {
	return len(s)
}

func (s SortZonesByDomain) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s SortZonesByDomain) Less(i, j int) bool {
	return s[i].GetZoneSpec().Domain < s[j].GetZoneSpec().Domain
}

// SortDNSZonesByDomain is a list of DNS zones which can be sorted by the spec domain
// +kubebuilder:object:generate=false
type SortDNSZonesByDomain []DNSZone

func (s SortDNSZonesByDomain) Len() int {
	return len(s)
}

func (s SortDNSZonesByDomain) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s SortDNSZonesByDomain) Less(i, j int) bool {
	return s[i].Spec.Domain < s[j].Spec.Domain
}

// Check implementation:
var _ GlobalOrScopedDNSZone = &DNSZone{}

// +kubebuilder:webhook:name=dnszones.networking.appvia.io,mutating=false,path=/validate/networking.appvia.io/dnszones,verbs=create;update;delete,groups="networking.appvia.io",resources=dnszone,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=dnszones.networking.appvia.io,mutating=true,path=/mutate/networking.appvia.io/dnszones,verbs=create;update,groups="networking.appvia.io",resources=dnszone,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DNSZone represents a DNS zone in a specific cloud provider DNS implementation
// cloud account
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=dnszone,scope=Namespaced,categories={wayfinder}
type DNSZone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DNSZoneSpec   `json:"spec,omitempty"`
	Status DNSZoneStatus `json:"status,omitempty"`
}

func (DNSZone) APIPath() string {
	return "dnszones"
}
func (c *DNSZone) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *DNSZone) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*DNSZone))
}
func (DNSZone) ListType() corev1alpha1.ObjectList {
	return &DNSZoneList{}
}

// GetCommonStatus implements CommonStatusAware
func (c *DNSZone) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// IsGlobal implements GlobalOrScopedDNSZone
func (c *DNSZone) IsGlobal() bool {
	return false
}

// GetZoneSpec implements GlobalOrScopedDNSZone
func (c *DNSZone) GetZoneSpec() *DNSZoneSpec {
	return &c.Spec
}

// GetZoneStatus implements GlobalOrScopedDNSZone
func (c *DNSZone) GetZoneStatus() *DNSZoneStatus {
	return &c.Status
}

// GetRef implements GlobalOrScopedDNSZone
func (c *DNSZone) GetRef() *DNSZoneRef {
	return &DNSZoneRef{
		Namespace: c.Namespace,
		Name:      c.Name,
	}
}

// AllocatedToWorkspace implements the Allocatable interface for DNSZone, it will return true if the
// specified workspace is allowed to use this DNSZone according to the allocation settings on
// the spec.
func (c *DNSZone) AllocatedToWorkspace(ws corev1alpha1.WorkspaceKey) bool {
	// DNSZones are not allocatable, only GlobalDNSZones are allocatable, therefore this applies
	// only to its own workspace
	return corev1alpha1.Workspace(c) == ws
}

// GetStatus returns the current overall status
func (c DNSZone) GetStatus() corev1alpha1.Status {
	return c.Status.Status
}

// +k8s:openapi-gen=true
type DNSZoneRef struct {
	// Namespace which contains the DNSZone, leave empty if this is a reference to a GlobalDNSZone
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
	// Name of the DNSZone or GlobalDNSZone
	Name string `json:"name"`
}

// Is returns true if this reference points at the passed-in zone
func (r *DNSZoneRef) Is(z GlobalOrScopedDNSZone) bool {
	if r == nil {
		return false
	}
	return r.Namespace == z.GetNamespace() && r.Name == z.GetName()
}

// DNSZoneSpec defines the specification of a DNS zone which should be provisioned
// +k8s:openapi-gen=true
type DNSZoneSpec struct {
	// Provider defines which DNS provider to use to create this zone
	// +kubebuilder:validation:Enum=aws;gcp;azure
	// +kubebuilder:validation:Required
	Provider string `json:"provider"`
	// CloudAccountRef is a reference to the cloud account that should be used to
	// create the DNS zone. This may not be required for all DNS providers.
	// +kubebuilder:validation:Optional
	CloudAccountRef *cloudaccessv1alpha1.CloudAccountReference `json:"cloudAccountRef,omitempty"`
	// ParentZone should be set to make this zone a child of another zone managed by Wayfinder.
	// Setting this allows Wayfinder to automatically manage the delegation of this zone.
	// +kubebuilder:validation:Optional
	ParentZone *DNSZoneRef `json:"parentZone,omitempty"`
	// Domain is the domain name that this zone will represent
	// +kubebuilder:validation:Required
	Domain string `json:"domain"`
	// Private indicates that this is a private DNS zone, if applicable for the provider
	// +kubebuilder:validation:Optional
	Private bool `json:"private"`
	// Network should be specified if private is true to indicate what network this private DNS
	// zone should be provisioned with. Ignored if Private is false/unspecified.
	// +kubebuilder:validation:Optional
	Network *corev1alpha1.Ownership `json:"network,omitempty"`
	// ProviderDetails provides additional fields which can be used for DNS-provider specific
	// data needed to provision this zone
	// +kubebuilder:validation:Optional
	ProviderDetails *DNSZoneProviderDetails `json:"providerDetails,omitempty"`
	// CloudResourceName specifies the name of the DNS zone in the DNS provider
	// Can be left blank so that the name is derived from the resource name
	// +kubebuilder:validation:Optional
	CloudResourceName string `json:"cloudResourceName,omitempty"`
	// Unmanaged should be set to true to indicate that this zone should not be built or deleted
	// by Wayfinder, but should be verified to exist in the specified DNS provider and, thus, will
	// be usable as a parent zone.
	//
	// Ensure CloudResourceName is also set if this provider requires a name (other than the domain)
	// to find this unmanaged zone, e.g. GCP's 'Zone name'.
	// +kubebuilder:validation:Optional
	Unmanaged bool `json:"unmanaged,omitempty"`
	// AvailableToClusters allows this zone, or automatically-created child zones of it, to be made
	// available automatically in the targeted clusters, with ExternalDNS deployed and configured.
	//
	// Leave unspecified to not make this zone available in any clusters.
	//
	// For a GlobalDNSZone, this provide DNS in *any* workspace (subject to the label selectors you
	// define *and* the allocation specified in the Allocation field).
	//
	// For a DNSZone, this will only provide DNS for clusters in the *same* workspace the zone is
	// created in.
	// +kubebuilder:validation:Optional
	AvailableToClusters *DNSZoneClusterAvailability `json:"availableToClusters,omitempty"`
	// ManualChildZoneCreation defines whether workspaces should be able to manually provision child
	// DNS zones within a cluster
	//
	// Leave unspecified to prevent the manual creation in clusters.
	// +kubebuilder:validation:Optional
	ManualChildZoneCreation bool `json:"manualChildZoneCreation,omitempty"`
	// Allocation defines which workspaces can create child zones from this DNS zone. Only
	// applicable to GlobalDNSZones. This must be set for AutoProvisionChildZones to have any
	// effect.
	// +kubebuilder:validation:Optional
	Allocation *corev1alpha1.ResourceAllocation `json:"allocation,omitempty"`
}

// DNSZoneClusterAvailabilityMode are the ways a DNS zone can be made available to clusters
type DNSZoneClusterAvailabilityMode string

const (
	// DNSZoneClusterAvailabilityModeDirect will make the zone *directly* available in the targeted
	// clusters - this mode is not available for global DNS zones
	DNSZoneClusterAvailabilityModeDirect DNSZoneClusterAvailabilityMode = "direct"
	// DNSZoneClusterAvailabilityModeCreateChildZone will auto-provision child zones of this zone
	// dedicated to each targeted cluster
	DNSZoneClusterAvailabilityModeCreateChildZone DNSZoneClusterAvailabilityMode = "createChildZone"
)

type DNSZoneClusterAvailability struct {
	// Mode defines how this zone will be made available in the targeted clusters. If 'direct',
	// this zone will be directly available in the targeted clusters using External DNS. If
	// 'createChildZone', child zones of this zone will be automatically created and made available
	// in the targeted clusters, using the CustomNamingRule if specified.
	//
	// For 'direct' mode, the zone can only be made available in clusters of the relevant type for
	// the provider of the DNS zone (e.g. AWS Route 53 zones can be made available directly in AWS
	// EKS clusters). This restriction does not apply for 'createChildZone' mode.
	//
	// 'direct' mode is only supported for workspace-scoped DNS zones, global DNS zones cannot be
	// made directly.
	// +kubebuilder:validation:Enum=createChildZone;direct
	// +kubebuilder:validation:Required
	Mode DNSZoneClusterAvailabilityMode `json:"mode"`

	// Selectors define which clusters the zone should be available in
	// +kubebuilder:validation:Required
	Selectors metav1.LabelSelector `json:"selectors"`

	// CustomNamingRule defines a non-default naming pattern for the child zone. You can use the
	// following as placeholders in a custom rule:
	//
	// {workspace}     The key of the workspace that owns the cluster
	// {clustername}   The metadata name of the cluster
	// {stage}         The stage the cluster is in
	//
	// If unspecified, the default {workspace}-{clustername} will be used.
	//
	// If you choose a custom naming rule, be careful to ensure that multiple clusters don't create
	// overlapping ingress names within in the same DNS zone.
	//
	// This is not valid for mode 'direct'
	// +kubebuilder:validation:Optional
	CustomNamingRule string `json:"customNamingRule,omitempty"`
}

// IsDirect is a convenience wrapper to determine if the provided DNS zone is directly targeted to a
// cluster
func IsDirect(z GlobalOrScopedDNSZone) bool {
	return !z.IsGlobal() && z.GetZoneSpec().AvailableToClusters != nil && z.GetZoneSpec().AvailableToClusters.Mode == DNSZoneClusterAvailabilityModeDirect
}

// NameChild uses the any specified custom naming rule, or the default naming convention, to name a
// child zone for a workspace, cluster and stage
func (ap DNSZoneClusterAvailability) NameChild(ws corev1alpha1.WorkspaceKey, clusterName string, stage string) string {
	namePattern := "{workspace}-{clustername}"
	if ap.CustomNamingRule != "" {
		namePattern = ap.CustomNamingRule
	}
	rp := strings.NewReplacer("{workspace}", ws.Key(), "{clustername}", clusterName, "{stage}", stage)
	return rp.Replace(namePattern)
}

// DNSZoneType represents the concrete type of a DNS Zone to provide
// +kubebuilder:validation:Enum=AWS;Azure;GCP
type DNSZoneType string

const (
	// DNSZoneTypeAWS is for AWS Route 53 zone
	DNSZoneTypeAWS DNSZoneType = "AWS"
	// DNSZoneTypeAzure is for Azure DNS zone
	DNSZoneTypeAzure DNSZoneType = "Azure"
	// DNSZoneTypeGCP is for GCP Cloud DNS zone
	DNSZoneTypeGCP DNSZoneType = "GCP"
)

// DNSZoneProviderDetails provides parameters that are specific to a particular type of
// DNS zone
// +union
type DNSZoneProviderDetails struct {
	// +unionDiscriminator
	// +kubebuilder:validation:Required
	// +required
	Type DNSZoneType `json:"type"`

	// AWS holds parameters specific to an AWS DNS zone. Present only if
	// type is AWS.
	// +optional
	AWS *AWSDNSZoneParameters `json:"aws,omitempty"`

	// GCP holds parameters specific to a GCP DNS zone. Present only if
	// type is GCP.
	// +optional
	GCP *GCPDNSZoneParameters `json:"gcp,omitempty"`

	// Azure holds parameters specific to an Azure DNS zone. Present only if
	// type is Azure.
	// +optional
	Azure *AzureDNSZoneParameters `json:"azure,omitempty"`
}

// AWSDNSZoneParameters is the parameters for an AWS DNS zone
type AWSDNSZoneParameters struct{}

// GCPDNSZoneParameters is the parameters for a GCP DNS zone
type GCPDNSZoneParameters struct{}

// AzureDNSZoneParameters is the parameters for an Azure DNS zone
type AzureDNSZoneParameters struct {
	// ResourceGroup identifies an *existing* resource group in which to place this DNS zone. If
	// this is unpopulated, a new resource group will be created for the zone.
	// +kubebuilder:validation:Optional
	ResourceGroup string `json:"resourceGroup,omitempty"`
}

// DNSZoneStatus defines the status of a cloud account
// +k8s:openapi-gen=true
type DNSZoneStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// Nameservers are the authoritative nameservers that are required to be set for this zone.
	// +kubebuilder:validation:Optional
	Nameservers []string `json:"nameservers"`
	// Delegated indicates this zone has been correctly delegated
	// +kubebuilder:validation:Optional
	Delegated bool `json:"delegated"`
	// VerificationRecord contains a txt record if needed for verification
	// +kubebuilder:validation:Optional
	VerificationRecord string `json:"txtrecord,omitempty"`
	// ZoneID contains a DNS-provider specific unique reference to the zone created for this
	// resource. Azure - ResourceID, AWS - ZoneID, GCP - ZoneName
	// +kubebuilder:validation:Optional
	ZoneID string `json:"zoneID,omitempty"`
}

// CloudResourceName returns the DNS provider name for this DNS zone
func (c *DNSZone) CloudResourceName() string {
	if c.Spec.CloudResourceName != "" {
		return c.Spec.CloudResourceName
	}
	return fmt.Sprintf("%s-%s", c.Namespace, c.Name)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DNSZoneList provides a list of workload identities
type DNSZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DNSZone `json:"items"`
}

func (DNSZoneList) ObjectType() corev1alpha1.Object {
	return &DNSZone{}
}
func (l *DNSZoneList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *DNSZoneList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*DNSZoneList))
}

func (l *DNSZoneList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *DNSZoneList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []DNSZone{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*DNSZone)))
	}
}

type DNSRecordType string

const (
	// DNSRecordTypeNS is an ns record
	DNSRecordTypeNS DNSRecordType = "NS"
	// DNSRecordTypeTXT is a txt record
	DNSRecordTypeTXT DNSRecordType = "TXT"
)

type DNSRecord struct {
	RecordType DNSRecordType
	Records    []string
}
