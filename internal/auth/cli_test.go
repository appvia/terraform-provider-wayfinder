// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/auth"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

func cliBool(value bool) *bool { return &value }

func cliTokenExpiry(t *testing.T, token string) time.Time {
	t.Helper()

	payload, err := base64.RawURLEncoding.DecodeString(strings.Split(token, ".")[1])
	if err != nil {
		t.Fatal(err)
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatal(err)
	}

	return time.Unix(claims.Exp, 0)
}

func cliEnv(cfg *auth.Config, values map[string]string) {
	previous := cfg.Getenv
	cfg.Getenv = func(k string) string {
		if value, ok := values[k]; ok {
			return value
		}

		return previous(k)
	}
}

func cliEditFixture(t *testing.T, path string, edit func(map[string]any)) {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var profile map[string]any
	if err := json.Unmarshal(body, &profile); err != nil {
		t.Fatal(err)
	}

	edit(profile)

	body, err = json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
}

func cliFixture(t *testing.T, server, tenant string, user map[string]any) (auth.Config, string) {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"current-profile": "test",
		"users":           map[string]any{"alice": user},
		"profiles":        map[string]any{"test": map[string]string{"user": "alice", "server": "api", "tenant": tenant}},
		"servers":         map[string]any{"api": map[string]string{"server": server}},
	})
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "profiles")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}

	return auth.Config{Getenv: func(k string) string {
		if k == "WAYFINDER_CONFIG" {
			return path
		}

		return ""
	}}, path
}

func TestCLIAuth_whenRefreshProfileAuthenticatesEagerly(t *testing.T) {
	t.Parallel()
	// Given
	s := fakeapi.New(t)
	s.AddRefreshToken("cli-refresh")
	cfg, _ := cliFixture(t, s.URL, "acme", map[string]any{"identity": map[string]string{"refresh-token": "cli-refresh"}})
	// When
	r, err := auth.Resolve(cfg)
	// Then
	if err != nil {
		t.Fatal(err)
	}

	if s.Mints(fakeapi.GrantRefreshToken) != 1 || r.Server != s.URL || r.Tenant != "acme" {
		t.Fatal("CLI profile was not authenticated eagerly")
	}

	token, err := r.TokenSource.Token(context.Background())
	if err != nil || token == "" || token == "cli-refresh" || s.Mints(fakeapi.GrantRefreshToken) != 1 {
		t.Fatalf("eager token was not cached: %v", err)
	}
}

func TestCLIAuth_whenRefreshBoundaryHasConcurrentCallers(t *testing.T) {
	t.Parallel()
	// Given
	s := fakeapi.New(t)
	s.AddRefreshToken("cli-refresh")
	s.SetTokenTTL(5 * time.Minute)

	now := time.Now().Truncate(time.Second)
	cfg, path := cliFixture(t, s.URL, "acme", map[string]any{"identity": map[string]string{
		"refresh-token": "cli-refresh", "token": "must-not-use-cached-profile-token",
		"client-id": "wrong", "client-secret": "wrong",
	}})
	cfg.Now = func() time.Time { return now }

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	r, err := auth.Resolve(cfg)
	if err != nil {
		t.Fatal(err)
	}

	first, err := r.TokenSource.Token(context.Background())
	if err != nil || s.Mints(fakeapi.GrantRefreshToken) != 1 {
		t.Fatalf("eager token was not reused: %v", err)
	}
	// The fake API uses its own clock; extend the next token beyond the injected clock.
	s.SetTokenTTL(15 * time.Minute)
	now = cliTokenExpiry(t, first).Add(-time.Minute)
	start := make(chan struct{})
	tokens := make(chan string, 32)

	var wg sync.WaitGroup
	// When
	for range 32 {
		wg.Go(func() {
			<-start

			token, err := r.TokenSource.Token(context.Background())
			if err != nil {
				t.Error(err)
				return
			}

			tokens <- token
		})
	}

	close(start)
	wg.Wait()
	close(tokens)
	// Then
	var refreshed string

	count := 0

	for token := range tokens {
		if token == "" || token == first || (refreshed != "" && token != refreshed) {
			t.Fatal("callers did not receive the same refreshed token")
		}

		refreshed = token
		count++
	}

	if count != 32 || s.Mints(fakeapi.GrantRefreshToken) != 2 {
		t.Fatalf("got %d callers and %d mints", count, s.Mints(fakeapi.GrantRefreshToken))
	}

	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("profiles file changed: %v", err)
	}
}

