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
	"k8s.io/apimachinery/pkg/types"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

const CloudAccessConfigKind = "CloudAccessConfig"

func CloudAccessConfigGroupVersionKind() schema.GroupVersionKind {
	return schema.FromAPIVersionAndKind(GroupVersion.String(), CloudAccessConfigKind)
}

// CloudAccountOrOrg allows the role management functionality to operate for both cloud orgs and
// cloud access configs without worry
// +kubebuilder:object:generate=false
type CloudAccountOrOrg interface {
	corev1alpha1.Object
	Cloud() string
	Identifier() string
	OrgIdentifier() string
	DefaultRegion() string
	GetRoles() []CloudAccessConfigRole
	SetAssumeProviderRole(role string, assumeProviderRole string) bool
	SetDeployedRoleHash(role string, hash string) bool
	SetIdentity(identity CloudIdentityReference) bool
	Stage() string
}

func GetRoleCloudResourceName(a CloudAccountOrOrg, role string) string {
	if a == nil {
		return ""
	}
	for _, r := range a.GetRoles() {
		if r.Role == role {
			return r.CloudResourceName
		}
	}
	return ""
}

// +kubebuilder:webhook:name=cloudaccessconfigs.cloudaccess.appvia.io,mutating=false,path=/validate/cloudaccess.appvia.io/cloudaccessconfigs,verbs=create;update;delete,groups="cloudaccess.appvia.io",resources=cloudaccessconfigs,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=cloudaccessconfigs.cloudaccess.appvia.io,mutating=true,path=/mutate/cloudaccess.appvia.io/cloudaccessconfigs,verbs=create;update,groups="cloudaccess.appvia.io",resources=cloudaccessconfigs,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudAccessConfig represents an account/project/subscription in a cloud provider which
// Wayfinder has access to
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=cloudaccessconfigs,scope=Namespaced,categories={wayfinder}
type CloudAccessConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudAccessConfigSpec   `json:"spec,omitempty"`
	Status CloudAccessConfigStatus `json:"status,omitempty"`
}

func (CloudAccessConfig) APIPath() string {
	return "cloudaccessconfigs"
}
func (c *CloudAccessConfig) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *CloudAccessConfig) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*CloudAccessConfig))
}
func (CloudAccessConfig) ListType() corev1alpha1.ObjectList {
	return &CloudAccessConfigList{}
}

// GetCommonStatus implements CommonStatusAware for CloudAccessConfig
func (c *CloudAccessConfig) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// GetNamespacedName returns a namespaced name for this CloudAccessConfig
func (c *CloudAccessConfig) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: c.Namespace,
		Name:      c.Name,
	}
}

// GetStatus returns the current overall status
func (c CloudAccessConfig) GetStatus() corev1alpha1.Status {
	return c.Status.Status
}

// GetRoles provides a consistent interface for getting the cloud roles on both cloud access configuration and
// cloud orgs
func (c *CloudAccessConfig) GetRoles() []CloudAccessConfigRole {
	return c.Spec.Roles
}

// SetAssumeProviderRole provides a consistent interface for setting a provider role on both
// cloud access configurations and cloud orgs
func (c *CloudAccessConfig) SetAssumeProviderRole(role string, assumeProviderRole string) bool {
	for i, r := range c.Spec.Roles {
		if r.Role == role {
			if r.AssumeProviderRole != assumeProviderRole {
				c.Spec.Roles[i].AssumeProviderRole = assumeProviderRole
				return true
			}
			return false
		}
	}
	// Add the role to the account structure
	c.Spec.Roles = append(c.Spec.Roles, CloudAccessConfigRole{
		Role:               role,
		AssumeProviderRole: assumeProviderRole,
	})
	return true
}

// SetAssumeProviderRole provides a consistent interface for setting a provider role on both
// cloud access configurations and cloud orgs
func (c *CloudAccessConfig) SetCloudResourceName(role string, name string) bool {
	for i, r := range c.Spec.Roles {
		if r.Role == role {
			if r.CloudResourceName != name {
				c.Spec.Roles[i].CloudResourceName = name
				return true
			}
			return false
		}
	}
	// Add the role to the account structure
	c.Spec.Roles = append(c.Spec.Roles, CloudAccessConfigRole{
		Role:              role,
		CloudResourceName: name,
	})
	return true
}

