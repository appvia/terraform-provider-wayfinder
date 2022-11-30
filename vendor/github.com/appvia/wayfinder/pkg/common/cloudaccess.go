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
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/types"

	corev1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	cloudaccessv2 "github.com/appvia/wayfinder/server/cloudaccess/apis/cloudaccess/v2beta1"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . CloudAccessSessionProvider
//go:generate go run golang.org/x/tools/cmd/goimports -local github.com/appvia/wayfinder -e -w .
//go:generate gofmt -s -l -w commonfakes/fake_cloud_access_session_provider.go

const (
	// CloudProviderAWS represents Amazon Web Services
	CloudProviderAWS = "aws"
	// CloudProviderGCP represetns Google Cloud Platform
	CloudProviderGCP = "gcp"
	// CloudProviderAzure represents Microsoft Azure
	CloudProviderAzure = "azure"
	// CloudProviderUnknown represents unknown clouds
	CloudProviderUnknown = "unknown"
)

// CloudProviders is the set of supported cloud providers in Wayfinder
var CloudProviders = []string{
	CloudProviderAWS,
	CloudProviderGCP,
	CloudProviderAzure,
}

// WorkspaceCloudAccountName provides a standard name for the cloud access config object in a workspace
// for the given cloud and stage
func WorkspaceCloudAccountName(cloud, stage string) string {
	// This naming convention must match generateCloudAccountResource in ui/lib/api/resources/cloudaccount.ts
	return fmt.Sprintf("%s-%s", cloud, stage)
}

// IdentityCredName returns the name to use for a CloudIdentity in the admin workspace for
// Wayfinder's default identity credential for the specified cloud
func IdentityCredName(cloud string) cloudaccessv2.CloudIdentityReference {
	return cloudaccessv2.CloudIdentityReference{
		Name:      fmt.Sprintf("cloudidentity-%s", cloud),
		Namespace: corev1.AdminWorkspace.Namespace(),
	}
}

// CloudAccessTrustProvider provides trust information to wayfinder setup roles via the API
// TODO: should be moved onto the wayfinder cluster when that exists
var CloudAccessTrustProvider string

// CloudAccessRole represents a role wayfinder will use to provide functionality against a cloud provider
type CloudAccessRole string

type CloudAccessRolesType struct {
	None                      CloudAccessRole
	NetworkManager            CloudAccessRole
	ClusterManager            CloudAccessRole
	DNSZoneManager            CloudAccessRole
	DNSZoneEditor             CloudAccessRole
	AccountFactory            CloudAccessRole
	AccountManager            CloudAccessRole
	AccountManagerForProvider CloudAccessRole
	RegistryManager           CloudAccessRole
	CostsManager              CloudAccessRole
	CloudInfo                 CloudAccessRole
}

// CloudAccessRoles are the set of roles that Wayfinder may need to assume when using a
// cloud access config - these relate to our architecture
var CloudAccessRoles = CloudAccessRolesType{
	None:                      "None",
	NetworkManager:            "NetworkManager",
	ClusterManager:            "ClusterManager",
	DNSZoneManager:            "DNSZoneManager",
	DNSZoneEditor:             "DNSZoneEditor",
	AccountFactory:            "AccountFactory",
	AccountManager:            "AccountManager",
	AccountManagerForProvider: "AccountManagerForProvider",
	RegistryManager:           "RegistryManager",
	CostsManager:              "CostsManager",
	CloudInfo:                 "CloudInfo",
}

func NewErrCloudAccessRoleNotAvailable(role CloudAccessRole, caNamespace, caName string) error {
	return errCloudAccessRoleNotAvailble{err: fmt.Errorf("cloud access role %s is not available on the cloud access config %s/%s", role, caNamespace, caName)}
}

type errCloudAccessRoleNotAvailble struct {
	err error
}

func (r errCloudAccessRoleNotAvailble) Error() string {
	return r.err.Error()
}

func IsErrCloudAccessRoleNotAvailable(err error) bool {
	if _, ok := err.(errCloudAccessRoleNotAvailble); ok {
		return true
	}
	return errors.As(err, &errCloudAccessRoleNotAvailble{})
}

// GetServiceAccountName derives a service account name from any role
func (c CloudAccessRole) GetServiceAccountName(cloudAccountName string) string {
	return fmt.Sprintf("wayfinder-%s-%s", strings.ToLower(string(c)), cloudAccountName)
}