func TestCLIAuth_whenRefreshProfileExpires(t *testing.T) {
	t.Parallel()
	// Given
	s := fakeapi.New(t)
	s.AddRefreshToken("cli-refresh")
	s.SetTokenTTL(5 * time.Minute)

	now := time.Now()
	cfg, _ := cliFixture(t, s.URL, "acme", map[string]any{"identity": map[string]string{"refresh-token": "cli-refresh"}})
	cfg.Now = func() time.Time { return now }

	r, err := auth.Resolve(cfg)
	if err != nil {
		t.Fatal(err)
	}

	first, err := r.TokenSource.Token(context.Background())
	if err != nil || s.Mints(fakeapi.GrantRefreshToken) != 1 {
		t.Fatalf("eager token not cached: %v", err)
	}

	s.ExpireMintedTokens()
	s.SetTokenTTL(15 * time.Minute)

	now = cliTokenExpiry(t, first).Add(time.Second)
	// When
	refreshed, err := r.TokenSource.Token(context.Background())
	// Then
	if err != nil || refreshed == first || refreshed == "" || s.Mints(fakeapi.GrantRefreshToken) != 2 {
		t.Fatalf("expired CLI token not refreshed: %v", err)
	}

	reused, err := r.TokenSource.Token(context.Background())
	if err != nil || reused != refreshed || s.Mints(fakeapi.GrantRefreshToken) != 2 {
		t.Fatalf("refreshed token not cached: %v", err)
	}
}

func TestCLIAuth_whenProfileSelected(t *testing.T) {
	t.Parallel()

	for _, selection := range []string{"current", "attribute", "environment"} {
		t.Run(selection, func(t *testing.T) {
			t.Parallel()
			// Given
			cfg, path := cliFixture(t, "https://selected.example", "acme", map[string]any{"token": "static"})
			if selection != "current" {
				cliEditFixture(t, path, func(doc map[string]any) { doc["current-profile"] = "absent" })
			}

			if selection == "attribute" {
				cfg.CLIProfile = "test"
				cliEnv(&cfg, map[string]string{"WAYFINDER_CLI_PROFILE": "absent"})
			}

			if selection == "environment" {
				cliEnv(&cfg, map[string]string{"WAYFINDER_CLI_PROFILE": "test"})
			}
			// When
			r, err := auth.Resolve(cfg)
			// Then
			if err != nil || r.Server != "https://selected.example" || r.Tenant != "acme" {
				t.Fatalf("selected profile: %v", err)
			}
		})
	}
}

