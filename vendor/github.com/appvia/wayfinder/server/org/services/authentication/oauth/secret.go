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

package oauth

import (
	"encoding/json"
	"strings"

	v1 "k8s.io/api/core/v1"
)

const (
	ServerURLSecretField                  = "WF_IDP_SERVER_URL"
	ClientScopesSecretField               = "WF_IDP_CLIENT_SCOPES"
	ClientIDSecretField                   = "WF_IDP_CLIENT_ID"
	ClientSecretSecretField               = "WF_IDP_CLIENT_SECRET"
	UserClaimsSecretField                 = "WF_IDP_USER_CLAIMS"
	RequiredGroupsSecretField             = "WF_IDP_REQUIRED_GROUPS_CLAIMS"
	ProviderSecretField                   = "WF_IDP_PROVIDER"
	ProviderParamsSecretField             = "WF_IDP_PROVIDER_PARAMS"
	AllowedWayfinderRedirectURLHostsField = "WF_IDP_ALLOWED_REDIRECT_HOSTS"
)

// DecodeSecret converts a provided secret into the configuration for the OAuth provider
func DecodeSecret(secret *v1.Secret) Config {
	config := Config{}

	if secret == nil {
		return config
	}

	if v, ok := secret.Data[ServerURLSecretField]; ok {
		config.ServerURL = string(v)
	}
	if v, ok := secret.Data[ClientScopesSecretField]; ok {
		config.ClientScopes = strings.Split(string(v), ",")
	}
	if v, ok := secret.Data[ClientIDSecretField]; ok {
		config.ClientID = string(v)
	}
	if v, ok := secret.Data[ClientSecretSecretField]; ok {
		config.ClientSecret = string(v)
	}
	if v, ok := secret.Data[UserClaimsSecretField]; ok {
		config.UserClaims = strings.Split(string(v), ",")
	}
	if v, ok := secret.Data[ProviderSecretField]; ok {
		config.Provider = string(v)
	}
	if v, ok := secret.Data[RequiredGroupsSecretField]; ok {
		config.RequiredIDPGroups = strings.Split(string(v), ",")
	}
	if v, ok := secret.Data[AllowedWayfinderRedirectURLHostsField]; ok {
		config.AllowedWayfinderRedirectURLHosts = strings.Split(string(v), ",")
	}
	if v, ok := secret.Data[ProviderParamsSecretField]; ok {
		params := map[string]string{}
		// Ignore errors decoding the map, for now...
		if err := json.Unmarshal([]byte(v), &params); err == nil {
			config.ProviderParameters = params
		}
	}

	return config
}

// SetSecretConfig writes the provided config into the secret StringData, overwriting the client
// secret ONLY if a new value is provided.
func SetSecretConfig(config Config, secret *v1.Secret) {
	params := ""
	if config.ProviderParameters != nil {
		if bytes, err := json.Marshal(config.ProviderParameters); err == nil {
			params = string(bytes)
		}
	}
	secret.StringData = map[string]string{
		ServerURLSecretField:      config.ServerURL,
		ClientScopesSecretField:   strings.Join(config.ClientScopes, ","),
		ClientIDSecretField:       config.ClientID,
		UserClaimsSecretField:     strings.Join(config.UserClaims, ","),
		ProviderSecretField:       config.Provider,
		ProviderParamsSecretField: params,
	}
	// Only overwrite the client secret if a new one is provided
	if config.ClientSecret != "" {
		secret.StringData[ClientSecretSecretField] = config.ClientSecret
	}
	// Only overwrite the allowed redirect hosts if a new value is provided
	if len(config.AllowedWayfinderRedirectURLHosts) > 0 {
		secret.StringData[AllowedWayfinderRedirectURLHostsField] = strings.Join(config.AllowedWayfinderRedirectURLHosts, ",")
	}
	if len(config.RequiredIDPGroups) > 0 {
		secret.StringData[RequiredGroupsSecretField] = strings.Join(config.RequiredIDPGroups, ",")
	}
}
