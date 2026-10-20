// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/auth"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

func TestToken_when_environment_credentials_reused_and_expired(t *testing.T) {
	// Given
	t.Cleanup(func() { t.Log("teardown: fake HTTP server closed") })
	s := fakeapi.New(t)
	s.AddClientCredential("acme:robot:key", "secret")
	s.SetTokenTTL(5 * time.Minute)

	now := time.Now()
	env := map[string]string{"WAYFINDER_SERVER": s.URL, "WAYFINDER_CLIENT_ID": "acme:robot:key", "WAYFINDER_CLIENT_SECRET": "secret"}

	r, err := auth.Resolve(auth.Config{Getenv: func(k string) string { return env[k] }, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}

	first, err := r.TokenSource.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// When: reuse, then cross expiry without sleeps.
	reused, err := r.TokenSource.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if first != reused || s.Mints(fakeapi.GrantClientCredentials) != 1 {
		t.Fatal("token was not cached")
	}

	s.ExpireMintedTokens()

	now = now.Add(6 * time.Minute)
	refreshed, err := r.TokenSource.Token(context.Background())
	// Then
	if err != nil || refreshed == first || s.Mints(fakeapi.GrantClientCredentials) != 2 {
		t.Fatalf("refresh failed: %v", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, s.URL+"/api/v3/resources/tenants/acme/org/v3/workspaces/missing", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+refreshed)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("minted token rejected: HTTP %d", resp.StatusCode)
	}

	for _, req := range s.Requests() {
		t.Logf("HTTP %s %s -> %d", req.Method, req.Path, req.Status)
	}

	t.Log("environment client credentials: minted once, reused, refreshed after expiry; fake server teardown registered with t.Cleanup")
}

func TestToken_when_concurrent_callers(t *testing.T) {
	// Given
	s := fakeapi.New(t)
	s.AddClientCredential("acme:robot", "secret")

	r, err := auth.Resolve(auth.Config{Server: s.URL, ClientID: "acme:robot", ClientSecret: "secret", Getenv: emptyEnv})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup

	start := make(chan struct{})
	// When
	for range 32 {
		wg.Go(func() {
			<-start

			token, err := r.TokenSource.Token(context.Background())
			if err != nil || token == "" {
				t.Errorf("Token: %v", err)
			}
		})
	}

	close(start)
	wg.Wait()
	// Then
	if s.Mints(fakeapi.GrantClientCredentials) != 1 {
		t.Fatal("concurrent callers minted more than once")
	}
}

func TestToken_when_sixty_seconds_before_expiry(t *testing.T) {
	// Given
	s := fakeapi.New(t)
	s.AddClientCredential("acme:robot", "secret")
	s.SetTokenTTL(5 * time.Minute)

	now := time.Now().Truncate(time.Second)

	r, err := auth.Resolve(auth.Config{Server: s.URL, ClientID: "acme:robot", ClientSecret: "secret", Getenv: emptyEnv, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}

	first, err := r.TokenSource.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	now = now.Add(239 * time.Second)

	prior, err := r.TokenSource.Token(context.Background())
	if err != nil || prior != first {
		t.Fatal("refreshed too early")
	}
	// When
	now = now.Add(time.Second)
	got, err := r.TokenSource.Token(context.Background())
	// Then
	if err != nil || got == first || s.Mints(fakeapi.GrantClientCredentials) != 2 {
		t.Fatalf("refresh boundary: %v", err)
	}
}

func TestToken_when_wrong_secret(t *testing.T) {
	// Given
	s := fakeapi.New(t)
	s.AddClientCredential("acme:robot", "right")

	const secret = "wrong-secret-never-display"

	r, err := auth.Resolve(auth.Config{Server: s.URL, ClientID: "acme:robot", ClientSecret: secret, Getenv: emptyEnv})
	if err != nil {
		t.Fatal(err)
	}
	// When
	_, err = r.TokenSource.Token(context.Background())
	// Then
	if err == nil || !strings.Contains(err.Error(), "authentication failed") || !strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), secret) {
		t.Fatalf("unsafe error: %v", err)
	}

	t.Logf("wrong secret: %s (secret absent)", err)
}

