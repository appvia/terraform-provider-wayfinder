// Package auth resolves provider authentication without reading local CLI profiles.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Config contains provider attributes. Empty strings fall back to environment variables.
// Getenv, HTTP and Now optionally inject environment, transport and clock dependencies.
type Config struct {
	Server, Tenant, Token, ClientID, ClientSecret, ServiceAccount string
	FederatedTokenFile, FederatedTokenRequestURL                  string
	FederatedTokenRequestToken, FederatedTokenRequestAudience     string
	Getenv                                                        func(string) string
	HTTP                                                          *http.Client
	Now                                                           func() time.Time
}

// TokenSource is structurally compatible with the REST client's token-source contract.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// Resolved contains the endpoint, tenant and concurrency-safe authentication source.
type Resolved struct {
	Server, Tenant string
	TokenSource    TokenSource
}

var (
	errCredentials = errors.New("set token, service_account with a federated token source, or client_id and client_secret in the provider block; alternatively set WAYFINDER_TOKEN, WAYFINDER_SERVICE_ACCOUNT, or WAYFINDER_CLIENT_ID and WAYFINDER_CLIENT_SECRET")
	errTenant      = &configurationError{message: "Set tenant in the provider block or WAYFINDER_TENANT."}
)

// Resolve mirrors pinned pkg/wfclient/config/helper.go:353-435. Resource workspace
// and environment are deliberately not inferred; CLI profiles are never read.
func Resolve(cfg Config) (*Resolved, error) {
	getenv := cfg.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	value := func(attr, name string) string {
		if attr != "" {
			return attr
		}
		return getenv(name)
	}
	server := value(cfg.Server, "WAYFINDER_SERVER")
	if server == "" {
		server = "https://api.on.wayfinder.run"
	}
	tenant := value(cfg.Tenant, "WAYFINDER_TENANT")
	token := value(cfg.Token, "WAYFINDER_TOKEN")
	clientID := value(cfg.ClientID, "WAYFINDER_CLIENT_ID")
	secret := value(cfg.ClientSecret, "WAYFINDER_CLIENT_SECRET")
	account := value(cfg.ServiceAccount, "WAYFINDER_SERVICE_ACCOUNT")
	httpClient := cfg.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	source := &cachedSource{server: strings.TrimRight(server, "/"), http: httpClient, now: now}
	var selected TokenSource
	switch {
	case token != "":
		claims, _ := parseClaims(token)
		if claims.refreshable() {
			source.request = tokenRequest{GrantType: grantRefresh, RefreshToken: token}
			selected = source
		} else {
			selected = bearerSource{token: token, expires: claims.Exp, now: now}
		}
	case account != "":
		refTenant, err := parseServiceAccountTenant(account)
		if err != nil {
			return nil, fmt.Errorf("invalid service_account / WAYFINDER_SERVICE_ACCOUNT: %w", err)
		}
		if tenant != "" && tenant != refTenant {
			return nil, &configurationError{message: fmt.Sprintf("tenant / WAYFINDER_TENANT (%q) does not match the tenant in service_account / WAYFINDER_SERVICE_ACCOUNT (%q) - they must agree, or leave tenant / WAYFINDER_TENANT unset", tenant, refTenant)}
		}
		tenant = refTenant
		identity := federatedIdentity{
			file:     value(cfg.FederatedTokenFile, "WAYFINDER_FEDERATED_TOKEN_FILE"),
			url:      value(cfg.FederatedTokenRequestURL, "WAYFINDER_FEDERATED_TOKEN_REQUEST_URL"),
			token:    value(cfg.FederatedTokenRequestToken, "WAYFINDER_FEDERATED_TOKEN_REQUEST_TOKEN"),
			audience: value(cfg.FederatedTokenRequestAudience, "WAYFINDER_FEDERATED_TOKEN_REQUEST_AUDIENCE"),
			tenant:   tenant,
		}
		if identity.url == "" && getenv("ACTIONS_ID_TOKEN_REQUEST_URL") != "" && getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN") != "" {
			identity.url, identity.token = getenv("ACTIONS_ID_TOKEN_REQUEST_URL"), getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
		}
		if identity.file == "" && identity.url == "" {
			return nil, &configurationError{message: "service_account / WAYFINDER_SERVICE_ACCOUNT is set, but no identity token source is configured - set federated_token_file / WAYFINDER_FEDERATED_TOKEN_FILE or federated_token_request_url / WAYFINDER_FEDERATED_TOKEN_REQUEST_URL, or run inside GitHub Actions with 'permissions: id-token: write'"}
		}
		source.identity = &identity
		source.request = tokenRequest{GrantType: grantExchange, SubjectTokenType: "urn:ietf:params:oauth:token-type:jwt", ServiceAccount: account}
		selected = source
	case clientID != "" && secret != "":
		if tenant == "" {
			tenant, _, _ = strings.Cut(clientID, ":")
		}
		source.request = tokenRequest{GrantType: grantClient, ClientID: clientID, ClientSecret: secret}
		selected = source
	default:
		return nil, errCredentials
	}
	if tenant == "" {
		return nil, errTenant
	}
	return &Resolved{Server: server, Tenant: tenant, TokenSource: selected}, nil
}

type configurationError struct{ message string }

func (e *configurationError) Error() string { return e.message }

// Grammar and errors mirror pinned server/org/apis/org/v3/serviceaccount_ref.go:72-97.
func parseServiceAccountTenant(ref string) (string, error) {
	invalid := func(message string) (string, error) { return "", &configurationError{message: message} }
	if strings.TrimSpace(ref) == "" {
		return invalid("a service account reference is required")
	}
	if strings.Contains(ref, "/") {
		return invalid(fmt.Sprintf("invalid service account reference %q: separate tenant, workspace and name with ':' (e.g. tenant:workspace:name), not '/'", ref))
	}
	parts := strings.Split(ref, ":")
	for _, p := range parts {
		if p == "" {
			return invalid(fmt.Sprintf("invalid service account reference %q: segments must not be empty", ref))
		}
	}
	switch len(parts) {
	case 2, 3:
		return parts[0], nil
	case 4:
		return invalid(fmt.Sprintf("environment-scoped service accounts (%q) are not yet supported", ref))
	case 1:
		return invalid(fmt.Sprintf("invalid service account reference %q: must be tenant-qualified, in the form tenant:name or tenant:workspace:name", ref))
	default:
		return invalid(fmt.Sprintf("invalid service account reference %q: expected tenant:name or tenant:workspace:name", ref))
	}
}