func TestCLIAuth_whenExplicitCredentialsOrPartialCredentialsProvided(t *testing.T) {
	t.Parallel()

	for _, variant := range []string{"token", "client credentials", "service account", "client ID only", "client secret only"} {
		for _, location := range []string{"attribute", "environment"} {
			t.Run(variant+"/"+location, func(t *testing.T) {
				t.Parallel()
				// Given a profiles file pointing at a server that cannot be reached.
				trap := fakeapi.New(t)
				unreachable := httptest.NewServer(trap)
				unreachable.Close()
				cfg, _ := cliFixture(t, unreachable.URL, "trap", map[string]any{"identity": map[string]string{"refresh-token": "trap"}})
				s := fakeapi.New(t)
				s.AddClientCredential("acme:robot", "secret")
				s.AddFederatedIdentity("acme:robot", "subject")
				cfg.Server, cfg.Tenant, cfg.UseCLI = s.URL, "acme", cliBool(true)
				values := map[string]string{}

				switch variant {
				case "token":
					cfg.Token, values["WAYFINDER_TOKEN"] = "bearer", "bearer"
				case "client credentials":
					cfg.ClientID, cfg.ClientSecret = "acme:robot", "secret"
					values["WAYFINDER_CLIENT_ID"], values["WAYFINDER_CLIENT_SECRET"] = cfg.ClientID, cfg.ClientSecret
				case "service account":
					cfg.ServiceAccount, cfg.FederatedTokenFile = "acme:robot", subjectFile(t, "subject")
					values["WAYFINDER_SERVICE_ACCOUNT"], values["WAYFINDER_FEDERATED_TOKEN_FILE"] = cfg.ServiceAccount, cfg.FederatedTokenFile
				case "client ID only":
					cfg.ClientID, values["WAYFINDER_CLIENT_ID"] = "acme:robot", "acme:robot"
				case "client secret only":
					cfg.ClientSecret, values["WAYFINDER_CLIENT_SECRET"] = "secret", "secret"
				}

				if location == "environment" {
					cfg.Token, cfg.ClientID, cfg.ClientSecret, cfg.ServiceAccount, cfg.FederatedTokenFile = "", "", "", "", ""
					cliEnv(&cfg, values)
				}

				previous := cfg.Getenv
				cfg.Getenv = func(k string) string {
					if k == "WAYFINDER_CONFIG" {
						t.Error("explicit credentials consulted the profiles path")
					}

					return previous(k)
				}
				cfg.CLIHTTP = func([]byte) (*http.Client, error) {
					t.Error("explicit credentials consulted the CLI CA")
					return nil, errors.New("unexpected CLI call")
				}
				// When
				r, err := auth.Resolve(cfg)
				// Then
				if strings.Contains(variant, "only") {
					if err == nil || strings.Contains(err.Error(), "wf CLI") || !strings.Contains(err.Error(), "WAYFINDER_CLIENT_SECRET") {
						t.Fatalf("partial credentials: %v", err)
					}
				} else {
					if err != nil {
						t.Fatal(err)
					}

					token, err := r.TokenSource.Token(context.Background())
					if err != nil || token == "" || r.HTTP == nil {
						t.Fatalf("explicit authentication: %v", err)
					}
				}

				if len(trap.Requests()) != 0 {
					t.Fatal("explicit credentials reached the profile server")
				}
			})
		}
	}
}

func TestCLIAuth_whenUseCLIResolved(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, env, want string
		attr            *bool
	}{
		{"implicit missing", "", "or sign in with the wf CLI", nil},
		{"explicit missing", "", "profiles file", cliBool(true)},
		{"env true", "true", "profiles file", nil},
		{"env false", "false", "WAYFINDER_TOKEN", nil},
		{"disabled", "", "WAYFINDER_TOKEN", cliBool(false)},
		{"false overrides true", "true", "WAYFINDER_TOKEN", cliBool(false)},
		{"true overrides false", "false", "profiles file", cliBool(true)},
		{"invalid environment", "maybe", "WAYFINDER_USE_CLI must be true or false", nil},
		{"attribute overrides invalid", "maybe", "WAYFINDER_TOKEN", cliBool(false)},
		{"ParseBool uppercase", "TRUE", "profiles file", nil},
		{"ParseBool numeric", "0", "WAYFINDER_TOKEN", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given
			values := map[string]string{"WAYFINDER_CONFIG": filepath.Join(t.TempDir(), "absent"), "WAYFINDER_USE_CLI": tc.env}

			cfg := auth.Config{UseCLI: tc.attr, Getenv: func(k string) string { return values[k] }}
			if tc.want == "WAYFINDER_TOKEN" || tc.env == "maybe" {
				cfg.Getenv = func(k string) string {
					if k == "WAYFINDER_CONFIG" {
						t.Error("disabled or invalid use_cli read the profiles path")
					}

					return values[k]
				}
			}
			// When
			_, err := auth.Resolve(cfg)
			// Then
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("use_cli error = %v, want %q", err, tc.want)
			}

			if tc.want == "WAYFINDER_TOKEN" && strings.Contains(err.Error(), "wf CLI") {
				t.Fatalf("disabled CLI changed the original credentials error: %v", err)
			}
		})
	}
}

