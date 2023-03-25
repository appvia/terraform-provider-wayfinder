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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	"github.com/appvia/wayfinder/pkg/common"
	computev2beta1 "github.com/appvia/wayfinder/server/cloudinfra/clusters/apis/compute/v2beta1"
	orgv2 "github.com/appvia/wayfinder/server/org/apis/org/v2beta1"
	"github.com/appvia/wayfinder/server/org/services/authentication/oauth"
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
	// Username is your username
	Username string `json:"username,omitempty"`
	// Workspaces is a collection of workspaces you're a member of
	Workspaces []string `json:"workspaces,omitempty"`
	// WayfinderGroups is the list of Wayfinder-scoped groups that you are a member of
	WayfinderGroups []string `json:"wayfinderGroups,omitempty"`
	// WorkspaceGroups is a map of a workspace key to the list of groups you are a member of in that
	// workspace
	WorkspaceGroups map[string][]string `json:"workspaceGroups,omitempty"`
	// SubjectKind lets you know what you ARE
	SubjectKind orgv2.SubjectKind `json:"subjectKind,omitempty"`
}

// WorkspaceInvitationResponse returns the workspace
type WorkspaceInvitationResponse struct {
	// Workspace is the name of workspace which the user just has been added to
	Workspace corev1alpha1.WorkspaceKey `json:"workspace"`
}

// WorkspacesOverview is a list of workspaces, including an overview
// including clusters and count of namespaces and members for each workspace
type WorkspacesOverview struct {
	WorkspaceList orgv2.WorkspaceList `json:"workspaces"`

	WorkspaceResources []WorkspaceResources `json:"workspaceResources,omitempty"`
}

// NamespaceSummary is the details of a namespace
type NamespaceSummary struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Stage    string `json:"stage"`
}

// AppSummar is the details of an App
type AppSummary struct {
	Name   string   `json:"name"`
	Cloud  string   `json:"cloud"`
	Stages []string `json:"stages"`
}

// ClusterSummary is the details of a cluster
type ClusterSummary struct {
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	Stage        string `json:"stage"`
	Environments int    `json:"environments"`
	Namespaces   int    `json:"namespaces"`
}

// CloudAccountSummary is the details of a cloud account
type CloudAccountSummary struct {
	Name   string   `json:"name"`
	Cloud  string   `json:"cloud"`
	Type   string   `json:"type"`
	Stages []string `json:"stages"`
}

type Stages map[string]int

type WorkspaceResourceApplications struct {
	Total int `json:"total"`
}
type WorkspaceResourceClusters struct {
	Stages  `json:"stages"`
	Total   int              `json:"total"`
	Summary []ClusterSummary `json:"summary"`
}
type WorkspaceResourceNamespaces struct {
	Stages `json:"stages"`
	Total  int `json:"total"`
}
type WorkspaceResourceEnvironments struct {
	Stages `json:"stages"`
	Total  int `json:"total"`
}

type WorkspaceResourceClouds struct {
	EnabledClouds `json:",inline"`
}

// WorkspaceResources is an overview of a workspace, including clusters and a count
// of namespace claims and members, plus other counts related to the workspace
type WorkspaceResources struct {
	WorkspaceKey corev1alpha1.WorkspaceKey `json:"workspaceKey"`
	// Clouds
	Clouds WorkspaceResourceClouds `json:"clouds"`
	// Applications
	Applications WorkspaceResourceApplications `json:"applications"`
	// Clusters
	Clusters WorkspaceResourceClusters `json:"clusters"`

	// Namespaces
	Namespaces WorkspaceResourceNamespaces `json:"namespaces"`
	// Environments
	Environments WorkspaceResourceEnvironments `json:"environments"`
	MemberCount  int                           `json:"memberCount"`
}

// AdminSetup is the present state of the setup process to the platform admin
type AdminSetup struct {
	IdpConfigured        bool               `json:"idpConfigured"`
	CloudAccounts        CloudAccountSetup  `json:"cloudAccounts"`
	StageCount           int                `json:"stageCount"`
	EKSClusterPlanCount  int                `json:"eksClusterPlanCount"`
	AKSClusterPlanCount  int                `json:"aksClusterPlanCount"`
	GKEClusterPlanCount  int                `json:"gkeClusterPlanCount"`
	ClusterPlanCount     int                `json:"clusterPlanCount"`
	WorkspaceCount       int                `json:"workspaceCount"`
	EmptyWorkspaceCount  int                `json:"emptyWorkspaceCount"`
	AppCount             int                `json:"appCount"`
	AppSummary           []AppSummary       `json:"appSummary"`
	ClusterCount         int                `json:"clusterCount"`
	ClusterSummaryList   []ClusterSummary   `json:"clusterSummaryList"`
	NamespaceSummaryList []NamespaceSummary `json:"namespaceSummaryList"`
	Networking           NetworkingSetup    `json:"networking"`
	UserCount            int                `json:"userCount"`
	WayfinderAdminCount  int                `json:"wayfinderAdminCount"`
	WorkspaceGroupCount  int                `json:"workspaceGroupCount"`
	AccessRoleCount      int                `json:"accessRoleCount"`
	AccessPolicyCount    int                `json:"accessPolicyCount"`
	WorkspaceAdminCount  int                `json:"workspaceAdminCount"`
	WorkspaceMemberCount int                `json:"workspaceMemberCount"`
	Costs                CostsSetup         `json:"costs"`
	Packages             int                `json:"packages"`
	FromCache            bool               `json:"fromCache"`
}

