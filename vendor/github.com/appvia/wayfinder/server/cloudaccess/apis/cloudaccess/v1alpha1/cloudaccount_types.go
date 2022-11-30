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
	"k8s.io/apimachinery/pkg/types"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

const CloudAccountKind = "CloudAccount"

func CloudAccountGroupVersionKind() schema.GroupVersionKind {
	return schema.FromAPIVersionAndKind(GroupVersion.String(), CloudAccountKind)
}

// +kubebuilder:webhook:name=cloudaccounts.cloudaccess.appvia.io,mutating=false,path=/validate/cloudaccess.appvia.io/cloudaccounts,verbs=create;update;delete,groups="cloudaccess.appvia.io",resources=cloudaccount,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudAccount represents an account/project/subscription in a cloud provider which
// Wayfinder should know about
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=cloudaccount,scope=Namespaced,categories={wayfinder}
type CloudAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudAccountSpec   `json:"spec,omitempty"`
	Status CloudAccountStatus `json:"status,omitempty"`
}

func (CloudAccount) APIPath() string {
	return "cloudaccounts"
}
func (c *CloudAccount) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *CloudAccount) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*CloudAccount))
}
func (CloudAccount) ListType() corev1alpha1.ObjectList {
	return &CloudAccountList{}
}

// GetCommonStatus implements CommonStatusAware for CloudAccount
func (c *CloudAccount) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// GetNamespacedName returns a namespaced name for this account
func (c *CloudAccount) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: c.Namespace,
		Name:      c.Name,
	}
}

// Identifier returns the account/project/subscription ID for this cloud account. For managed
// accounts, this may return an empty string until the account factory process has completed.
func (c *CloudAccount) Identifier() string {
	if c.Spec.Identifier != "" {
		return c.Spec.Identifier
	}

	return c.Status.Identifier
}

// GetStatus returns the current overall status
func (c CloudAccount) GetStatus() corev1alpha1.Status {
	return c.Status.Status
}