// GetIAMName will obtain the default name to use in the cloud provider for this role
func (c CloudAccessRole) GetIAMName() string {
	return fmt.Sprintf("%s-%s", CloudResourceNamePrefix, string(c))
}

// CloudAccessFeature is a representation of a "feature" that requires cloud access permissions
// - given a feature, a provider can workout what subset of roles are required for a cloudaccount
type CloudAccessFeature string

// Meta will obtain the metadata for a feature
func (c CloudAccessFeature) Meta() CloudAccessFeatureMetaData {
	for _, f := range CloudAccessFeaturesMetaData {
		if c == f.Feature {
			return f
		}
	}
	return CloudAccessFeatureMetaData{}
}

// ApplicableToAccountType indicates if this feature is applicable to the given account type
func (c CloudAccessFeature) ApplicableToAccountType(accountType CloudAccountType) bool {
	for _, a := range c.Meta().AccountTypes {
		if a == accountType {
			return true
		}
	}

	return false
}

// GetCloudAccessFeature will obtain a known feature from a string
// will return false if not known
func GetCloudAccessFeature(feature string) (CloudAccessFeature, error) {
	for _, f := range CloudAccessFeaturesMetaData {
		if feature == string(f.Feature) {
			return f.Feature, nil
		}
	}
	return CloudAccessFeature(""), fmt.Errorf("unknown feature - %s", feature)
}

// CloudAccessFeatureType is the set of features supported
// - some "features" have overlapping actual roles, see CloudAccessFeatureMeta for feature descriptions.
// - features are distinct from roles as they relate to sets of product features not the Wayfinder architecture
type CloudAccessFeatureType struct {
	AccountAutomation           CloudAccessFeature
	AccountAdministration       CloudAccessFeature
	CostsAudit                  CloudAccessFeature
	CostsEstimates              CloudAccessFeature
	Provisioning                CloudAccessFeature
	DNSZoneManagement           CloudAccessFeature
	NetworkPeering              CloudAccessFeature
	ContainerRegistryManagement CloudAccessFeature
}

// CloudAccessFeatures are the features known to Wayfinder for permissions management
// It is the responsibility of a cloud access config provider to understand which underlying roles are required
//
// See CloudAccountSpec Features property
var CloudAccessFeatures = CloudAccessFeatureType{
	AccountAutomation:           "AccountAutomation",
	AccountAdministration:       "AccountAdministration",
	CostsAudit:                  "CostsAudit",
	CostsEstimates:              "CostsEstimates",
	Provisioning:                "Provisioning",
	DNSZoneManagement:           "DNSZoneManagement",
	NetworkPeering:              "NetworkPeering",
	ContainerRegistryManagement: "ContainerRegistryManagement",
}

// CloudAccessFeatureMetaData is a Wayfinder feature with meta data
type CloudAccessFeatureMetaData struct {
	Feature CloudAccessFeature `json:"feature"`
	// Title is a human-readable name of this feature
	Title        string             `json:"title"`
	Description  string             `json:"description"`
	AccountTypes []CloudAccountType `json:"accountType"`
}

// CloudAccessFeaturesMetaData are "features" with the human readable descriptions
var CloudAccessFeaturesMetaData = []CloudAccessFeatureMetaData{
	{
		Feature:     CloudAccessFeatures.AccountAutomation,
		Title:       "Account Automation",
		Description: "Create and manage cloud accounts for your workspaces",
		AccountTypes: []CloudAccountType{
			CloudAccountTypeOrganization,
		},
	},
	{
		Feature:     CloudAccessFeatures.AccountAdministration,
		Title:       "Account Administration",
		Description: "Allows Wayfinder to manage IAM and service enablement in this account, so it can provision other roles as needed",
		AccountTypes: []CloudAccountType{
			CloudAccountTypeLinked,
		},
	},
	{
		Feature:     CloudAccessFeatures.CostsAudit,
		Title:       "Cost Audit",
		Description: "Prepare and import actual cost data for your infrastructure into Wayfinder from this account",
		AccountTypes: []CloudAccountType{
			CloudAccountTypeOrganization,
			CloudAccountTypeLinked,
		},
	},
	{
		Feature:     CloudAccessFeatures.CostsEstimates,
		Title:       "Cost Estimates",
		Description: "Access to cloud pricing information for estimating future costs",
		AccountTypes: []CloudAccountType{
			CloudAccountTypeLinked,
		},
	},
	{
		Feature:     CloudAccessFeatures.DNSZoneManagement,
		Title:       "DNS Zone Management",
		Description: "Create and manage shared DNS zones to provide DNS automatically for workspaces",
		AccountTypes: []CloudAccountType{
			CloudAccountTypeLinked,
		},
	},
	{
		Feature:     CloudAccessFeatures.NetworkPeering,
		Title:       "Network Peering",
		Description: "Automate establishment of network connectivity between cloud accounts",
		AccountTypes: []CloudAccountType{
			CloudAccountTypeLinked,
		},
	},
	{
		Feature:     CloudAccessFeatures.Provisioning,
		Title:       "Provisioning",
		Description: "Deliver Kubernetes clusters, relevant networking, DNS management and cloud container registries",
		AccountTypes: []CloudAccountType{
			CloudAccountTypeLinked,
		},
	},
	{
		Feature:     CloudAccessFeatures.ContainerRegistryManagement,
		Title:       "Container Registry Management",
		Description: "Create container registries for building and pushing a workspace's containers",
		AccountTypes: []CloudAccountType{
			CloudAccountTypeLinked,
		},
	},
}