// CloudAccountSetup is a count of the number of orgs and managed/shared accounts set up
type CloudAccountSetup struct {
	OrganizationCount   int                   `json:"organizationCount"`
	ManagedAccountCount int                   `json:"managedAccountCount"`
	SharedAccountCount  int                   `json:"sharedAccountCount"`
	SummaryList         []CloudAccountSummary `json:"summaryList"`
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
	// AccountRoles describes which wayfinder roles are supported by this provider and their
	// associated provider-specific IAM permissions for shared accounts
	AccountRoles []common.CloudAccessRoleInfo `json:"accountRoles"`
	// OrgRoles describes which wayfinder roles are supported by this provider and their
	// associated provider-specific IAM permissions for org accounts
	OrgRoles []common.CloudAccessRoleInfo `json:"orgRoles"`
	// OrgExtraRoles contains any extra roles needed in the cloud provider but not directly
	// used by wayfinder
	OrgExtraRoles []common.CloudAccessRoleInfo `json:"orgExtraRoles"`
	// AccountFeatures provides the set of features which this cloud provider supports for linked
	// accounts
	AccountFeatures []common.CloudAccessFeatureMetaData `json:"sharedFeatures"`
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

type ResourceList struct {
	Items []runtime.Object `json:"items"`
}

type Resource runtime.Object

// GenerateNamespaceSummaryList adds namespaces to an array for displaying in summary information
func GenerateNamespaceSummaryList(nsList computev2beta1.NamespaceClaimList, clusterList computev2beta1.ClusterList) []NamespaceSummary {
	nsSummaryList := []NamespaceSummary{}
	// add namespaces to summary list
	for i := 0; i < len(nsList.Items); i++ {
		if !common.IsSystemResource(&nsList.Items[i]) {
			nsName := nsList.Items[i].Name
			kind := nsList.Items[i].Spec.Cluster.Kind
			if kind == computev2beta1.ClusterKind {
				for j := 0; j < len(clusterList.Items); j++ {
					name := clusterList.Items[j].Name
					stage := clusterList.Items[j].Spec.Stage
					provider := clusterList.Items[j].Spec.Provider
					if name == nsList.Items[i].Spec.Cluster.Name {
						nsSummaryList = append(nsSummaryList, NamespaceSummary{Name: nsName, Provider: provider, Stage: stage})
					}
				}
			}
		}
	}
	return nsSummaryList
}

// ApplicationDeploymentRequest describes a desire to generate some manifests or do a deployment for
// the specified application environment
type ApplicationDeploymentRequest struct {
	// AppEnv is the name of the application environment
	AppEnv string `json:"appEnv"`
	// Components is the names of the components that should be returns
	Components []string `json:"components"`

	// ImageTag to use for all components, unless overridden by an entry in Images or ImageTags
	ImageTag string `json:"imageTag,omitempty"`
	// Images is an optional set of image overrides for the components being deployed. If no image
	// is specified here, the default image from the component definition will be used.
	Images map[string]string `json:"images,omitempty"`
	// ImageTags is an optional set of image tag overrides for the components being deployed. If no
	// image tag is specified here (and no entry is provided in Images), the default image from the
	// component definition will be used.
	ImageTags map[string]string `json:"componentImageTags,omitempty"`

	// Force runs the patch with force to override conflicts
	Force bool `json:"force,omitempty"`
	// Remove causes the deployment to remove all resources from the manifest instead of deploying
	// them.
	Remove bool `json:"remove,omitempty"`
}

type EnabledCloud struct {
	// Enabled indicates if the cloud is enabled
	Enabled bool `json:"enabled"`
	// Stages indicates which stages are applicable (only valid at the workspace level)
	Stages []string `json:"stages,omitempty"`
}

// EnabledClouds represents whether a cloud is enabled
type EnabledClouds struct {
	AWS   EnabledCloud `json:"aws"`
	Azure EnabledCloud `json:"azure"`
	GCP   EnabledCloud `json:"gcp"`
}

// RepositoryCredential represents some kind of credential required for accessing
// a Terraform module repository.
//
// Currently it assumes a URL and a Personal Access Token (PAT) for GitHub, as that is
// the only credential form supported by the Terranetes API at the time of writing.
type RepositoryCredential struct {
	// URL is the URL the credential is associated with.
	URL string `json:"url"`

	// Token is the Personal Access Token.
	Token string `json:"token,omitempty"`
}

// ImportModule represents a request to import a specific terraform module found at the
// supplied URL.
type ImportModule struct {
	// URL is the URL of the resource to import into the Terranetes API.
	URL string `json:"url"`
}

// TLSConfig represents the TLS configuration for wayfinder to use
type TLSConfig struct {
	// Email is the email for use
	Email string `json:"email"`
}
