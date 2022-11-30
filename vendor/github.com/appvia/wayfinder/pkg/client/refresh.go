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
	"fmt"
	"time"

	"github.com/appvia/wayfinder/pkg/apiserver/types"
	"github.com/appvia/wayfinder/server/org/services/authentication"
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

// ExchangeAccessToken is used to exchange an access token for a valid API token
func ExchangeAccessToken(client Interface, exchange []byte, expiration time.Duration) ([]byte, error) {
	if found, err := authentication.IsExchangeToken(exchange); err != nil {
		return nil, err
	} else if !found {
		return nil, authentication.ErrNonExchangeToken
	}

	token := &types.IssuedToken{}
	if err := client.Request().
		Authorization(string(exchange)).
		Result(token).
		Endpoint("/exchange").
		Parameters(QueryParameter("ttl", expiration.String())).
		Post().
		Error(); err != nil {
		return nil, fmt.Errorf("failed to exchange access token for API token - please check the access token is valid: %w", err)
	}

	return []byte(token.Token), nil
}