// CloudAccountSpec defines the specification of an account known to wayfinder
// +k8s:openapi-gen=true
type CloudAccountSpec struct {
	// Name is the unique logical name for this cloud account
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Description is an optional longer human-readable description of this cloud account to help
	// users understand which cloud account to choose.
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
	// Cloud defines which cloud provider this account is for
	// +kubebuilder:validation:Enum=aws;gcp;azure
	// +kubebuilder:validation:Required
	Cloud string `json:"cloud"`
	// Identifier is the unique identifier for this account with the cloud provider, i.e. AWS
	// account ID, GCP project ID, Azure subscription, etc. Required unless the accountType is managed.
	//
	// For accountType organization this should be the account ID/subscription ID/project ID to
	// place shared org-wide resources such as DNS root zones, etc. For AWS this must be the same
	// AWS Master Account ID used for OrgIdentifier.
	//
	// To use the identifier for a CloudAccount, ALWAYS reference Status.Identifier, not this field,
	// as this will be unpopulated and ignored on managed accounts.
	// +kubebuilder:validation:Optional
	Identifier string `json:"identifier"`
	// OrgIdentifier, required only for accountType organization and must be populated with the
	// identifier for the organization - for example, AWS Master Account ID, Azure Tenant ID, GCP
	// Organization ID, etc.
	//
	// Ignored if accountType is not organization.
	//
	// +kubebuilder:validation:Optional
	OrgIdentifier string `json:"orgIdentifier"`
	// AccountType identifies whether this is an organization account (which can be used by
	// Wayfinder to create managed accounts), a shared account (used directly to build workspace
	// infrastructure), or a managed acccount (created by Wayfinder's account management
	// features)
	// +kubebuilder:validation:Enum=organization;shared;managed
	// +kubebuilder:validation:Required
	AccountType string `json:"accountType"`
	// DefaultRegion is an optional default region to use for API access in this account when
	// no region is specified for the operation. This is used to determine, for example, which
	// region to use to talk to global services such as Route53 in AWS.
	// E.g. eu-west-2, europe-west2, uksouth
	// +kubebuilder:validation:Optional
	DefaultRegion string `json:"defaultRegion,omitempty"`
	// ParentAccount is a reference to another CloudAccount (with account type 'organization') of
	// this cloud account, for managed (required) and shared (optional) accounts.
	// +kubebuilder:validation:Optional
	ParentAccount *CloudAccountReference `json:"parentAccount,omitempty"`
	// ProviderDetails provides additional fields which can be used for cloud-provider specific
	// data, such as a GCP billing account ID.
	// +kubebuilder:validation:Optional
	ProviderDetails *CloudAccountProviderDetails `json:"providerDetails,omitempty"`
	// Stages lists the stages that this cloudaccount may be used for
	// +kubebuilder:validation:Optional
	Stages []string `json:"stages,omitempty"`
	// NamingRules describes for Organization type accounts how to name child accounts based on the
	// plan chosen. Required for account factory functionality to operate for an Organization
	// account.
	// +kubebuilder:validation:Optional
	NamingRules *[]CloudAccountNamingRule `json:"namingRules,omitempty"`
	// IdentityCred is a reference to the credential for Wayfinder to identify itself to this cloud
	// provider when using this account.
	//
	// To use workload identity, specify an empty namespace and name - this will only work if
	// Workload Identity has been configured in the Wayfinder management cluster, this account is in the
	// same cloud provider as the management cluster, and that workload identity is given access to
	// this account)
	// +kubebuilder:validation:Required
	IdentityCred CloudCredentialReference `json:"identityCred,omitempty"`
	// Features lists the ways in which it is intended for this cloud account to be used. This will
	// allow the relevant set of roles to be determined for this cloud account.
	// +kubebuilder:validation:Optional
	Features []string `json:"features,omitempty"`
	// Roles defines the possible ways in which Wayfinder can use this account, along with details of
	// how Wayfinder should identify itself (or provider-specific roles that need to be assumed) to
	// use this account in the specified way. The set of roles required for a cloud account is
	// defined by the enabled features.
	// +kubebuilder:validation:Optional
	Roles []CloudAccountRole `json:"roles,omitempty"`
	// UserRoles are the roles which will be available for user access into this cloud account
	// +kubebuilder:validation:Optional
	UserRoles []string `json:"userRoles,omitempty"`
	// Allocation describes which workspaces can use this cloud account.
	// +kubebuilder:validation:Required
	Allocation corev1alpha1.ResourceAllocation `json:"allocation"`
	// OrgUserRoles is the set of user roles to make available in child accounts of this org
	//
	// Ignored if accountType is not organization.
	//
	// +kubebuilder:validation:Optional
	OrgUserRoles []OrgUserRole `json:"orgUserRoles,omitempty"`
}

