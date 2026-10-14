package auth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

func resolveCLI(cfg Config, getenv func(string) string, explicitServer, tenant string, httpClient *http.Client, now func() time.Time, explicit bool) (*Resolved, error) {
	path, err := cliConfigPath(getenv)
	if err != nil {
		return nil, err
	}
	name := cfg.CLIProfile
	if name == "" {
		name = getenv("WAYFINDER_CLI_PROFILE")
	}
	profile, err := loadCLIProfile(path, name)
	if err != nil {
		if !explicit && errors.Is(err, fs.ErrNotExist) {
			return nil, errCredentialsOrCLI
		}
		return nil, err
	}
	if explicitServer != "" && strings.TrimRight(explicitServer, "/") != strings.TrimRight(profile.Server, "/") {
		return nil, &configurationError{message: fmt.Sprintf("server %q does not match wf CLI profile %q (%q) - unset server / WAYFINDER_SERVER, or choose another cli_profile", explicitServer, profile.Name, profile.Server)}
	}
	if tenant == "" {
		tenant = profile.Tenant
	}
	if len(profile.CAPEM) > 0 && cfg.CLIHTTP != nil {
		httpClient, err = cfg.CLIHTTP(profile.CAPEM)
		if err != nil {
			return nil, fmt.Errorf("could not trust the wf CLI profile's CA certificate: %w", err)
		}
	}
	source := &cachedSource{server: strings.TrimRight(profile.Server, "/"), http: httpClient, now: now}
	var selected TokenSource
	var expired error
	switch {
	case profile.RefreshToken != "":
		source.request = tokenRequest{GrantType: grantRefresh, RefreshToken: profile.RefreshToken}
		selected = source
	case profile.ClientID != "" && profile.ClientSecret != "":
		if tenant == "" {
			tenant, _, _ = strings.Cut(profile.ClientID, ":")
		}
		source.request = tokenRequest{GrantType: grantClient, ClientID: profile.ClientID, ClientSecret: profile.ClientSecret}
		selected = source
	case profile.Federated != nil:
		identity := profile.Federated
		refTenant, err := parseServiceAccountTenant(identity.ServiceAccount)
		if err != nil {
			return nil, fmt.Errorf("invalid service account stored in wf CLI profile %q: %w", profile.Name, err)
		}
		if tenant != "" && tenant != refTenant {
			return nil, &configurationError{message: fmt.Sprintf("tenant %q does not match the tenant of the service account stored in wf CLI profile %q (%q) - they must agree, or leave tenant / WAYFINDER_TENANT unset", tenant, profile.Name, refTenant)}
		}
		tenant = refTenant
		source.identity = &federatedIdentity{
			file: identity.SubjectTokenFile, url: identity.TokenServer,
			token: identity.TokenServerToken, audience: identity.Audience, tenant: tenant,
		}
		source.request = tokenRequest{GrantType: grantExchange, SubjectTokenType: "urn:ietf:params:oauth:token-type:jwt", ServiceAccount: identity.ServiceAccount}
		selected = source
	default:
		claims, _ := parseClaims(profile.Token)
		if claims.refreshable() {
			source.request = tokenRequest{GrantType: grantRefresh, RefreshToken: profile.Token}
			selected = source
		} else {
			expired = &configurationError{message: fmt.Sprintf(`the access token stored in wf CLI profile %q has expired and cannot be renewed - sign in again with "wf login --profile %s"`, profile.Name, profile.Name)}
			selected = bearerSource{token: profile.Token, expires: claims.Exp, now: now, expired: expired}
		}
	}
	if tenant == "" {
		return nil, errTenant
	}
	ctx := cfg.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if _, err := selected.Token(ctx); err != nil {
		if expired != nil && err == expired {
			return nil, err
		}
		return nil, fmt.Errorf(`could not authenticate with wf CLI profile %q: %w - check the profile's server is reachable, or sign in again with "wf login --profile %s"`, profile.Name, err, profile.Name)
	}
	return &Resolved{Server: profile.Server, Tenant: tenant, TokenSource: selected, HTTP: httpClient}, nil
}
