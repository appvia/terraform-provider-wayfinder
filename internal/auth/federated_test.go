package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
			oidc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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
				cfg.FederatedTokenRequestAudience = ""
				delete(env, "WAYFINDER_FEDERATED_TOKEN_REQUEST_AUDIENCE")
				proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if req.URL.Path == "/api/v3/general/serverinfo" {
						if _, err := w.Write([]byte(`{"ui":"https://portal.example/"}`)); err != nil {
							t.Error(err)
						}
						return
					}
					s.ServeHTTP(w, req)
				}))
				t.Cleanup(proxy.Close)
				cfg.Server = proxy.URL
				cfg.HTTP = proxy.Client()
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
		})
	}
}

func TestToken_when_federated_source_fails(t *testing.T) {
	for _, variant := range []string{"stdin", "missing file", "missing UI", "empty OIDC", "OIDC failure"} {
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
				if variant == "missing UI" {
					cfg.FederatedTokenRequestAudience = ""
				}
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
