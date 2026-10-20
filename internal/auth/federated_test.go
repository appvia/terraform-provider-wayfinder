// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/auth"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

func subjectFile(t *testing.T, value string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "subject")
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestToken_when_federated_file_overrides_url(t *testing.T) {
	// Given
	s := fakeapi.New(t)
	s.RecordBodies(true)
	s.AddFederatedIdentity("acme:robot", "subject")
	env := map[string]string{"WAYFINDER_SERVER": s.URL, "WAYFINDER_SERVICE_ACCOUNT": "acme:robot", "WAYFINDER_FEDERATED_TOKEN_FILE": subjectFile(t, "  subject\n"), "WAYFINDER_FEDERATED_TOKEN_REQUEST_URL": "http://invalid.invalid"}

	r, err := auth.Resolve(auth.Config{Getenv: func(k string) string { return env[k] }})
	if err != nil {
		t.Fatal(err)
	}
	// When
	_, err = r.TokenSource.Token(context.Background())
	// Then
	if err != nil || r.Tenant != "acme" || s.Mints(fakeapi.GrantTokenExchange) != 1 {
		t.Fatalf("federation: %v", err)
	}

	var body map[string]string
	if err := json.Unmarshal(s.Requests()[0].Body, &body); err != nil {
		t.Fatal(err)
	}

	if len(body) != 4 || body["grant_type"] != fakeapi.GrantTokenExchange || body["subject_token"] != "subject" || body["subject_token_type"] != fakeapi.SubjectTokenTypeJWT || body["serviceaccount"] != "acme:robot" {
		t.Fatal("federated JSON differs from CLI")
	}
}

func TestToken_when_federated_url_or_actions_fallback(t *testing.T) {
	for _, variant := range []string{"provider attributes", "environment", "actions", "derived audience"} {
		t.Run(variant, func(t *testing.T) {
			// Given
			s := fakeapi.New(t)
			s.AddFederatedIdentity("acme:ci:robot", "subject")

			audience := "custom/audience"
			if variant == "derived audience" {
				audience = "https://portal.example/t/acme"
			}

			var oidcCalls atomic.Int32

			oidc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				oidcCalls.Add(1)

				if req.Method != http.MethodGet || req.URL.Query().Get("audience") != audience || req.URL.Query().Get("existing") != "1" || req.Header.Get("Authorization") != "Bearer oidc-request-token" {
					t.Error("OIDC request differs from CLI")
				}

				if _, err := w.Write([]byte(`{"value":"subject"}`)); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(oidc.Close)

			env := map[string]string{"ACTIONS_ID_TOKEN_REQUEST_URL": "http://invalid.invalid", "ACTIONS_ID_TOKEN_REQUEST_TOKEN": "bad"}
			cfg := auth.Config{Server: s.URL, ServiceAccount: "acme:ci:robot", FederatedTokenRequestAudience: audience, Getenv: func(k string) string { return env[k] }}

			switch variant {
			case "provider attributes", "derived audience":
				cfg.FederatedTokenRequestURL, cfg.FederatedTokenRequestToken = oidc.URL+"?existing=1", "oidc-request-token"
				env["WAYFINDER_FEDERATED_TOKEN_REQUEST_URL"] = "http://invalid.invalid"
				env["WAYFINDER_FEDERATED_TOKEN_REQUEST_TOKEN"] = "bad"
				env["WAYFINDER_FEDERATED_TOKEN_REQUEST_AUDIENCE"] = "bad"
			case "environment":
				cfg.FederatedTokenRequestAudience = ""
				env["WAYFINDER_FEDERATED_TOKEN_REQUEST_URL"], env["WAYFINDER_FEDERATED_TOKEN_REQUEST_TOKEN"], env["WAYFINDER_FEDERATED_TOKEN_REQUEST_AUDIENCE"] = oidc.URL+"?existing=1", "oidc-request-token", audience
			case "actions":
				env["ACTIONS_ID_TOKEN_REQUEST_URL"], env["ACTIONS_ID_TOKEN_REQUEST_TOKEN"] = oidc.URL+"?existing=1", "oidc-request-token"
			}

			if variant == "derived audience" {
				env = map[string]string{
					"WAYFINDER_SERVER": s.URL, "WAYFINDER_SERVICE_ACCOUNT": "acme:ci:robot",
					"ACTIONS_ID_TOKEN_REQUEST_URL": oidc.URL + "?existing=1", "ACTIONS_ID_TOKEN_REQUEST_TOKEN": "oidc-request-token",
				}
				cfg = auth.Config{Getenv: func(k string) string { return env[k] }}
			}

			r, err := auth.Resolve(cfg)
			if err != nil {
				t.Fatal(err)
			}
			// When
			_, err = r.TokenSource.Token(context.Background())
			// Then
			if err != nil || s.Mints(fakeapi.GrantTokenExchange) != 1 {
				t.Fatalf("URL federation: %v", err)
			}

			if oidcCalls.Load() != 1 || s.CountRequests("", "/api/v3/general/serverinfo") != 0 {
				t.Fatal("expected one OIDC request and no serverinfo request")
			}

			wantDiscovery := 0
			if variant == "derived audience" {
				wantDiscovery = 1
			}

			if got := s.CountRequests(http.MethodGet, "/apiinfo"); got != wantDiscovery {
				t.Fatalf("apiinfo requests = %d, want %d", got, wantDiscovery)
			}
		})
	}
}

