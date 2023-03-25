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
	"context"
	"fmt"
	"strings"

	"github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	"github.com/appvia/wayfinder/pkg/common"
	"github.com/appvia/wayfinder/pkg/utils"
	jwsutils "github.com/appvia/wayfinder/pkg/utils/jwt"
)

// WithIdentity adds a identity to the context, returns a new context
func WithIdentity(ctx context.Context, user Identity) context.Context {
	return context.WithValue(ctx, ContextKey{}, user)
}

// IsWayfinderServiceAccount will return true if the user in the provided context is the Wayfinder
// service account (i.e. this is an action by a controller or webhook but NOT directly tied to a
// Wayfinder user)
func IsWayfinderServiceAccount(ctx context.Context) bool {
	user, found := GetIdentity(ctx)
	return found && user.Username() == fmt.Sprintf("system:serviceaccount:%s:%s", common.Namespace, common.WayfinderServiceAccount)
}

// IsWayfinderOrKubeSystem will return true of the user in the provided context is the Wayfinder
// service account or a kubernetes system service account. This is useful in webhooks to permit
// Wayfinder and kube system controllers to bypass certain validations/mutations.
func IsWayfinderOrKubeSystem(ctx context.Context) bool {
	user, found := GetIdentity(ctx)
	return found && (user.Username() == fmt.Sprintf("system:serviceaccount:%s:%s", common.Namespace, common.WayfinderServiceAccount) ||
		strings.HasPrefix(user.Username(), "system:serviceaccount:kube-system:"))
}

// GetIdentity returns the identity from the context if any
func GetIdentity(ctx context.Context) (Identity, bool) {
	v := ctx.Value(ContextKey{})
	if v == nil {
		return nil, false
	}

	return v.(Identity), true
}

// MustGetIdentity returns the identity from the context if any
func MustGetIdentity(ctx context.Context) Identity {
	v := ctx.Value(ContextKey{})
	if v == nil {
		panic("no user identity found in the context")
	}

	return v.(Identity)
}

// IsTokenExpired checks the token for expiration
func IsTokenExpired(t string) (bool, error) {
	if t == "" {
		return true, nil
	}

	token, err := jwsutils.NewClaimsFromRawToken(t)
	if err != nil {
		return false, err
	}

	return token.HasExpired(), nil
}

// IsExchangeToken checks if the token is for exchange
func IsExchangeToken(token []byte) (bool, error) {
	claims, err := jwsutils.NewClaimsFromRawBytesToken(token)
	if err != nil {
		return false, err
	}

	return IsExchangeScoped(claims), nil
}

// AccessTokenPrefix is the prefix used for access token usernames
func AccessTokenPrefix() string {
	return ScopeAccessToken
}

// IsExchangeScoped checks if the has the scopes for a platform or workspace access token exchange
func IsExchangeScoped(claims *jwsutils.Claims) bool {
	scopes, found := claims.GetScopes()
	if !found {
		return false
	}

	return utils.Contains(ScopeExchange, scopes)
}

// GetWayfinderKubernetesGroup returns the representation of the specified Wayfinder group as
// presented to Kubernetes RBAC for users with this group
func GetWayfinderKubernetesGroup(groupName string) string {
	return fmt.Sprintf("wayfinder:global:%s", groupName)
}

// GetWorkspaceKubernetesGroup returns the representation of the specified workspace group as
// presented to Kubernetes RBAC for users with this group
func GetWorkspaceKubernetesGroup(ws v1alpha1.WorkspaceKey, groupName string) string {
	return fmt.Sprintf("wayfinder:workspace:%s:%s", ws, groupName)
}

// GetKubernetesGroups returns the groups as presented to Kubernetes RBAC for this user identity
func GetKubernetesGroups(id Identity) []string {
	groups := []string{}

	for _, g := range id.WayfinderGroups() {
		groups = append(groups, GetWayfinderKubernetesGroup(g))
	}

	for _, ws := range id.Workspaces() {
		for _, g := range id.WorkspaceGroups(ws) {
			groups = append(groups, GetWorkspaceKubernetesGroup(ws, g))
		}
	}

	return groups
}
