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
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"

	"github.com/appvia/wayfinder/pkg/common"
	"github.com/appvia/wayfinder/pkg/utils"
	"github.com/appvia/wayfinder/pkg/utils/jwt"
)

func New() *OAuthLoginProvider {
	return &OAuthLoginProvider{
		lock: &sync.RWMutex{},
	}
}

type OAuthLoginProvider struct {
	lock       *sync.RWMutex
	config     *Config
	oidcConfig *oauth2.Config
	provider   *oidc.Provider
	verifier   *oidc.IDTokenVerifier

	// isGoogle indicates if we're using gsuite auth, as it has some weird behaviours
	isGoogle bool
}

const googleAuthURL = "https://accounts.google.com"

// GetActiveConfig returns the active IDP configuration. This will be nil if the IDP integration is
// currently not enabled
func (l *OAuthLoginProvider) GetActiveConfig() *Config {
	return l.config
}

func (l *OAuthLoginProvider) Configure(ctx context.Context, config *Config) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.config = config
	if l.config == nil || l.config.Empty() {
		// Disabled, clear any old configuration
		l.oidcConfig = nil
		l.provider = nil
		l.verifier = nil
		return nil
	}

	provider, err := oidc.NewProvider(context.Background(), l.config.ServerURL)
	if err != nil {
		// Set config as nil to disable the provider
		l.config = nil
		return fmt.Errorf("failed to create OpenID provider: %w", err)
	}
	l.provider = provider

	// Configure an OpenID Connect aware OAuth2 client.
	scopes := append([]string{oidc.ScopeOpenID}, l.config.ClientScopes...)
	l.isGoogle = strings.HasPrefix(l.config.ServerURL, googleAuthURL)
	if l.isGoogle {
		// Remove offline scope if present as it causes failures on google.
		scopes = func() (ret []string) {
			for _, s := range scopes {
				if s != oidc.ScopeOfflineAccess {
					ret = append(ret, s)
				}
			}
			return
		}()
	}
	l.oidcConfig = &oauth2.Config{
		ClientID:     l.config.ClientID,
		ClientSecret: l.config.ClientSecret,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}

	l.verifier = provider.Verifier(&oidc.Config{ClientID: l.config.ClientID})

	return nil
}

func (l *OAuthLoginProvider) Enabled() bool {
	return l != nil && l.config != nil && !l.config.Empty() && l.oidcConfig != nil
}

func (l *OAuthLoginProvider) AuthRedirect(ctx context.Context, state string, callbackURL string) string {
	l.lock.RLock()
	defer l.lock.RUnlock()

	if l.oidcConfig == nil {
		panic("AuthRedirect called on a disabled provider - check Enabled() before calling this")
	}

	l.oidcConfig.RedirectURL = callbackURL

	var authCodeURL string
	if l.isGoogle {
		// Google has different ideas about how to request a refresh token.
		authCodeURL = l.oidcConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	} else {
		authCodeURL = l.oidcConfig.AuthCodeURL(state)
	}

	return authCodeURL
}

type CallbackResult struct {
	Issuer   string
	Subject  string
	Username string
	Email    string
}

// HandleCallback handles the callback - it **must** be passed the same callbackURL as was passed to
// AuthRedirect else bad things happen
func (l *OAuthLoginProvider) HandleCallback(ctx context.Context, code string, callbackURL string) (CallbackResult, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	if l.oidcConfig == nil {
		panic("HandleCallback called on a disabled provider - check Enabled() before calling this")
	}

	l.oidcConfig.RedirectURL = callbackURL

	// @step: exchange the authorization code with the identity provider
	otoken, err := l.oidcConfig.Exchange(ctx, code)
	if err != nil {
		common.Log(ctx).WithError(err).Errorf("failed to exchange token with IDP %s (client ID: %s)", l.config.Provider, l.config.ClientID)

		return CallbackResult{}, fmt.Errorf("failed to exchange token with IDP: %w", err)
	}

	// @step: extract the id token from oauth2 token
	rawToken, ok := otoken.Extra("id_token").(string)
	if !ok {
		return CallbackResult{}, errors.New("token from IDP response is missing identity token")
	}

	// @step: parse and verify id token payload
	token, err := l.verifier.Verify(ctx, rawToken)
	if err != nil {
		common.Log(ctx).WithError(err).Error("failed to verify token from IDP")

		return CallbackResult{}, errors.New("unable to verify token from IDP")
	}

	claims, err := jwt.NewClaimsFromToken(token)
	if err != nil {
		common.Log(ctx).WithError(err).Error("unable to parse claims from IDP token")

		return CallbackResult{}, errors.New("failed to parse claims from IDP token")
	}

	// @step: extract the username and email address
	username, found := claims.GetUserClaim(l.config.UserClaims...)
	if !found {
		return CallbackResult{}, errors.New("token from IDP did not include any of the configured User Claims")
	}

	// @step: check if groups are enabled and provided
	if len(l.config.RequiredIDPGroups) > 0 {
		// Wayfinder is configured to use groups, so we need to check the user is in the right group
		groups, found := claims.GetStringSlice("groups")
		if !found {
			common.Log(ctx).WithField("user", username).Error("user does not contain any IDP groups allowed to access Wayfinder")

			return CallbackResult{}, fmt.Errorf("user does not contain any IDP groups allowed to access Wayfinder")
		}
		if !utils.StringsSliceIntersectEqualFold(groups, l.config.RequiredIDPGroups) {
			common.Log(ctx).WithField("user", username).Error("user is not in the configured IDP groups")

			return CallbackResult{}, fmt.Errorf("user is in any IDP groups allowed to access Wayfinder")
		}
	}

	email, _ := claims.GetEmail()
	// Making this optional so not returning an error any more here...
	// if !found {
	// 	return CallbackResult{}, errors.New("token from IDP did not include the email address in the provided claims")
	// }

	return CallbackResult{
		Issuer:   token.Issuer,
		Subject:  token.Subject,
		Username: username,
		Email:    email,
	}, nil
}

// ValidateWayfinderRedirectURL will return true if the provided redirect URL is valid based on the
// allowed redirect URL hosts provided on this provider
func (l *OAuthLoginProvider) ValidateWayfinderRedirectURL(redirectURL *url.URL) (bool, error) {
	if len(l.config.AllowedWayfinderRedirectURLHosts) == 0 {
		return false, nil
	}

	for _, host := range l.config.AllowedWayfinderRedirectURLHosts {
		if len(host) < 4 {
			// Ignore very short values, don't want any global wildcards
			continue
		}
		if strings.HasSuffix(host, "*") || strings.HasSuffix(host, "?") {
			// Ignore anything that ends in a wildcard, not secure
			continue
		}
		valid, err := path.Match(host, redirectURL.Hostname())
		if err != nil {
			return false, fmt.Errorf("unable to match hostname from AllowedWayfinderRedirectURLHosts - %w", err)
		}
		if valid {
			return true, nil
		}
	}

	return false, nil
}
