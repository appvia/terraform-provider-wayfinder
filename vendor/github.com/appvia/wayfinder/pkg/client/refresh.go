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

package client

import (
	"errors"
	"time"

	corev1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	"github.com/appvia/wayfinder/pkg/apiserver/types"
	"github.com/appvia/wayfinder/pkg/authentication"
	"github.com/appvia/wayfinder/pkg/utils"
	jwsutils "github.com/appvia/wayfinder/pkg/utils/jwt"
	policyutil "github.com/appvia/wayfinder/server/policydelivery/services/utils"
)

// RefreshWayfinderIdentityToken is used to exchange the refresh token for a new access token
func RefreshWayfinderIdentityToken(client Interface, refresh []byte) ([]byte, error) {
	issued := &types.IssuedToken{}

	err := client.Request().
		Authorization(string(refresh)).
		Endpoint("/login/token").
		Payload(&types.IssuedToken{RefreshToken: string(refresh)}).
		Result(issued).
		Post().
		Error()

	if err != nil {
		return nil, err
	}

	return []byte(issued.Token), nil
}

// ExchangeRobotToken is used to exchange a robot token for us
func ExchangeRobotToken(client Interface, exchange []byte, expiration time.Duration) ([]byte, error) {
	if found, err := authentication.IsExchangeToken(exchange); err != nil {
		return nil, err
	} else if !found {
		return nil, authentication.ErrNonExchangeToken
	}

	claims, err := jwsutils.NewClaimsFromRawBytesToken(exchange)
	if err != nil {
		return nil, err
	}

	groups, found := claims.GetGroups()
	if !found {
		return nil, errors.New("no groups claim in the token")
	}
	if len(groups) <= 0 {
		return nil, errors.New("groups claim is empty")
	}
	if len(groups) > 1 {
		return nil, errors.New("groups claim has multiple names")
	}

	scopes, found := claims.GetScopes()
	if !found {
		return nil, errors.New("invalid token, no scopes found")
	}

	name, found := claims.GetPreferredUsername()
	if !found {
		return nil, errors.New("robot subject name not found in token")
	}

	// @step: ensure the token has the exchange scope
	if !utils.ContainsMatchingItems([]string{authentication.ScopeExchange}, scopes) {
		return nil, errors.New("token cannot used for exchange")
	}

	token := &types.IssuedToken{}
	err = client.Request().
		Authorization(string(exchange)).
		Result(token).
		Workspace(corev1.WorkspaceKey(groups[0])).
		Name(policyutil.GetRobotAccountName(name)).
		Resource("robots").
		SubResource("exchange").
		Parameters(QueryParameter("ttl", expiration.String())).
		Post().
		Error()

	return []byte(token.Token), err
}
