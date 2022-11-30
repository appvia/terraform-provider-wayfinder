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

package authentication

import (
	corev1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	orgv2 "github.com/appvia/wayfinder/server/org/apis/org/v2beta1"
)

// MemberRole is the name of the default member role
const MemberRole = "member"
const (
	// AuthenicationModeToken indicates a admin token
	AuthenicationModeToken = "token"
	// AuthenicationModeKubernetes indicates a internal service account
	AuthenicationModeKubernetes = "kubernetes"
	// AuthenicationModeSSO indicates an sso account
	AuthenicationModeSSO = "sso"
	// AuthenicationModeJWT indicates an jwt account
	AuthenicationModeJWT = "jwt"
	// SystemAuthenticated indicates the request has an authenticated user
	SystemAuthenticated = "system:authenticated"
	// WorkspaceAuthenticated indicates a member of request resource
	WorkspaceAuthenticated = "workspace:authenticated"
)

const (
	// Audience is the audience of the tokens
	Audience = "wayfinder"
	// KubernetesAudience is the audience of the tokens
	KubernetesAudience = "kubernetes"
	// RefreshTokenAudience is the audience for wayfinder refresh tokens
	RefreshTokenAudience = "refresh"
)

const (
	// ScopeExchange indicates a token exchange scope for an access token
	ScopeExchange = "wayfinder:auth:exchange"
	// ScopeRefresh indicates a refresh token
	ScopeRefresh = "wayfinder:auth:refresh"
	// ScopeAccessToken is a scope used for tokens for access tokens (both workspace and
	// wayfinder-scoped)
	ScopeAccessToken = "wayfinder:system:accesstoken"
	// ScopeKubernetesAccount indicates a kubernetes account
	ScopeKubernetesAccount = "wayfinder:system:kubernetes"
	// ScopeUser is a user account type
	ScopeUser = "wayfinder:system:user"
)

// ContextKey is the context key
type ContextKey struct{}

// ContextAdmissionKey is the context key for a admissions request
type ContextAdmissionKey struct{}

// Identity provides the user
type Identity interface {
	// Address returns the ip address of the user
	Address() string
	// AuthMethod is the method the identity logged in with
	AuthMethod() string
	// IsAccessToken will be true if this is a workspace or global scoped access token identity
	IsAccessToken() bool
	// IsUser checks if this identity represents a user
	IsUser() bool
	// IsGlobalAdmin checks if the identity is a global admin
	IsGlobalAdmin() bool
	// IsMember checks if the identity is a member of a workspace
	IsMember(corev1.WorkspaceKey) bool
	// Disabled checks if the identity is disabled
	Disabled() bool
	// SubjectKind describes what kind of user this entity represents
	SubjectKind() orgv2.SubjectKind
	// Username is a unique username for this identity
	Username() string
	// Scopes is a list of scope for the identity
	Scopes() []string
	// Workspaces is a list of workspaces for the user
	Workspaces() corev1.WorkspaceKeys
	// WorkspaceGroups is a collection of groups the identity is a member of in the specified workspace
	WorkspaceGroups(corev1.WorkspaceKey) []string
	// AllWorkspaceGroups is a collection of groups the identity is a member of in all workspaces
	AllWorkspaceGroups() map[corev1.WorkspaceKey][]string
	// WayfinderGroups is a collection of the groups defined by Wayfinder that identity is a member of
	WayfinderGroups() []string
	// InGlobalGroup is a convenience wrapper to detect if the user is in a specific global group
	InWayfinderGroup(string) bool
	// InWorkspaceGroup is a convenience wrapper to detect if the user is in a specific workspace group
	InWorkspaceGroup(corev1.WorkspaceKey, string) bool
	// UID returns the associated UID for this identity - only typically used with access tokens
	UID() string
	// ToSubject returns a representation of this identity as a subject
	ToSubject() orgv2.Subject
}
