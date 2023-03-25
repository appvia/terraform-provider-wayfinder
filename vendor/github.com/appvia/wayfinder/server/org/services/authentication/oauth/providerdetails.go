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
	"fmt"
	"regexp"
	"strings"

	"github.com/appvia/wayfinder/pkg/utils/validation"
	"github.com/appvia/wayfinder/server/org/services/authentication/oauth/help"
)

type OAuthProviderDetails struct {
	// ID is the unique identifier used to identify this OAuth provider in Wayfinder
	ID string `json:"id"`
	// Name is the name of this OAuth provider
	Name string `json:"name"`
	// DefaultConfig provides a set of configuration values to use as defaults for this provider.
	// The included ServerURL may reference variables from the ServerURLParameters map
	DefaultConfig Config `json:"defaultConfig"`
	// ServerURLParameters provides a map of parameter name to description for the parameters this
	// provider needs in its ServerURL
	ServerURLParameters map[string]ServerURLParameter `json:"serverURLParameters,omitempty"`
	// DocsLink is where to find additional information on configuring this authentication provider
	// for Wayfinder
	DocsLink string `json:"docsLink,omitempty"`
	// Help is a markdown-formatted block of instructions on how to prepare the parameters for this
	// auth provider
	Help string `json:"help,omitempty"`
}

func (o OAuthProviderDetails) GetServerURL(params map[string]string) (string, error) {
	verr := validation.NewError("invalid details")
	if o.DefaultConfig.ServerURL == "" {
		return "", verr.WithFieldError("provider", validation.InvalidValue, "No template server URL available for this provider")
	}

	serverURL := o.DefaultConfig.ServerURL
	for paramKey, paramDetails := range o.ServerURLParameters {
		val, ok := params[paramKey]
		if !ok {
			verr.AddFieldErrorf(fmt.Sprintf("providerServerURLParameters.%s", paramKey), validation.Required, "You must provide a value for %s", paramKey)
			continue
		}
		if ok, err := regexp.MatchString(paramDetails.Pattern, val); err != nil {
			// All patterns are hard-coded and covered by a unit test, so this should not occur. If it does, it's nice
			// to not explode though.
			verr.AddFieldErrorf(fmt.Sprintf("providerServerURLParameters.%s", paramKey), validation.InvalidValue, "Invalid pattern in template for field %s - cannot validate", paramKey)
			continue
		} else if !ok {
			verr.AddFieldErrorf(fmt.Sprintf("providerServerURLParameters.%s", paramKey), validation.InvalidValue, "Value must match pattern %s", paramDetails.Pattern)
			continue
		}

		serverURL = strings.Replace(serverURL, fmt.Sprintf("{%s}", paramKey), val, -1)
	}

	if verr.HasErrors() {
		return "", verr
	}

	return serverURL, nil
}

type ServerURLParameter struct {
	Name        string `json:"name"`
	Pattern     string `json:"pattern,omitempty"`
	Description string `json:"description"`
}

type OAuthProviders []OAuthProviderDetails

func (o OAuthProviders) Get(id string) *OAuthProviderDetails {
	for _, p := range o {
		if p.ID == id {
			return &p
		}
	}

	return nil
}

var Providers = OAuthProviders{
	{
		ID:   "auth0",
		Name: "Auth0",
		DefaultConfig: Config{
			ClientID:     "",
			ClientSecret: "",
			ServerURL:    "https://{tenant}.{region}.auth0.com/",
			ClientScopes: []string{"email", "profile"},
			UserClaims:   []string{"preferred_username", "email", "name", "username"},
		},
		ServerURLParameters: map[string]ServerURLParameter{
			"tenant": {Name: "Tenant", Pattern: "^[a-z0-9][a-z0-9-]+[a-z0-9]$", Description: "The Auth0 tenant to use"},
			"region": {Name: "Region", Pattern: "^[a-z]+$", Description: "The region in which your Auth0 tenant is configured, e.g. 'eu'"},
		},
		DocsLink: "/wayfinder/admin/auth/auth0",
		Help:     string(help.MustHelpAsset("auth0")),
	},
	{
		ID:   "google",
		Name: "Google",
		DefaultConfig: Config{
			ClientID:     "",
			ClientSecret: "",
			ServerURL:    "https://accounts.google.com",
			ClientScopes: []string{"email", "profile"},
			UserClaims:   []string{"preferred_username", "email", "name", "username"},
		},
		DocsLink: "/wayfinder/admin/auth/google",
		Help:     string(help.MustHelpAsset("google")),
	},
	{
		ID:   "azure",
		Name: "Azure AD",
		DefaultConfig: Config{
			ClientID:     "",
			ClientSecret: "",
			ServerURL:    "https://login.microsoftonline.com/{tenant}/v2.0",
			ClientScopes: []string{"email", "profile"},
			UserClaims:   []string{"preferred_username", "email", "name", "username"},
		},
		ServerURLParameters: map[string]ServerURLParameter{
			"tenant": {Name: "Tenant ID", Pattern: "^[0-9a-fA-F]{8}-([0-9a-fA-F]{4}-){3}[0-9a-fA-F]{12}$", Description: "Azure Active Directory Tenant ID"},
		},
		DocsLink: "/wayfinder/admin/auth/azure",
		Help:     string(help.MustHelpAsset("azure")),
	},
	{
		ID:   "generic",
		Name: "Generic OIDC Provider",
		DefaultConfig: Config{
			ClientID:     "",
			ClientSecret: "",
			ServerURL:    "",
			ClientScopes: []string{"email", "profile"},
			UserClaims:   []string{"preferred_username", "email", "name", "username"},
		},
		DocsLink: "/wayfinder/admin/auth",
		Help:     string(help.MustHelpAsset("generic")),
	},
}
