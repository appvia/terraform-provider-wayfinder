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

package openid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/appvia/wayfinder/pkg/utils/httputils"
)

// TokenRefresher implements the token refreshing
type TokenRefresher interface {
	// RefreshToken is responsible for refreshing the token
	RefreshToken(ctx context.Context, refresh string, endpoint string, cliendid string, secret string) (string, error)
}

// RefreshResponse is the expected response from the refresh
type RefreshResponse struct {
	// AccessToken is the access token provided
	AccessToken string `json:"access_token,omitempty" yaml:"access_token"`
	// IDToken string is the identity token
	IDToken string `json:"id_token,omitempty" yaml:"id_token"`
}

type tokenImpl struct {
	hc *http.Client
}

var (
	// DefaultTokenRefresher provides a token refresher to all
	DefaultTokenRefresher = &tokenImpl{hc: httputils.DefaultHTTPClient}
)

// RefreshToken implements the token
func (t *tokenImpl) RefreshToken(ctx context.Context, refresh, endpoint, id, secret string) (*RefreshResponse, error) {
	params := url.Values{
		"client_id":     {id},
		"client_secret": {secret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
	}
	resp, err := t.hc.PostForm(endpoint, params)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected response from the token endpoint: %d", resp.StatusCode)
	}

	token := &RefreshResponse{}
	if err := json.NewDecoder(resp.Body).Decode(token); err != nil {
		return nil, err
	}

	return token, nil
}