// SetDeployedRoleHash provides a consistent interface for setting a hash of a provider role for both
// cloud access and cloud orgs
func (c *CloudAccessConfig) SetDeployedRoleHash(role string, hash string) bool {
	for i, r := range c.Spec.Roles {
		if r.Role == role {
			if r.DeployedResourceHash != hash {
				c.Spec.Roles[i].DeployedResourceHash = hash
				return true
			}
			return false
		}
	}
	// Add the role to the account structure
	c.Spec.Roles = append(c.Spec.Roles, CloudAccessConfigRole{
		Role:                 role,
		DeployedResourceHash: hash,
	})
	return true
}

// SetIdentity provides a consistent interface for setting the identity on both cloud access config and
// cloud orgs
func (c *CloudAccessConfig) SetIdentity(identity CloudIdentityReference) bool {
	if c.Spec.IdentityCred.Name != identity.Name ||
		c.Spec.IdentityCred.Namespace != identity.Namespace {
		c.Spec.IdentityCred.Name = identity.Name
		c.Spec.IdentityCred.Namespace = identity.Namespace
		return true
	}
	return false
}

// Stage returns the stage of the cloud access config
func (c *CloudAccessConfig) Stage() string {
	return c.Spec.Stage
}

// GetIdentity provides a consistent interface for getting the identity on both cloud access and
// cloud orgs
func (c *CloudAccessConfig) GetIdentity() CloudIdentityReference {
	return c.Spec.IdentityCred
}

// Identifier returns the account/project/subscription ID for this cloud access config.
func (c *CloudAccessConfig) Identifier() string {
	return c.Spec.Identifier
}

// OrgIdentifier provides a consistent interface to get the org identifier on both cloud access
// and cloud orgs
func (c *CloudAccessConfig) OrgIdentifier() string {
	return c.Spec.OrgIdentifier
}

// Cloud provides a consistent interface to get the relevant cloud from both cloud access and
// cloud orgs
func (c *CloudAccessConfig) Cloud() string {
	return c.Spec.Cloud
}

// DefaultRegion provides a consistent interface to get the region from both cloud access and
// cloud orgs
func (c *CloudAccessConfig) DefaultRegion() string {
	return c.Spec.DefaultRegion
}

// CloudAccessConfigSpec defines the specification of an account known to wayfinder
// +k8s:openapi-gen=true
type CloudAccessConfigSpec struct {
	// Name is the unique logical name for this cloud access config
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Description is an optional longer human-readable description of this cloud access config to help
	// users understand which cloud access configuration to choose.
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
	// Cloud defines which cloud provider this account is for
	// +kubebuilder:validation:Enum=aws;gcp;azure
	// +kubebuilder:validation:Required
	Cloud string `json:"cloud"`
	// Identifier is the unique identifier for this account with the cloud provider, i.e. AWS
	// account ID, GCP project ID, Azure subscription, etc.
	// +kubebuilder:validation:Required
	Identifier string `json:"identifier"`
	// OrgIdentifier is an identifier for the cloud organization, i.e. AWS master account ID, Azure
	// tenant, GCP organization ID, etc. May be required for certain functionality on some clouds.
	// +kubebuilder:validation:Optional
	OrgIdentifier string `json:"orgIdentifier,omitempty"`
	// DefaultRegion is an optional default region to use for API access in this account when
	// no region is specified for the operation. This is used to determine, for example, which
	// region to use to talk to global services such as Route53 in AWS.
	// E.g. eu-west-2, europe-west2, uksouth
	// +kubebuilder:validation:Optional
	DefaultRegion string `json:"defaultRegion,omitempty"`
	// Stage defines the stage this cloud access config will be used for in the workspace. Optional for
	// 'admin' cloud access configs, required for workspace cloud access configs.
	// +kubebuilder:validation:Optional
	Stage string `json:"stage,omitempty"`
	// IdentityCred is a reference to the credential for Wayfinder to identify itself to this cloud
	// provider when using this configuration.
	//
	// Will be populated by Wayfinder with the default identity cred for this cloud if unspecified
	// on entry.
	// +kubebuilder:validation:Optional
	IdentityCred CloudIdentityReference `json:"identityCred,omitempty"`
	// Features lists the ways in which it is intended for this cloud access config to be used. This will
	// allow the relevant set of roles to be determined.
	// +kubebuilder:validation:Optional
	Features []string `json:"features,omitempty"`
	// Roles defines the possible ways in which Wayfinder can use this cloud, along with details of
	// how Wayfinder should identify itself (or provider-specific roles that need to be assumed) to
	// use this account in the specified way. The set of roles required for a cloud is
	// defined by the enabled features.
	// +kubebuilder:validation:Optional
	Roles []CloudAccessConfigRole `json:"roles,omitempty"`
}