// OrgUserRole describes a specific user role to orchestrate into child accounts of an org
type OrgUserRole struct {
	// UserRole is the top-level role to provide in child accounts
	// +kubebuilder:validation:Required
	UserRole string `json:"userRole"`
	// Enabled controls whether this role is enabled
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`
	// Allocation describes which workspaces can use this role. Will only be effective as a subset
	// of the workspaces allowed to use the cloud account in spec.Allocation.
	// +kubebuilder:validation:Required
	Allocation corev1alpha1.ResourceAllocation `json:"allocation"`
}

// EnabledForWorkspace determines if the specified workspace is allowed to use this user role on
// managed accounts derived from this org
func (o OrgUserRole) EnabledForWorkspace(ws corev1alpha1.WorkspaceKey) bool {
	return o.Enabled && o.Allocation.Allowed("", ws)
}

// AllocatedToWorkspace implements the Allocatable interface for CloudAccount, it will return true if the
// specified workspace is allowed to use this cloud account according to the allocation settings on
// the spec.
func (c *CloudAccount) AllocatedToWorkspace(ws corev1alpha1.WorkspaceKey) bool {
	return c.Spec.Allocation.Allowed(c.Namespace, ws)
}

// +k8s:openapi-gen=true
type CloudAccountReference struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// GetNamespacedName converts the CloudAccountReference to a standard NamespacedName
func (r CloudAccountReference) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName(r)
}

func (r CloudAccountReference) GetOwnership() corev1alpha1.Ownership {
	return corev1alpha1.Ownership{
		Group:     GroupVersion.Group,
		Version:   GroupVersion.Version,
		Kind:      "CloudAccount",
		Namespace: r.Namespace,
		Name:      r.Name,
	}
}

func GetCloudAccountReference(o corev1alpha1.Ownership) CloudAccountReference {
	return CloudAccountReference{
		Namespace: o.Namespace,
		Name:      o.Name,
	}
}

// GetReference returns a reference to this cloud account.
func (c CloudAccount) GetReference() CloudAccountReference {
	return CloudAccountReference{
		Namespace: c.Namespace,
		Name:      c.Name,
	}
}

// GetRole returns a CloudAccountRole from this cloud account, or nil if the role is not present
func (c CloudAccount) GetRole(role string) *CloudAccountRole {
	for i, r := range c.Spec.Roles {
		if r.Role == role {
			return &c.Spec.Roles[i]
		}
	}
	return nil
}

// HasUserRole returns true if the specified user role is present on the spec
func (c CloudAccount) HasUserRole(role string) bool {
	for _, r := range c.Spec.UserRoles {
		if r == role {
			return true
		}
	}
	return false
}

// GetOrgUserRole finds the specified user role on a cloud account, or returns nil if the role is
// not present
func (c *CloudAccount) GetOrgUserRole(userRole string) *OrgUserRole {
	for i, r := range c.Spec.OrgUserRoles {
		if r.UserRole == userRole {
			return &c.Spec.OrgUserRoles[i]
		}
	}
	return nil
}

func (c *CloudAccount) DisableOrgUserRole(userRole string) bool {
	for i, r := range c.Spec.OrgUserRoles {
		if r.UserRole == userRole {
			c.Spec.OrgUserRoles[i].Enabled = false
			return true
		}
	}
	return false
}

// CloudAccountProviderDetails provides parameters that are specific to a particular type of cloud account
// +union
type CloudAccountProviderDetails struct {
	// +unionDiscriminator
	// +kubebuilder:validation:Required
	// +required
	Type ProviderAccountType `json:"type"`

	// GCPOrganization holds parameters specific to GCP organization accounts. Present only if
	// type is GCPOrganization.
	// +optional
	GCPOrganization *GCPOrganizationParameters `json:"gcpOrganization,omitempty"`

	// GCPProject holds parameters specific to GCP projects. Present only if
	// type is GCPProject.
	// +optional
	GCPProject *GCPProjectParameters `json:"gcpProject,omitempty"`

	// AWSOrganization holds parameters specific to AWS organization accounts. Present only if
	// type is AWSOrganization.
	// +optional
	AWSOrganization *AWSOrganizationParameters `json:"awsOrganization,omitempty"`

	// AWSAccount holds parameters specific to AWS accounts. Present only if
	// type is AWSAccount.
	// +optional
	AWSAccount *AWSAccountParameters `json:"awsAccount,omitempty"`

	// AzureTenant holds parameters specific to Azure tenant accounts. Present only if
	// type is AzureTenant.
	// +optional
	AzureTenant *AzureTenantParameters `json:"azureTenant,omitempty"`

	// AzureSubscription holds parameters specific to Azure subscriptions. Present only if
	// type is AzureSubscription.
	// +optional
	AzureSubscription *AzureSubscriptionParameters `json:"azureSubscription,omitempty"`
}

// ProviderAccountType represents the concrete type of account that a CloudAccount represents
// +kubebuilder:validation:Enum=GCPOrganization;AWSOrganization;AzureTenant;GCPProject;AWSAccount;AzureSubscription
type ProviderAccountType string

const (
	// ProviderAccountTypeGCPOrg is a root organization account for GCP account management
	ProviderAccountTypeGCPOrg ProviderAccountType = "GCPOrganization"
	// ProviderAccountTypeGCPProject is a GCP project for running workloads
	ProviderAccountTypeGCPProject ProviderAccountType = "GCPProject"
	// ProviderAccountTypeAWSOrg is a root organization account for AWS account management
	ProviderAccountTypeAWSOrg ProviderAccountType = "AWSOrganization"
	// ProviderAccountTypeAWSAccount is an AWS account for running workloads
	ProviderAccountTypeAWSAccount ProviderAccountType = "AWSAccount"
	// ProviderAccountTypeAzureOrg is a root organization tenant for Azure account management
	ProviderAccountTypeAzureOrg ProviderAccountType = "AzureTenant"
	// ProviderAccountTypeAzureSubscription is an Azure subscription for running workloads
	ProviderAccountTypeAzureSubscription ProviderAccountType = "AzureSubscription"
)

// GCPOrganizationParameters provides the specific parameters for a GCP organisation account
type GCPOrganizationParameters struct {
	// ParentType is the type of parent this project has
	// Valid types are: "organization", "folder", and "project"
	// +kubebuilder:validation:Enum=organization;folder;project
	// +kubebuilder:validation:Required
	ParentType string `json:"parentType"`
	// DEPRECATED: Use OrgIdentifier on Spec. This will be ignored if OrgIdentifier is populated
	// on the Spec.
	// +kubebuilder:validation:Optional
	ParentID string `json:"parentID"`
	// BillingAccountName is the resource name of the billing account associated with the project
	// e.g. '012345-567890-ABCDEF'
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	BillingAccount string `json:"billingAccount"`
}

// GCPProjectParameters provides the specific parameters for a GCP project account
type GCPProjectParameters struct {
}

// AWSOrganizationParameters provides the specific parameters for an AWS organisation account
type AWSOrganizationParameters struct {
	// SsoUser is the user who will be the organisational account owner for all accounts.
	// Required if feature AccountManagement enabled.
	// +kubebuilder:validation:Optional
	SsoUser AWSSSOUser `json:"ssoUser"`
	// OuName is the name of the parent Organizational Unit (OU) to use for provisioning accounts
	// Required if feature AccountManagement enabled.
	// +kubebuilder:validation:Optional
	OuName string `json:"ouName"`
	// Region is the region where control tower is enabled in the master account
	// Required if feature AccountManagement enabled.
	// +kubebuilder:validation:Optional
	Region string `json:"region"`
	// UserRoles contains the ARNs of stacksets to create instances of in each managed account
	// +kubebuilder:validation:Optional
	UserRoles map[string]string `json:"userRoles,omitempty"`
}

// AWSSSOUser describes the details required to identify an AWS SSO user to user for all accounts
type AWSSSOUser struct {
	// Email is the unique user email address specified for the AWS SSO user
	// Required if feature AccountManagement enabled.
	// +kubebuilder:validation:Optional
	Email string `json:"email"`
	// FirstName is the firstname(s) field for an AWS SSO user
	// Required if feature AccountManagement enabled.
	// +kubebuilder:validation:Optional
	FirstName string `json:"firstName"`
	// LastName is the last name of an SSO user
	// Required if feature AccountManagement enabled.
	// +kubebuilder:validation:Optional
	LastName string `json:"lastName"`
}

// AWSAccountParameters provides the specific parameters for an AWS account
type AWSAccountParameters struct {
}

// AzureTenantParameters provides the specific parameters for an Azure tenant (organisation) account
type AzureTenantParameters struct {
	// AgreementType defines whether we're building subscriptions in an MCA or Enterprise
	// Agreement backed Azure setup
	// +kubebuilder:validation:Enum=EA;MCA
	// +kubebuilder:validation:Required
	AgreementType string `json:"agreementType"`

	// OwnerObjectID specifies the Object ID of an Azure AD group, user or service principal to
	// grant Owner privilege on all created subscriptions. This is required to ensure that
	// generated subscriptions are owned by an object controlled by your company.
	//
	// Example: 8bf96a8f-abcd-ef12-a389-883d6116a5da
	// +kubebuilder:validation:Required
	OwnerObjectID string `json:"ownerObjectID"`

	// ContributorObjectID specifies an optional object ID of an Azure AD group, user or service
	// principal to grant Contributor privilege on all created subscriptions.
	//
	// Example: 8bf96a8f-dcef-abc1-a389-883d6116a5da
	ContributorObjectID string `json:"contributorObjectID,omitempty"`

	// ManagementGroupID specifies an optional ID of an Azure Management Group in which
	// subscriptions created by Wayfinder should be placed.
	//
	// Example: wf-subscription-mgt-group
	ManagementGroupID string `json:"managementGroupID,omitempty"`

	// BillingAccount is the billing account identifier. Required for both agreement types.
	//
	// Example (super-catchy, isn't it): aaa111b-abcd-ef01-2345-bcdabc123fed:1234aaab-0100-1234-abcd-abcd0123abcd_2019-05-31
	// +kubebuilder:validation:Required
	BillingAccount string `json:"billingAccount"`

	// EnrollmentAccount defines for an Enterprise Agreement agreement type which enrollment
	// account to create subscriptions within. Required for EA.
	//
	// Example: 7654321
	EnrollmentAccount string `json:"enrollmentAccount,omitempty"`

	// BillingProfile defined for an MCA agreement type which billing profile contains the
	// invoice section you wish subscriptions to be created in. Required for MCA.
	//
	// Example: AW4F-APQW-0AH-ABC
	BillingProfile string `json:"billingProfile,omitempty"`
	// InvoiceSection defines for an MCA agreement type which invoice section to create
	// subscriptions within inside the selected BillingProfile. Required for MCA.
	//
	// Example: PQRS-ALDS-012-DEF
	InvoiceSection string `json:"invoiceSection,omitempty"`
}

// AzureSubscriptionParameters provides the specific parameters for an Azure subscription
type AzureSubscriptionParameters struct {
}

// CloudAccountNamingRule describes the rules for naming a child account based on the selected
// plan
type CloudAccountNamingRule struct {
	// Name is the given name of the rule
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Description provides an optional description for the account rule
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
	// Stages is a list of stages permitted
	// +kubebuilder:validation:Optional
	Stages []string `json:"stages"`
	// Suffix is the applied suffix
	// +kubebuilder:validation:Optional
	Suffix string `json:"suffix,omitempty"`
	// Prefix is a prefix for the account name
	// +kubebuilder:validation:Optional
	Prefix string `json:"prefix,omitempty"`
}

// NameAccount returns the resulting name for this naming rule for a workspace account
func (r *CloudAccountNamingRule) NameAccount(ws corev1alpha1.WorkspaceKey) string {
	accountName := string(ws)
	if r.Suffix != "" {
		accountName = fmt.Sprintf("%s-%s", accountName, r.Suffix)
	}
	if r.Prefix != "" {
		accountName = fmt.Sprintf("%s-%s", r.Prefix, accountName)
	}
	return accountName
}

type CloudAccountRole struct {
	// Role is the Wayfinder cloud role that this account can be used for
	// +kubebuilder:validation:Required
	Role string `json:"role"`
	// AssumeProviderRole contains a reference to the identifier that should be
	// assumed by Wayfinder when using this account for this role, i.e. AWS ARN, GCP
	// Service Account, Azure Role, etc.
	// +kubebuilder:validation:Optional
	AssumeProviderRole string `json:"assumeProviderRole,omitempty"`
}

// CloudAccountStatus defines the status of a cloud account
// +k8s:openapi-gen=true
type CloudAccountStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// Identifier is the assigned unique identifier for this account. For emanaged accounts this
	// will be the identifier for the created account. For all other accounts, this will be the
	// value provided in Spec.Identifier.
	//
	// This field should ALWAYS be used if you need the correct identifier for a cloud account. Do
	// not rely on Spec.Identifier which is unpopulated for managed accounts.
	Identifier string `json:"identifier,omitempty"`
	// ProviderAccountRef is a reference to the provider account for this cloud account where the type is managed.
	ProviderAccountRef *corev1alpha1.Ownership `json:"providerAccountRef,omitempty"`
	// ProviderStatus can be populated with provider-specific status information, particularly
	// relevant on accounts of type managed.
	ProviderStatus *ProviderStatus `json:"providerStatus,omitempty"`
	// Features describes the status of any features specified on this cloud account.
	Features map[string]CloudAccountFeatureStatus `json:"features,omitempty"`
	// Roles provides the status of each underlying required role. The keys of the
	// map are the role names.
	Roles map[string]CloudAccountRoleStatus `json:"roles,omitempty"`
	// UserRoles provides the status of the user roles on this cloud account
	UserRoles map[string]CloudAccountUserRoleStatus `json:"userRoles,omitempty"`
	// OrgIdentifier is the identifier for an organisation owner of this account when known
	// identifier for the organization - for example, AWS Master Account ID, Azure Tenant ID, GCP
	// Organization ID, etc.
	OrgIdentifier string `json:"orgIdentifier,omitempty"`
}

// FeatureReady will return true if this cloud account is configured and ready for use for the specified
// feature
func (s CloudAccountStatus) FeatureReady(feature string) bool {
	f, ok := s.Features[feature]
	if !ok {
		return false
	}
	return f.Ready
}

// RoleReady will return true if this cloud account is configured and ready for use for the specified
// role
func (s CloudAccountStatus) RoleReady(role string) bool {
	r, ok := s.Roles[role]
	if !ok {
		return false
	}
	return r.Status == RoleValid
}

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

// CloudAccountFeatureStatus describes the status of a cloud account feature
type CloudAccountFeatureStatus struct {
	// Ready indicates whether this feature is ready to use.
	Ready bool `json:"ready,omitempty"`
	// SetupRequired indicates this feature needs wayfinder setup roles run to sort it out (i.e. one or
	// more roles is missing, requires a provider role specifying, or requires updating). This will
	// be false if a role is correct and specified but somehow not valid.
	SetupRequired bool `json:"setupRequired,omitempty"`
	// RequiredRoles indicates the list of roles that this cloud account needs working in order for
	// this feature to work. Each role identified here will have an entry in status.Roles to
	// understand the status of these underlying roles.
	RequiredRoles []string `json:"requiredRoles,omitempty"`
}

// CloudAccountRoleStatus is the status of a role on a cloud account
type CloudAccountRoleStatus struct {
	Status  RoleStatus `json:"status"`
	Message string     `json:"message,omitempty"`
}

// CloudAccountUserRoleStatus is the status of a specific user role on the cloud account
type CloudAccountUserRoleStatus struct {
	// UserRole is the role this status describes
	UserRole string `json:"userRole"`
	// AssumeProviderRole may be populated for a specific cloud provider in order to describe
	// how Wayfinder will orchestrate user access into this role
	// +kubebuilder:validation:Optional
	AssumeProviderRole string `json:"assumeProviderRole,omitempty"`
	// Status provides an overall status.
	// +kubebuilder:validation:Optional
	Status corev1alpha1.Status `json:"status,omitempty"`
	// Message is the description of the current status.
	// +kubebuilder:validation:Optional`json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

// RoleStatus is a possible status of a role on a cloud account
type RoleStatus string

// RolePending indicates that the role has not yet been checked
const RolePending RoleStatus = "Pending"

// RoleValid indicates this cloud account role is ready to use
const RoleValid RoleStatus = "Valid"

// RoleMissing indicates that a required role for a specfied feature is not set on this cloudaccount
const RoleMissing RoleStatus = "Missing"

// RoleInvalid indicates that a specified role is not usable, for example it cannot be accessed from
// the identity associated with this cloud account or does not exist
const RoleInvalid RoleStatus = "Invalid"

// RoleRequiresProviderRole indicates that a specified role requires an AssumeProviderRole but none
// has been provided
const RoleRequiresProviderRole RoleStatus = "RequiresProviderRole"

// RoleRequiresUpdate indicates that the permissions required for the role are not correct in the
// cloud provider so this role needs to be updated
const RoleRequiresUpdate RoleStatus = "RequiresUpdate"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudAccountList provides a list of cloud accounts
type CloudAccountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudAccount `json:"items"`
}

func (CloudAccountList) ObjectType() corev1alpha1.Object {
	return &CloudAccount{}
}

func (l *CloudAccountList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *CloudAccountList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*CloudAccountList))
}

func (l *CloudAccountList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *CloudAccountList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []CloudAccount{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*CloudAccount)))
	}
}
