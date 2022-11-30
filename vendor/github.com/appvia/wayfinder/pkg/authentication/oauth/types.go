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

type Config struct {
	// Provider is an optional provider name. This should match one of the supported provider names
	// from the oauth.Providers struct. If Provider (and any requiredProviderParameters) are set, you
	// can leave ClientScopes, UserClaims and ServerURL blank.
	Provider string `json:"provider,omitempty"`
	// ProviderParameters provides the set of parameters required by the provider to form
	// the server URL.
	ProviderParameters map[string]string `json:"providerParameters,omitempty"`
	// ClientID is the client id for the openid authenticator
	ClientID string `json:"clientID,omitempty"`
	// ClientScopes are additional scopes to add to the request
	ClientScopes []string `json:"clientScopes,omitempty"`
	// ClientSecret is the client secret to use.
	// NOTE: this will not be populated when retrieving this object from the API, and if unspecified
	// when providing an update, its value will be left at the current value.
	ClientSecret string `json:"clientSecret,omitempty"`
	// ServerURL is the openid server url
	ServerURL string `json:"serverURL,omitempty"`
	// UserClaims is collection of claims to identify the username
	UserClaims []string `json:"userClaims,omitempty"`
	// AllowedWayfinderRedirectURLHosts provides a set of additional hosts that Wayfinder can
	// redirect to after completing oauth login flows with this provider. Wayfinder will always
	// allow redirects to localhost and the configured public UI URL.
	// Supports ? and * wildcards but must be >3 characters and cannot end with a wildcard
	AllowedWayfinderRedirectURLHosts []string `json:"allowedRedirectURLHosts,omitempty"`
}

// Empty returns true if all the fields are empty - i.e. the IDP config represents the 'disabled'
// state
func (c Config) Empty() bool {
	return c.ClientID == "" &&
		len(c.ClientScopes) == 0 &&
		c.ClientSecret == "" &&
		c.ServerURL == "" &&
		len(c.UserClaims) == 0
}

type ConfigSource interface {
	// GetConfig should return the current configuration or nil if the provider is disabled
	GetConfig() *Config
	// RegisterConfigWatcher registers a function to be called back when the config is updated. The
	// provided function will be called whenever the Config is updated - call GetConfig to get the
	// latest version of the confic when notified
	RegisterConfigWatcher(ConfigWatcher)
}

type ConfigWatcher func()