func TestCLIAuth_whenProfileConfigurationFails(t *testing.T) {
	t.Parallel()

	for _, variant := range []string{"server mismatch", "environment server mismatch", "missing tenant", "federated tenant mismatch", "unknown profile", "broken profiles"} {
		t.Run(variant, func(t *testing.T) {
			t.Parallel()
			// Given
			s := fakeapi.New(t)
			cfg, path := cliFixture(t, s.URL, "acme", map[string]any{"identity": map[string]string{"refresh-token": "secret-sentinel"}})
			want := "does not match wf CLI profile"

			switch variant {
			case "server mismatch":
				cfg.Server = "https://different.example"
			case "environment server mismatch":
				cliEnv(&cfg, map[string]string{"WAYFINDER_SERVER": "https://different.example"})
			case "missing tenant":
				cliEditFixture(t, path, func(doc map[string]any) { doc["profiles"].(map[string]any)["test"].(map[string]any)["tenant"] = "" })

				want = "Set tenant in the provider block or WAYFINDER_TENANT."
			case "federated tenant mismatch":
				cliEditFixture(t, path, func(doc map[string]any) {
					doc["users"] = map[string]any{"alice": map[string]any{"identity": map[string]any{"federated-identity": map[string]any{
						"service-account": map[string]string{"tenant": "other", "name": "robot"},
					}}}}
				})

				want = "they must agree"
			case "unknown profile":
				cfg.CLIProfile, want = "absent", "(available: test)"
			case "broken profiles":
				if err := os.WriteFile(path, []byte("users: [secret-sentinel"), 0o600); err != nil {
					t.Fatal(err)
				}

				want = "parse wf CLI profiles file"
			}
			// When
			_, err := auth.Resolve(cfg)
			// Then
			if err == nil || !strings.Contains(err.Error(), want) || strings.Contains(err.Error(), "secret-sentinel") || len(s.Requests()) != 0 {
				t.Fatalf("unsafe configuration failure: %v; requests=%d", err, len(s.Requests()))
			}
		})
	}
}

func TestCLIAuth_whenIdentityShapeSelected(t *testing.T) {
	t.Parallel()

	for _, variant := range []string{"client credentials", "federated file", "federated URL", "static identity", "legacy static", "refresh-scoped static"} {
		t.Run(variant, func(t *testing.T) {
			t.Parallel()
			// Given
			s := fakeapi.New(t)
			s.AddClientCredential("acme:robot", "secret")
			s.AddFederatedIdentity("acme:robot", "subject")

			identity := map[string]any{"token": "static"}
			user := map[string]any{"identity": identity}
			tenant, grant := "acme", ""

			switch variant {
			case "client credentials":
				tenant, grant = "", fakeapi.GrantClientCredentials
				identity["client-id"], identity["client-secret"] = "acme:robot", "secret"
				identity["federated-identity"] = map[string]any{}
			case "federated file", "federated URL":
				tenant, grant = "", fakeapi.GrantTokenExchange

				federated := map[string]any{"service-account": map[string]string{"tenant": "acme", "name": "robot"}}
				if variant == "federated file" {
					federated["subject-token-file"] = subjectFile(t, "subject")
				} else {
					oidc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
						if req.URL.Query().Get("audience") != "audience" || req.Header.Get("Authorization") != "Bearer oidc-secret" {
							t.Error("profile federated configuration was lost")
						}

						if _, err := io.WriteString(w, `{"value":"subject"}`); err != nil {
							t.Error(err)
						}
					}))
					t.Cleanup(oidc.Close)
					federated["token-server"], federated["token-server-token"], federated["audience"] = oidc.URL, "oidc-secret", "audience"
				}

				identity["federated-identity"] = federated
			case "legacy static":
				user = map[string]any{"token": "static"}
			case "refresh-scoped static":
				refresh := refreshScopedToken(t)
				s.AddRefreshToken(refresh)
				identity["token"], grant = refresh, fakeapi.GrantRefreshToken
			}

			cfg, _ := cliFixture(t, s.URL, tenant, user)
			// When
			r, err := auth.Resolve(cfg)
			// Then
			if err != nil {
				t.Fatal(err)
			}

			token, err := r.TokenSource.Token(context.Background())
			if err != nil || token == "" || r.Tenant != "acme" {
				t.Fatalf("profile identity: %v", err)
			}

			if grant == "" {
				if token != "static" || len(s.Requests()) != 0 {
					t.Fatal("static profile token was exchanged")
				}
			} else if s.Mints(grant) != 1 {
				t.Fatalf("profile identity selected wrong grant: %q", grant)
			}
		})
	}
}