func TestToken_when_federated_source_fails(t *testing.T) {
	for _, variant := range []string{"stdin", "missing file", "empty OIDC", "OIDC failure"} {
		t.Run(variant, func(t *testing.T) {
			// Given
			s := fakeapi.New(t)
			cfg := auth.Config{Server: s.URL, ServiceAccount: "acme:robot", Getenv: emptyEnv, FederatedTokenRequestAudience: "audience"}

			switch variant {
			case "stdin":
				cfg.FederatedTokenFile = "-"
			case "missing file":
				cfg.FederatedTokenFile = filepath.Join(t.TempDir(), "absent")
			default:
				oidc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if variant == "OIDC failure" {
						w.WriteHeader(http.StatusUnauthorized)
					}

					if _, err := w.Write([]byte(`{"value":"","secret":"do-not-echo"}`)); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(oidc.Close)
				cfg.FederatedTokenRequestURL = oidc.URL
			}

			r, err := auth.Resolve(cfg)
			if err != nil {
				t.Fatal(err)
			}
			// When
			_, err = r.TokenSource.Token(context.Background())
			// Then
			if err == nil || strings.Contains(err.Error(), "do-not-echo") {
				t.Fatalf("unsafe or missing error: %v", err)
			}
		})
	}
}

func TestToken_returns_audience_guidance_when_apiinfo_unavailable(t *testing.T) {
	causes := map[string]string{
		"missing UI":        "server API info has no ui URL",
		"non-2xx":           "HTTP 502 Bad Gateway",
		"malformed JSON":    "decode authentication response failed",
		"transport failure": "authentication HTTP request failed",
		"oversized":         "exceeds",
	}

	for _, variant := range []string{"missing UI", "non-2xx", "malformed JSON", "transport failure", "oversized"} {
		t.Run(variant, func(t *testing.T) {
			// Given an API-info endpoint that cannot supply the UI URL.
			var calls atomic.Int32

			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)

				if req.URL.Path != "/apiinfo" || req.Method != http.MethodGet || req.Header.Get("Authorization") != "" {
					t.Error("discovery must GET apiinfo without Authorization")
				}

				body := `{"generalAPI":"/api/v3/general","resourceAPI":"/api/v3/resources"}`

				switch variant {
				case "non-2xx":
					w.WriteHeader(http.StatusBadGateway)

					body = "do-not-echo"
				case "malformed JSON":
					body = "do-not-echo"
				case "oversized":
					body = `{"ui":"` + strings.Repeat("a", 2<<20) + `"}`
				}
				// The provider stops reading once the 1 MiB cap is hit, so writing the
				// oversized body can legitimately fail with EPIPE. That is not a test failure.
				if _, err := w.Write([]byte(body)); err != nil && variant != "oversized" {
					t.Error(err)
				}
			}))
			t.Cleanup(s.Close)

			if variant == "transport failure" {
				s.Close()
			}

			r, err := auth.Resolve(auth.Config{Server: s.URL, ServiceAccount: "acme:robot", FederatedTokenRequestURL: "http://invalid.invalid", Getenv: emptyEnv})
			if err != nil {
				t.Fatal(err)
			}
			// When
			_, err = r.TokenSource.Token(context.Background())
			// Then discovery failures give actionable guidance without URLs or response bodies.
			if err == nil || !strings.Contains(err.Error(), "set federated_token_request_audience / WAYFINDER_FEDERATED_TOKEN_REQUEST_AUDIENCE explicitly") || strings.Contains(err.Error(), s.URL) || strings.Contains(err.Error(), "do-not-echo") {
				t.Fatalf("unsafe or missing audience guidance: %v", err)
			}

			if !strings.Contains(err.Error(), causes[variant]) {
				t.Fatalf("missing cause %q: %v", causes[variant], err)
			}

			if variant == "non-2xx" {
				var status *auth.HTTPStatusError
				if !errors.As(err, &status) || status.StatusCode != http.StatusBadGateway {
					t.Fatalf("want *auth.HTTPStatusError 502, got: %v", err)
				}
			}

			if variant != "transport failure" && calls.Load() != 1 {
				t.Fatalf("discovery calls = %d, want 1", calls.Load())
			}
		})
	}
}
