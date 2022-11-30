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

package types

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	"github.com/appvia/wayfinder/pkg/authentication/oauth"
	"github.com/appvia/wayfinder/pkg/common"
	computev1alpha1 "github.com/appvia/wayfinder/server/cloudinfra/clusters/apis/compute/v1alpha1"
	orgv1alpha1 "github.com/appvia/wayfinder/server/org/apis/org/v1alpha1"
)

const (
	// APIVersion is the wayfinder api server we are serving
	APIVersion = "/api/v1alpha1"
)

type LoginResponse struct {
	// OIDCResponse
	OIDCResponse *OIDCResponse
	// IssuedToken
	IssuedToken *IssuedToken
}

// OIDCResponse contains the result of a authorization request
type OIDCResponse struct {
	// AuthorizationURL is the endpoint for identity provider
	AuthorizationURL string `json:"authorization-url,omitempty"`
	// ClientID is the client id of the login
	ClientID string `json:"client-id,omitempty"`
	// ClientSecret is used for refreshing
	ClientSecret string `json:"client-secret,omitempty"`
	// AccessToken is the access token provided
	AccessToken string `json:"access-token,omitempty"`
	// RefreshToken is a potential refresh token
	RefreshToken string `json:"refresh-token,omitempty"`
	// IDToken string is the identity token
	IDToken string `json:"id-token,omitempty"`
	// TokenEndpointURL is the token endpoint
	TokenEndpointURL string `json:"token-endpoint-url,omitempty"`
}

// IssuedToken is a minted token from wayfinder
type IssuedToken struct {
	// RefreshToken is a token to retrieve new tokens, populated only on
	// initial login
	RefreshToken string `json:"refresh-token,omitempty"`
	// Token is the actual token for accessing the API
	Token string `json:"token,omitempty"`
	// Expires is the time token will expire
	Expires int64 `json:"expires,omitempty"`
}

// LocalUser represents a local user for login purposes
type LocalUser struct {
	// Username is the user's username
	Username string `json:"username,omitempty"`
	// Password used to identify the local user
	Password string `json:"password,omitempty"`
}

// WhoAmI provides a description to who you are
type WhoAmI struct {
	// AuthMethod is the authentication method being used
	AuthMethod string `json:"authMethod,omitempty"`
	// Email is the user email
	Email string `json:"email,omitempty"`
	// Username is your username
	Username string `json:"username,omitempty"`
	// Workspaces is a collection of workspaces your in
	Workspaces []string `json:"workspaces,omitempty"`
}

// WorkspaceInvitationResponse returns the workspace
type WorkspaceInvitationResponse struct {
	// Workspace is the name of workspace which the user just has been added to
	Workspace corev1alpha1.WorkspaceKey `json:"workspace"`
}

// WorkspacesOverview is a list of workspaces, including an overview
// including clusters and count of namespaces and members for each workspace
type WorkspacesOverview struct {
	WorkspaceList orgv1alpha1.WorkspaceList `json:"workspaces"`

	WorkspaceResources []WorkspaceResources `json:"workspaceResources,omitempty"`
}

// WorkspaceResources is an overview of a workspace, including clusters and a count
// of namespace claims and members, plus other counts related to the workspace
type WorkspaceResources struct {
	WorkspaceKey           corev1alpha1.WorkspaceKey             `json:"workspaceKey"`
	Workspace              orgv1alpha1.Workspace                 `json:"workspace"`
	ClusterList            computev1alpha1.ClusterList           `json:"clusters"`
	ClusterAllocationsList computev1alpha1.ClusterAllocationList `json:"clusterAllocations"`
	NamespaceClaimCount    int                                   `json:"namespaceClaimCount"`
	DNSZoneCount           int                                   `json:"dnsZoneCount"`
	HelmCount              int                                   `json:"helmCount"`
	ContainerRegistryCount int                                   `json:"containerRegistryCount"`
	MemberCount            int                                   `json:"memberCount"`
}

// AdminSetup is the present state of the setup process to the platform admin
type AdminSetup struct {
	IdpConfigured    bool              `json:"idpConfigured"`
	CloudAccounts    CloudAccountSetup `json:"cloudAccounts"`
	StageCount       int               `json:"stageCount"`
	ClusterPlanCount int               `json:"clusterPlanCount"`
	WorkspaceCount   int               `json:"workspaceCount"`
	Networking       NetworkingSetup   `json:"networking"`
	UserCount        int               `json:"userCount"`
	Costs            CostsSetup        `json:"costs"`
	FromCache        bool              `json:"fromCache"`
}

// CloudAccountSetup is a count of the number of orgs and managed/shared accounts set up
type CloudAccountSetup struct {
	OrganizationCount   int `json:"organizationCount"`
	ManagedAccountCount int `json:"managedAccountCount"`
	SharedAccountCount  int `json:"sharedAccountCount"`
}