func TestCLIAuth_whenExplicitServerMatchesAndTenantOverrides(t *testing.T) {
	t.Parallel()

	for _, location := range []string{"attribute", "environment"} {
		t.Run(location, func(t *testing.T) {
			t.Parallel()
			// Given
			s := fakeapi.New(t)
			s.AddRefreshToken("refresh")

			cfg, _ := cliFixture(t, s.URL+"/", "profile-tenant", map[string]any{"identity": map[string]string{"refresh-token": "refresh"}})
			if location == "attribute" {
				cfg.Server, cfg.Tenant = s.URL, "override"
			} else {
				cliEnv(&cfg, map[string]string{"WAYFINDER_SERVER": s.URL, "WAYFINDER_TENANT": "override"})
			}
			// When
			r, err := auth.Resolve(cfg)
			// Then
			if err != nil || r.Tenant != "override" || r.Server != s.URL+"/" || s.Mints(fakeapi.GrantRefreshToken) != 1 {
				t.Fatalf("profile overrides: %v", err)
			}
		})
	}
}

func TestCLIAuth_whenSessionRejected(t *testing.T) {
	t.Parallel()

	for _, variant := range []string{"rejected refresh", "secret response body", "unreachable server"} {
		t.Run(variant, func(t *testing.T) {
			t.Parallel()
			// Given
			const secret = "secret-sentinel-never-display"

			s := fakeapi.New(t)
			server := s.URL

			if variant == "secret response body" {
				endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)

					if _, err := io.WriteString(w, secret); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(endpoint.Close)
				server = endpoint.URL
			}

			if variant == "unreachable server" {
				unreachable := httptest.NewServer(s)
				unreachable.Close()
				server = unreachable.URL
			}

			cfg, _ := cliFixture(t, server, "acme", map[string]any{"identity": map[string]string{"refresh-token": secret}})
			// When
			_, err := auth.Resolve(cfg)
			// Then
			if err == nil || !strings.Contains(err.Error(), `could not authenticate with wf CLI profile "test"`) ||
				!strings.Contains(err.Error(), `wf login --profile test`) || strings.Contains(err.Error(), secret) {
				t.Fatalf("unsafe session error: %v", err)
			}

			if variant != "unreachable server" {
				var status *auth.HTTPStatusError
				if !errors.As(err, &status) || status.StatusCode != http.StatusUnauthorized {
					t.Fatalf("lost authentication status: %v", err)
				}
			}
		})
	}
}