// CloudAccountType represents the type of a cloud account
type CloudAccountType string

const (
	// CloudAccountTypeOrganization is for organizational accounts used for account factory functionality
	CloudAccountTypeOrganization CloudAccountType = "organization"
	// CloudAccountTypeLinked is for user-specified accounts to be used by Wayfinder directly
	CloudAccountTypeLinked CloudAccountType = "linked"
	// CloudAccountTypeManaged is for Wayfinder-created and managed accounts which belong to a top-level
	// organisation account
	CloudAccountTypeManaged CloudAccountType = "managed"
)

// CloudAccessSessionOpts are the options for when requesting a credentials session
type CloudAccessSessionOpts struct {
	// Region the region the session should default to
	Region string
	// Workspace is the workspace requesting the session
	Workspace corev1.WorkspaceKey
	// ImplicitRetries lets the retry logic used for the session be controlled - nil means the
	// provider default (which we have true on AWS for historical, probably incorrect, reasons).
	ImplicitRetries *bool
}

type CloudAccessSessionOptionFunc func(*CloudAccessSessionOpts)

func (f CloudAccessSessionOptionFunc) ApplyToOptions(opts *CloudAccessSessionOpts) {
	f(opts)
}

func ResolveCloudAccessSessionOptions(list []CloudAccessSessionOptionFunc) *CloudAccessSessionOpts {
	opts := &CloudAccessSessionOpts{}
	for _, o := range list {
		o.ApplyToOptions(opts)
	}
	return opts
}

type CloudAccessSessionOptionsType struct {
	WithRegion          func(string) CloudAccessSessionOptionFunc
	WithAllocationCheck func(corev1.WorkspaceKey) CloudAccessSessionOptionFunc
	WithImplicitRetries func(bool) CloudAccessSessionOptionFunc
}

var CloudAccessOption = CloudAccessSessionOptionsType{
	WithRegion: func(region string) CloudAccessSessionOptionFunc {
		return func(c *CloudAccessSessionOpts) {
			c.Region = region
		}
	},
	WithAllocationCheck: func(ws corev1.WorkspaceKey) CloudAccessSessionOptionFunc {
		return func(c *CloudAccessSessionOpts) {
			c.Workspace = ws
		}
	},
	WithImplicitRetries: func(enabled bool) CloudAccessSessionOptionFunc {
		return func(c *CloudAccessSessionOpts) {
			e := enabled
			c.ImplicitRetries = &e
		}
	},
}

type CloudAccessSessionProvider interface {
	// GetProviderSession returns an initialised session ready for use with the cloud provider and
	// account specified by accountName, for the role supplied.
	//
	// Example usage:
	// sess, err := ca.GetProviderSession(ctx, types.NamespacedName{Namespace:"ws-admin",Name:"aws-account"}, wayfinder.CloudAccountRoles.ClusterManager)
	// if err != nil {
	//   return err // handle error
	// }
	// awsSession, ok := sess.(*session.Session)
	// if !ok {
	//   return errors.New("unexpected session type")
	// }
	//
	// See the provider implementation in server/cloudaccessproviders/(cloud) to understand what type
	// a session will have for a given cloud.
	GetProviderSession(ctx context.Context, accountName types.NamespacedName, role CloudAccessRole, opts ...CloudAccessSessionOptionFunc) (interface{}, error)

	// GetProviderSessionForCredential returns an initialised session ready for use with the cloud
	// provider for the specified credential.
	//
	// WARNING: Use GetProviderSession instead of this apart from in very specific circumstances
	// where a specific credential must be used outside the context of an account, e.g. when
	// pulling data from an API to help with setting up an account.
	GetProviderSessionForCredential(ctx context.Context, credName types.NamespacedName, opts ...CloudAccessSessionOptionFunc) (interface{}, error)
}