// +k8s:openapi-gen=true
type CloudAccessConfigReference struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// GetNamespacedName converts the CloudAccessConfigReference to a standard NamespacedName
func (r CloudAccessConfigReference) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName(r)
}

func (r CloudAccessConfigReference) GetOwnership() corev1alpha1.Ownership {
	return corev1alpha1.Ownership{
		Group:     GroupVersion.Group,
		Version:   GroupVersion.Version,
		Kind:      CloudAccessConfigKind,
		Namespace: r.Namespace,
		Name:      r.Name,
	}
}

func GetCloudAccessConfigReference(o corev1alpha1.Ownership) CloudAccessConfigReference {
	return CloudAccessConfigReference{
		Namespace: o.Namespace,
		Name:      o.Name,
	}
}

// GetReference returns a reference to this cloud access config.
func (c CloudAccessConfig) GetReference() CloudAccessConfigReference {
	return CloudAccessConfigReference{
		Namespace: c.Namespace,
		Name:      c.Name,
	}
}

// GetRole returns a CloudAccessConfigRole from this cloud access config, or nil if the role is not present
func (c CloudAccessConfig) GetRole(role string) *CloudAccessConfigRole {
	for i, r := range c.Spec.Roles {
		if r.Role == role {
			return &c.Spec.Roles[i]
		}
	}
	return nil
}

type CloudAccessConfigRole struct {
	// Role is the Wayfinder cloud role that this access configuration can be used for
	// +kubebuilder:validation:Required
	Role string `json:"role"`
	// CloudResourceName is the name of the object in the cloud provider that represents this role.
	// This should be left empty, Wayfinder will populate this on entry with an appropriate value.
	// +kubebuilder:validation:Optional
	CloudResourceName string `json:"cloudResourceName"`
	// AssumeProviderRole contains a reference to the identifier that should be
	// assumed by Wayfinder when using this account for this role, i.e. AWS ARN, GCP
	// Service Account, Azure Role, etc.
	// +kubebuilder:validation:Optional
	AssumeProviderRole string `json:"assumeProviderRole,omitempty"`
	// DeployedResourceHash is a checksum calculated from the role definition when created
	// - This is set by the cli when creating the cloud resources for the role
	// - It is used by the controller to indicate if the role setup needs to be repeated
	// +kubebuilder:validation:Optional
	DeployedResourceHash string `json:"deployedResourceHash,omitempty"`
}

// CloudAccessConfigStatus defines the status of a cloud access configuration
// +k8s:openapi-gen=true
type CloudAccessConfigStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// ProviderStatus can be populated with provider-specific status information, particularly
	// relevant on accounts of type managed.
	ProviderStatus *ProviderStatus `json:"providerStatus,omitempty"`
	// Features describes the status of any features specified on this cloud access config.
	Features map[string]CloudFeatureStatus `json:"features,omitempty"`
	// Roles provides the status of each underlying required role. The keys of the
	// map are the role names.
	Roles map[string]CloudAccessConfigRoleStatus `json:"roles,omitempty"`
}

// FeatureReady will return true if this cloud access is configured and ready for use for the specified
// feature
func (s CloudAccessConfigStatus) FeatureReady(feature string) bool {
	f, ok := s.Features[feature]
	if !ok {
		return false
	}
	return f.Ready
}