func TestCLIAuth_whenConfigureContextCancelled(t *testing.T) {
	t.Parallel()
	// Given: the endpoint stays blocked until its request context is cancelled.
	started, stopped := make(chan struct{}), make(chan struct{})
	endpoint := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		if _, err := io.Copy(io.Discard, req.Body); err != nil {
			t.Error(err)
			return
		}

		close(started)
		<-req.Context().Done()
		close(stopped)
	}))
	t.Cleanup(endpoint.Close)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bound, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()

	cfg, _ := cliFixture(t, endpoint.URL, "acme", map[string]any{"identity": map[string]string{"refresh-token": "refresh"}})
	cfg.Context = ctx
	done := make(chan error, 1)

	go func() {
		_, err := auth.Resolve(cfg)
		done <- err
	}()

	select {
	case <-started:
	case err := <-done:
		t.Fatalf("Resolve did not reach eager mint: %v", err)
	case <-bound.Done():
		t.Fatal("eager mint did not start")
	}
	// When
	cancel()
	// Then
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation was lost: %v", err)
		}
	case <-bound.Done():
		t.Fatal("Resolve did not observe cancellation")
	}

	select {
	case <-stopped:
	case <-bound.Done():
		t.Fatal("token request did not observe cancellation")
	}
}

func TestCLIAuth_whenProfileCAHookUsed(t *testing.T) {
	t.Parallel()
	// Given a token endpoint trusted only by the profile's CA.
	s := fakeapi.New(t)
	s.AddRefreshToken("refresh")
	endpoint := httptest.NewTLSServer(s)
	t.Cleanup(endpoint.Close)
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: endpoint.Certificate().Raw})
	cfg, path := cliFixture(t, endpoint.URL, "acme", map[string]any{"identity": map[string]string{"refresh-token": "refresh"}})
	cliEditFixture(t, path, func(doc map[string]any) {
		doc["servers"].(map[string]any)["api"].(map[string]any)["caCertificate"] = string(certificate)
	})

	var hooked *http.Client

	calls := 0
	cfg.CLIHTTP = func(ca []byte) (*http.Client, error) {
		calls++

		if !bytes.Equal(ca, certificate) {
			t.Fatal("CA hook did not receive the profile PEM")
		}

		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(ca) {
			t.Fatal("fixture certificate was invalid")
		}

		transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}
		t.Cleanup(transport.CloseIdleConnections)
		hooked = &http.Client{Transport: transport}

		return hooked, nil
	}
	// When
	r, err := auth.Resolve(cfg)
	// Then
	if err != nil || calls != 1 || r.HTTP != hooked || s.Mints(fakeapi.GrantRefreshToken) != 1 {
		t.Fatalf("CA hook: calls=%d err=%v", calls, err)
	}
}

func TestCLIAuth_whenCAHookAbsentOrFails(t *testing.T) {
	t.Parallel()

	for _, variant := range []string{"absent", "fails"} {
		t.Run(variant, func(t *testing.T) {
			t.Parallel()
			// Given
			s := fakeapi.New(t)
			s.AddRefreshToken("refresh")
			cfg, path := cliFixture(t, s.URL, "acme", map[string]any{"identity": map[string]string{"refresh-token": "refresh"}})
			cliEditFixture(t, path, func(doc map[string]any) {
				doc["servers"].(map[string]any)["api"].(map[string]any)["caCertificate"] = "profile-pem"
			})

			cfg.HTTP = &http.Client{}
			cause := errors.New("invalid certificate")

			if variant == "fails" {
				cfg.CLIHTTP = func([]byte) (*http.Client, error) { return nil, cause }
			}
			// When
			r, err := auth.Resolve(cfg)
			// Then
			if variant == "fails" {
				if !errors.Is(err, cause) || !strings.Contains(err.Error(), "could not trust the wf CLI profile's CA certificate") || len(s.Requests()) != 0 {
					t.Fatalf("CA error = %v", err)
				}
			} else if err != nil || r.HTTP != cfg.HTTP || s.Mints(fakeapi.GrantRefreshToken) != 1 {
				t.Fatalf("nil CA hook = %v", err)
			}
		})
	}
}