func exchangeToken(t *testing.T) string {
	t.Helper()

	payload, err := json.Marshal(struct {
		Scopes []string `json:"scopes"`
	}{[]string{"wayfinder:auth:exchange"}})
	if err != nil {
		t.Fatal(err)
	}

	return "eyJhbGciOiJub25lIn0." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func TestToken_when_exchange_scoped_token(t *testing.T) {
	// Given
	s := fakeapi.New(t)
	s.RecordBodies(true)

	refresh := exchangeToken(t)
	s.AddRefreshToken(refresh)

	r, err := auth.Resolve(auth.Config{Server: s.URL, Tenant: "acme", Token: refresh, Getenv: emptyEnv})
	if err != nil {
		t.Fatal(err)
	}
	// When
	got, err := r.TokenSource.Token(context.Background())
	// Then
	if err != nil || got == refresh || s.Mints(fakeapi.GrantRefreshToken) != 1 {
		t.Fatalf("exchange: %v", err)
	}

	var body map[string]string
	if err := json.Unmarshal(s.Requests()[0].Body, &body); err != nil {
		t.Fatal(err)
	}

	if len(body) != 2 || body["refresh_token"] != refresh || body["grant_type"] != fakeapi.GrantRefreshToken {
		t.Fatal("refresh JSON does not match CLI")
	}
}

func refreshScopedToken(t *testing.T) string {
	t.Helper()

	payload, err := json.Marshal(struct {
		Scopes []string `json:"scopes"`
	}{[]string{"wayfinder:auth:refresh"}})
	if err != nil {
		t.Fatal(err)
	}

	return "eyJhbGciOiJub25lIn0." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func TestToken_whenRefreshScopedTokenExpires(t *testing.T) {
	// Given
	s := fakeapi.New(t)
	refresh := refreshScopedToken(t)
	s.AddRefreshToken(refresh)
	s.SetTokenTTL(5 * time.Minute)

	now := time.Now()

	r, err := auth.Resolve(auth.Config{Server: s.URL, Tenant: "acme", Token: refresh, Getenv: emptyEnv, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}

	first, err := r.TokenSource.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// When: cross expiry without sleeps.
	s.ExpireMintedTokens()

	now = now.Add(6 * time.Minute)
	second, err := r.TokenSource.Token(context.Background())
	// Then
	if err != nil || first == refresh || second == refresh || second == first || s.Mints(fakeapi.GrantRefreshToken) != 2 {
		t.Fatalf("refresh-scoped token not re-minted: err=%v mints=%d", err, s.Mints(fakeapi.GrantRefreshToken))
	}
}

func TestToken_whenAPIScopedToken(t *testing.T) {
	// Given
	s := fakeapi.New(t)

	payload, err := json.Marshal(struct {
		Scopes []string `json:"scopes"`
	}{[]string{"wayfinder:api"}})
	if err != nil {
		t.Fatal(err)
	}

	token := "eyJhbGciOiJub25lIn0." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"

	r, err := auth.Resolve(auth.Config{Server: s.URL, Tenant: "acme", Token: token, Getenv: emptyEnv})
	if err != nil {
		t.Fatal(err)
	}
	// When
	got, err := r.TokenSource.Token(context.Background())
	// Then
	if err != nil || got != token || s.Mints(fakeapi.GrantRefreshToken) != 0 {
		t.Fatalf("api-scoped token should be a plain bearer: err=%v", err)
	}
}

func TestToken_when_client_credentials_posts_cli_json(t *testing.T) {
	// Given
	s := fakeapi.New(t)
	s.RecordBodies(true)
	s.AddClientCredential("acme:robot", "secret")

	r, err := auth.Resolve(auth.Config{Server: s.URL + "/", ClientID: "acme:robot", ClientSecret: "secret", Getenv: emptyEnv})
	if err != nil {
		t.Fatal(err)
	}
	// When
	_, err = r.TokenSource.Token(context.Background())
	// Then
	if err != nil {
		t.Fatal(err)
	}

	req := s.Requests()[0]

	var body map[string]string
	if err := json.Unmarshal(req.Body, &body); err != nil {
		t.Fatal(err)
	}

	if req.Method != http.MethodPost || req.Path != "/api/v3/general/login/token" || len(body) != 3 || body["grant_type"] != fakeapi.GrantClientCredentials || body["client_id"] != "acme:robot" || body["client_secret"] != "secret" {
		t.Fatal("client credential wire contract differs")
	}
}