// RoleReady will return true if this cloud access is configured and ready for use for the specified
// role
func (s CloudAccessConfigStatus) RoleReady(role string) bool {
	r, ok := s.Roles[role]
	if !ok {
		return false
	}
	return r.Status == RoleValid
}

// ProviderAccountType represents the concrete type of account that a CloudAccessConfiguration represents
// +kubebuilder:validation:Enum=GCPProject;AWSAccount;AzureSubscription
type ProviderAccountType string

const (
	// ProviderAccountTypeGCPProject is a GCP project for running workloads
	ProviderAccountTypeGCPProject ProviderAccountType = "GCPProject"
	// ProviderAccountTypeAWSAccount is an AWS account for running workloads
	ProviderAccountTypeAWSAccount ProviderAccountType = "AWSAccount"
	// ProviderAccountTypeAzureSubscription is an Azure subscription for running workloads
	ProviderAccountTypeAzureSubscription ProviderAccountType = "AzureSubscription"
)

// ProviderStatus provides status fields specific to a cloud provider
type ProviderStatus struct {
	// +unionDiscriminator
	// +kubebuilder:validation:Required
	// +required
	Type ProviderAccountType `json:"type"`

	// AWSAccount holds status specific to AWS accounts.
	// +optional
	AWSAccount *AWSAccountStatus `json:"awsAccount,omitempty"`
}

// AWSAccountStatus provides status specific to AWS accounts
type AWSAccountStatus struct {
	// ServiceCatalogProvisioningID is the Control Tower Account Factory Service Catalog
	// provisioning record ID. If set, creation is being tracked. Relevant only to managed AWS
	// accounts
	ServiceCatalogProvisioningID string `json:"serviceCatalogProvisioningID,omitempty"`
}

// CloudFeatureStatus describes the status of a cloud access feature
type CloudFeatureStatus struct {
	// Ready indicates whether this feature is ready to use.
	Ready bool `json:"ready,omitempty"`
	// SetupRequired indicates this feature needs wf setup cloudaccessconfig to sort it out (i.e. one or
	// more roles is missing, requires a provider role specifying, or requires updating). This will
	// be false if a role is correct and specified but somehow not valid.
	SetupRequired bool `json:"setupRequired,omitempty"`
	// RequiredRoles indicates the list of roles that this cloud access config needs working in order for
	// this feature to work. Each role identified here will have an entry in status.Roles to
	// understand the status of these underlying roles.
	RequiredRoles []string `json:"requiredRoles,omitempty"`
}

// CloudAccessConfigRoleStatus is the status of a role on a cloud access config
type CloudAccessConfigRoleStatus struct {
	Status  RoleStatus `json:"status"`
	Message string     `json:"message,omitempty"`
}

// RoleStatus is a possible status of a role on a cloud access configuration
type RoleStatus string

// RolePending indicates that the role has not yet been checked
const RolePending RoleStatus = "Pending"

// RoleValid indicates this cloud role is ready to use
const RoleValid RoleStatus = "Valid"

// RoleMissing indicates that a required role for a specified feature is not set for this cloud configuration
const RoleMissing RoleStatus = "Missing"

// RoleInvalid indicates that a specified role is not usable, for example it cannot be accessed from
// the identity associated with this cloud access config or does not exist
const RoleInvalid RoleStatus = "Invalid"

// RoleRequiresProviderRole indicates that a specified role requires an AssumeProviderRole but none
// has been provided
const RoleRequiresProviderRole RoleStatus = "RequiresProviderRole"

// RoleRequiresUpdate indicates that the permissions required for the role are not correct in the
// cloud provider so this role needs to be updated
const RoleRequiresUpdate RoleStatus = "RequiresUpdate"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudAccessConfigList provides a list of cloud access configurations
type CloudAccessConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudAccessConfig `json:"items"`
}

func (CloudAccessConfigList) ObjectType() corev1alpha1.Object {
	return &CloudAccessConfig{}
}

func (l *CloudAccessConfigList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *CloudAccessConfigList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*CloudAccessConfigList))
}

func (l *CloudAccessConfigList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *CloudAccessConfigList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []CloudAccessConfig{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*CloudAccessConfig)))
	}
}