func TestCLIAuth_whenBareTokenExpires(t *testing.T) {
	t.Parallel()

	for _, variant := range []string{"expired at configure", "expires during run", "explicit bearer unchanged"} {
		t.Run(variant, func(t *testing.T) {
			t.Parallel()
			// Given
			now := time.Unix(100, 0)
			token := "eyJhbGciOiJub25lIn0." + base64.RawURLEncoding.EncodeToString([]byte(`{"exp":200}`)) + ".signature"
			cfg, _ := cliFixture(t, "https://api.example", "acme", map[string]any{"identity": map[string]string{"token": token}})
			cfg.Now = func() time.Time { return now }

			if variant == "expired at configure" {
				now = time.Unix(200, 0)
			}

			if variant == "explicit bearer unchanged" {
				cfg.Token, cfg.Tenant = token, "acme"
			}
			// When
			r, err := auth.Resolve(cfg)
			if variant != "expired at configure" {
				if err != nil {
					t.Fatal(err)
				}

				now = time.Unix(200, 0)
				_, err = r.TokenSource.Token(context.Background())
			}
			// Then
			if err == nil || strings.Contains(err.Error(), token) {
				t.Fatalf("missing or unsafe expiry error: %v", err)
			}

			if variant == "explicit bearer unchanged" {
				if strings.Contains(err.Error(), "wf CLI") || !strings.Contains(err.Error(), "the bearer token has expired") {
					t.Fatalf("explicit bearer expiry changed: %v", err)
				}
			} else if !strings.Contains(err.Error(), `the access token stored in wf CLI profile "test" has expired and cannot be renewed`) ||
				!strings.Contains(err.Error(), `wf login --profile test`) {
				t.Fatalf("profile-specific expiry error lost: %v", err)
			}
		})
	}
}

func TestCLIAuth_whenExpiredBareTokenAtConfigure(t *testing.T) {
	t.Parallel()
	// Given
	token := "eyJhbGciOiJub25lIn0." + base64.RawURLEncoding.EncodeToString([]byte(`{"exp":200}`)) + ".signature"
	cfg, _ := cliFixture(t, "https://api.example", "acme", map[string]any{"identity": map[string]string{"token": token}})
	cfg.Now = func() time.Time { return time.Unix(200, 0) }
	// When
	_, err := auth.Resolve(cfg)
	// Then
	want := `the access token stored in wf CLI profile "test" has expired and cannot be renewed - sign in again with "wf login --profile test"`
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want exactly %q", err, want)
	}
}

func TestCLIAuth_whenEmptyFederatedIdentityIsOnlyCredential(t *testing.T) {
	t.Parallel()
	// Given
	s := fakeapi.New(t)
	cfg, _ := cliFixture(t, s.URL, "acme", map[string]any{"identity": map[string]any{"federated-identity": map[string]any{}}})
	// When
	_, err := auth.Resolve(cfg)
	// Then
	want := `wf CLI profile "test" has no stored credentials - run "wf login --profile test"`
	if err == nil || err.Error() != want || len(s.Requests()) != 0 {
		t.Fatalf("error = %v, want exactly %q; requests=%d", err, want, len(s.Requests()))
	}
}

func TestCLIAuth_whenFederatedProfileInvalid(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ name, tenant, serviceTenant, serviceName, want string }{
		{"invalid service account", "", "acme", "", `invalid service account stored in wf CLI profile "test": `},
		{"tenant mismatch", "other", "acme", "robot", `tenant "other" does not match the tenant of the service account stored in wf CLI profile "test" ("acme")`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given
			s := fakeapi.New(t)
			federated := map[string]any{"service-account": map[string]string{"tenant": tc.serviceTenant, "name": tc.serviceName}}
			cfg, _ := cliFixture(t, s.URL, tc.tenant, map[string]any{"identity": map[string]any{"federated-identity": federated}})
			// When
			_, err := auth.Resolve(cfg)
			// Then
			if err == nil || !strings.Contains(err.Error(), tc.want) || strings.Contains(err.Error(), "WAYFINDER_SERVICE_ACCOUNT") ||
				strings.Contains(err.Error(), "service_account /") || len(s.Requests()) != 0 {
				t.Fatalf("error = %v, want to contain %q; requests=%d", err, tc.want, len(s.Requests()))
			}
		})
	}
}