// NetworkingSetup is measure of the set up networking options including DNSZones,
// IP address ranges, and perring rules
type NetworkingSetup struct {
	DNSZones                DNSZoneSetup `json:"dnsZones"`
	AssignableNetworksCount int          `json:"assignableNetworksCount"`
	PeeringRuleCount        int          `json:"peeringRuleCount"`
}

// DNSZoneCount is a count of the number of global and any other DNS zones
type DNSZoneSetup struct {
	GlobalZoneCount int `json:"globalZoneCount"`
	ZoneCount       int `json:"zoneCount"`
}

type CostsSetup struct {
	Audit     bool `json:"audit"`
	Estimates bool `json:"estimates"`
}

// Health provides an indication of the health of the API.
type Health struct {
	// Healthy is true if the service is healthy.
	Healthy bool `json:"healthy"`
	// Unready is the list of unready components
	Unready []string `json:"unready,omitempty"`
}

// License provides high level status of the product license
type License struct {
	// Valid is the overall license validaty status
	Valid bool   `json:"valid"`
	Error string `json:"error"`
}

// ServerInfo provides details around the settings and version of api
type ServerInfo struct {
	// APIGroups is a collection of supported apigroups
	APIGroups []schema.GroupVersion `json:"apiGroups,omitempty"`
	// Features is a collection of supported / enabled features
	Features map[string]string `json:"features,omitempty"`
	// CloudAccessFeatures are the cloud access specific features
	CloudAccessFeatures []common.CloudAccessFeatureMetaData `json:"cloudAccessFeatures,omitempty"`
	// Version is the api version details
	Version Version `json:"version"`
	// InstanceIdentifier is the unique identifier of Wayfinder
	InstanceIdentifier string `json:"instanceIdentifier"`
	// Namespace is the namespace of the installation
	Namespace string `json:"namespace"`
	// ClusterTrustProvider provides trust information to setup roles
	// - the cloud can be identified by the cloudcredential implicit ID at this time
	// - for aws this field will be the OIDC provider host and path string
	// - for Azure this is the cluster id
	// TODO: should be moved onto the wayfinder cluster when that exists
	// - for gcp this could be the trusted service account - TBD
	ClusterTrustProvider string `json:"clusterTrustProvider,omitempty"`
}

// IsFeatureEnabled checks if the feature is enabled
func (s *ServerInfo) IsFeatureEnabled(feature string) bool {
	return s.Features[feature] == "true"
}

// Version defines the version of the api
type Version struct {
	// Release is the release tag
	Release string `json:"release"`
	// SHA is the git sha
	SHA string `json:"sha"`
}

// CloudAccessProviderMeta describes additional information about a cloud access provider
type CloudAccessProviderMeta struct {
	// Name of the provider.
	Name string `json:"name"`
	// CredentialFields describes the fields that make up a secret for credentials for this provider
	CredentialFields []common.CloudCredentialField `json:"credentialFields"`
	// SharedAccountRoles describes which wayfinder roles are supported by this provider and their
	// associated provider-specific IAM permissions for shared accounts
	SharedAccountRoles []common.CloudAccessRoleInfo `json:"sharedAccountRoles"`
	// OrgAccountRoles describes which wayfinder roles are supported by this provider and their
	// associated provider-specific IAM permissions for org accounts
	OrgAccountRoles []common.CloudAccessRoleInfo `json:"orgAccountRoles"`
	// OrgAccountExtraRoles contains any extra roles needed in the cloud provider but not directly
	// used by wayfinder
	OrgAccountExtraRoles []common.CloudAccessRoleInfo `json:"orgAccountExtraRoles"`
	// SharedFeatures provides the set of features which this cloud provider supports for shared
	// accounts
	SharedFeatures []common.CloudAccessFeatureMetaData `json:"sharedFeatures"`
	// OrgFeatures provides the set of features which this cloud provider supports for org accounts
	OrgFeatures []common.CloudAccessFeatureMetaData `json:"orgFeatures"`
	// ProviderData is the data in the data format understood by the provider
	ProviderData common.CloudAccessProviderData `json:"providerInfo"`
}

type IDPConfigurationVerification struct {
	// Success will be true if the IDP configuration provided is valid, false otherwise
	Success bool `json:"success"`
	// TestURL is a URL to open to test the oauth flow works
	TestURL string `json:"testURL,omitempty"`
	// Messages provides a list of one or more messages describing why the provided details did not
	// verify. Empty on successful.
	Messages []string `json:"messages,omitempty"`
}

type IDPProvidersInfo struct {
	// Providers is the list of tested providers which can be integrated with Wayfinder
	Providers oauth.OAuthProviders `json:"providers"`
}
