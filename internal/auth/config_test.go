// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"context"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/auth"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

func emptyEnv(string) string { return "" }

func TestResolve_when_credentials_compete(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cfg   auth.Config
		grant string
	}{
		{"token beats federation and credentials", auth.Config{Token: "bearer", Tenant: "acme", ServiceAccount: "invalid", ClientID: "other:robot", ClientSecret: "wrong"}, ""},
		{"federation beats credentials", auth.Config{ServiceAccount: "acme:ci:robot", ClientID: "other:robot", ClientSecret: "wrong"}, fakeapi.GrantTokenExchange},
		{"client credentials", auth.Config{ClientID: "acme:robot:key", ClientSecret: "secret"}, fakeapi.GrantClientCredentials},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			s := fakeapi.New(t)
			s.AddClientCredential("acme:robot:key", "secret")
			s.AddFederatedIdentity("acme:ci:robot", "subject")

			cfg := tc.cfg
			cfg.Server, cfg.Getenv = s.URL, emptyEnv
			cfg.FederatedTokenFile = subjectFile(t, "subject")

			resolved, err := auth.Resolve(cfg)
			if err != nil {
				t.Fatal(err)
			}
			// When
			token, err := resolved.TokenSource.Token(context.Background())
			// Then
			if err != nil || token == "" {
				t.Fatalf("Token failed: %v", err)
			}

			if resolved.Tenant != "acme" {
				t.Errorf("tenant = %q", resolved.Tenant)
			}

			if tc.grant == "" {
				if token != "bearer" || len(s.Requests()) != 0 {
					t.Fatal("bearer was exchanged")
				}
			} else if s.Mints(tc.grant) != 1 {
				t.Fatal("wrong grant selected")
			}
		})
	}
}

func TestResolve_when_tenant_or_credentials_missing(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  auth.Config
		want string
	}{
		{"missing credentials", auth.Config{UseCLI: cliBool(false)}, "WAYFINDER_TOKEN"},
		{"partial credentials", auth.Config{ClientID: "acme:robot"}, "WAYFINDER_CLIENT_SECRET"},
		{"missing tenant", auth.Config{Token: "bearer"}, "Set tenant in the provider block or WAYFINDER_TENANT."},
		{"mismatched tenant", auth.Config{Tenant: "other", ServiceAccount: "acme:robot"}, "they must agree"},
		{"missing federated source", auth.Config{ServiceAccount: "acme:robot"}, "no identity token source"},
		{"unqualified reference", auth.Config{ServiceAccount: "robot"}, "must be tenant-qualified"},
		{"slash reference", auth.Config{ServiceAccount: "acme/robot"}, "not '/'"},
		{"empty reference segment", auth.Config{ServiceAccount: "acme::robot"}, "segments must not be empty"},
		{"environment reference", auth.Config{ServiceAccount: "acme:ci:dev:robot"}, "environment-scoped"},
		{"too many reference segments", auth.Config{ServiceAccount: "a:b:c:d:e"}, "expected tenant:name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			cfg := tc.cfg
			cfg.Getenv = emptyEnv
			// When
			_, err := auth.Resolve(cfg)
			// Then
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}

			if tc.name == "missing credentials" {
				for _, name := range []string{"WAYFINDER_SERVICE_ACCOUNT", "WAYFINDER_CLIENT_ID", "WAYFINDER_CLIENT_SECRET"} {
					if !strings.Contains(err.Error(), name) {
						t.Errorf("missing diagnostic %s", name)
					}
				}

				t.Logf("no credentials: %s", err)
			}
		})
	}
}

func TestResolve_when_attributes_override_environment(t *testing.T) {
	// Given
	env := map[string]string{"WAYFINDER_SERVER": "https://env.example", "WAYFINDER_TENANT": "env", "WAYFINDER_TOKEN": "env-token", "WAYFINDER_WORKSPACE": "ignored", "WAYFINDER_ENVIRONMENT": "ignored"}
	cfg := auth.Config{Server: "https://attr.example", Tenant: "attr", Token: "attr-token", Getenv: func(k string) string { return env[k] }}
	// When
	r, err := auth.Resolve(cfg)
	// Then
	if err != nil {
		t.Fatal(err)
	}

	token, err := r.TokenSource.Token(context.Background())
	if err != nil || token != "attr-token" || r.Server != cfg.Server || r.Tenant != cfg.Tenant {
		t.Fatal("attributes did not win")
	}
}

func TestResolve_when_default_server_and_environment_token(t *testing.T) {
	// Given
	t.Setenv("WAYFINDER_TOKEN", "env-token")
	t.Setenv("WAYFINDER_TENANT", "acme")
	t.Setenv("WAYFINDER_SERVER", "")
	// When
	r, err := auth.Resolve(auth.Config{})
	// Then
	if err != nil {
		t.Fatal(err)
	}

	token, err := r.TokenSource.Token(context.Background())
	if err != nil || token != "env-token" || r.Server != "https://api.on.wayfinder.run" || r.Tenant != "acme" {
		t.Fatal("default environment resolution failed")
	}
}

func TestResolve_when_credential_attributes_override_environment(t *testing.T) {
	for _, variant := range []string{"client credentials", "federated file"} {
		t.Run(variant, func(t *testing.T) {
			// Given
			s := fakeapi.New(t)
			s.AddClientCredential("acme:robot", "attr-secret")
			s.AddFederatedIdentity("acme:robot", "attr-subject")

			env := map[string]string{"WAYFINDER_CLIENT_ID": "wrong:robot", "WAYFINDER_CLIENT_SECRET": "wrong-secret"}
			cfg := auth.Config{Server: s.URL, ClientID: "acme:robot", ClientSecret: "attr-secret", Getenv: func(k string) string { return env[k] }}
			grant := fakeapi.GrantClientCredentials

			if variant == "federated file" {
				env["WAYFINDER_SERVICE_ACCOUNT"] = "wrong:robot"
				env["WAYFINDER_FEDERATED_TOKEN_FILE"] = subjectFile(t, "wrong-subject")
				cfg.ServiceAccount, cfg.FederatedTokenFile = "acme:robot", subjectFile(t, "attr-subject")
				grant = fakeapi.GrantTokenExchange
			}

			r, err := auth.Resolve(cfg)
			if err != nil {
				t.Fatal(err)
			}
			// When
			_, err = r.TokenSource.Token(context.Background())
			// Then
			if err != nil || r.Tenant != "acme" || s.Mints(grant) != 1 {
				t.Fatalf("credential override: %v", err)
			}
		})
	}
}

func TestResolve_when_actions_fallback_is_incomplete(t *testing.T) {
	for _, key := range []string{"ACTIONS_ID_TOKEN_REQUEST_URL", "ACTIONS_ID_TOKEN_REQUEST_TOKEN"} {
		t.Run(key, func(t *testing.T) {
			// Given
			cfg := auth.Config{ServiceAccount: "acme:robot", Getenv: func(k string) string {
				if k == key {
					return "present"
				}

				return ""
			}}
			// When
			_, err := auth.Resolve(cfg)
			// Then
			if err == nil || !strings.Contains(err.Error(), "no identity token source") {
				t.Fatalf("partial Actions fallback: %v", err)
			}
		})
	}
}
