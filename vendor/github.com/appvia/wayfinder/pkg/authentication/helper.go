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
	"errors"

	"github.com/appvia/wayfinder/pkg/utils"
	jwsutils "github.com/appvia/wayfinder/pkg/utils/jwt"
	policyv1alpha1 "github.com/appvia/wayfinder/server/policydelivery/apis/policy/v1alpha1"
)

// WithIdentity adds a identity to the context, returns a new context
func WithIdentity(ctx context.Context, user Identity) context.Context {
	return context.WithValue(ctx, ContextKey{}, user)
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

// MustGetAdmissionRequest returns the admission request from context
func MustGetAdmissionRequest(ctx context.Context) *policyv1alpha1.AdmissionRequest {
	v := ctx.Value(ContextAdmissionKey{})
	if v == nil {
		panic("no admission request found in the context")
	}

	return v.(*policyv1alpha1.AdmissionRequest)
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

	return IsExchangeScoped(claims)
}

// IsRobotAccount checks if the token is a robot account
func IsRobotAccount(claims *jwsutils.Claims) bool {
	scopes, found := claims.GetScopes()
	if !found {
		return false
	}

	return utils.Contains(ScopeRobot, scopes)
}

// RobotAccountPrefix is the prefix used for Wayfinder robot account names
func RobotAccountPrefix() string {
	return ScopeRobot
}

// IsExchangeScoped checks if the has the correct audience and scopes
func IsExchangeScoped(claims *jwsutils.Claims) (bool, error) {
	aud, found := claims.GetAudience()
	if !found {
		return false, errors.New("token has no audience")
	}
	if aud != Audience {
		return false, nil
	}

	scopes, found := claims.GetScopes()
	if !found {
		return false, nil
	}

	return utils.Contains(ScopeExchange, scopes), nil
}