// CloudCredentialField describes a field which may (or must) be used for a credential
type CloudCredentialField struct {
	// Name this field must have in the persisted secret
	Name string `json:"name"`
	// Title is the human-readable title to give this field when asking a user to specify it
	Title string `json:"title"`
	// Required indicates whether this field must be present for the credential to be valid
	Required bool `json:"required"`
	// Sensitive informs whether the UI/CLI should 'mask' this value (e.g. as a password field) when
	// asking the user to enter it
	Sensitive bool `json:"sensitive"`
	// Multiline informs whether this value is a long-form value (such as a multi-line JSON blob)
	// to help UI/CLI know how to ask the user for this value (e.g. textarea / file vs. input)
	Multiline bool `json:"multiline"`
	// Description provides an additional descriptive sentence to help the user determine the right
	// value for this field. Optional.
	Description string `json:"description,omitempty"`
}

// CloudAccessRoleInfo describes what a Wayfinder cloud access role means for a cloud provider with a
// description and a list of the required IAM permissions needed for this role to work on this
// cloud provider.
type CloudAccessRoleInfo struct {
	// Name is a human-readable name for this role, used to identify this role to users
	Name string `json:"name"`
	// Role is the Wayfinder cloud access role
	Role CloudAccessRole `json:"role"`
	// Description is a longer description of the meaning this role has on this cloud provider
	Description string `json:"description,omitempty"`
	// Permissions is the set of IAM permissions required on this cloud provider for this role
	Permissions map[string]string `json:"permissions"`
	// ProviderRoleRequired indicates whether it is REQUIRED to specify a provider role (e.g.
	// Role ARN, service account email address) for this role on this provider. A provider role
	// can always be provided even if this is false.
	ProviderRoleRequired bool `json:"providerRoleRequired"`
	// NonWayfinderRole specifies that this role is required to be deployed into the cloud provider but
	// is NOT used directly by Wayfinder (e.g. a role AWS needs to exist for AWS tools to work)
	NonWayfinderRole bool `json:"providerRoleName,omitempty"`
	// PermissionsTemplate indicate the permissions should be used as a template directly
	// e.g. for as use the template directly as specified
	UseAsPermissionsTemplate bool `json:"permissionsTemplate,omitempty"`
	// Features is the specific set of features that this set of permissions is for
	Features []CloudAccessFeature `json:"feature"`
}

// RequiredForFeature returns true if this role is required for the given cloud access feature
func (r CloudAccessRoleInfo) RequiredForFeature(feature CloudAccessFeature) bool {
	for _, f := range r.Features {
		if f == feature {
			return true
		}
	}

	return false
}

// PermissionsHash obtains the checksum of the permissions for this role
func (r CloudAccessRoleInfo) PermissionsHash() string {
	// sort the permissions so we get a consistent checksum
	permissions := make([]string, 0, len(r.Permissions))
	for k := range r.Permissions {
		permissions = append(permissions, k)
	}
	sort.Strings(permissions)
	// now append the permissions to a single string
	var b strings.Builder
	for _, k := range permissions {
		b.WriteString(k)
		b.WriteString(r.Permissions[k])
	}
	// finally calculate the resulting Md5 checksum
	h := md5.New()
	h.Write([]byte(b.String()))
	return hex.EncodeToString(h.Sum(nil))
}

// CloudAccessProviderData is any provider specific information
type CloudAccessProviderData struct {
	// AWSProviderData is data specific to the AWS provider
	AWSProviderData *AWSProviderData `json:"awsProviderData,omitempty"`
}

// AWSProviderData is information the AWS provider can populate
type AWSProviderData struct {
	// WayfinderRoleOrgStackSet defines the stackset that needs to be deployed by the client when
	// setting up an organisation
	WayfinderRoleOrgStackSet string `json:"wayfinderRoleOrgStackSet"`
	// WayfinderRoleOrgStackSetParams defines the stackset parameters expected by the Wayfinder server
	WayfinderRoleOrgStackSetParams map[string]string
}
