package auth_test

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/auth"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

func TestToken_when_bearer_expired(t *testing.T) {
	// Given
	s := fakeapi.New(t)
	token := "eyJhbGciOiJub25lIn0." + base64.RawURLEncoding.EncodeToString([]byte(`{"exp":1}`)) + ".signature"
	r, err := auth.Resolve(auth.Config{Server: s.URL, Tenant: "acme", Token: token, Getenv: emptyEnv})
	if err != nil {
		t.Fatal(err)
	}
	// When
	_, err = r.TokenSource.Token(context.Background())
	// Then
	if err == nil || !strings.Contains(err.Error(), "expired") || len(s.Requests()) != 0 {
		t.Fatalf("expired bearer: %v", err)
	}
}

func TestToken_when_context_cancelled(t *testing.T) {
	for _, variant := range []string{"bearer", "client"} {
		t.Run(variant, func(t *testing.T) {
			// Given
			s := fakeapi.New(t)
			cfg := auth.Config{Server: s.URL, Tenant: "acme", ClientID: "acme:robot", ClientSecret: "secret", Getenv: emptyEnv}
			if variant == "bearer" {
				cfg.Token = "bearer"
			}
			r, err := auth.Resolve(cfg)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			// When
			_, err = r.TokenSource.Token(ctx)
			// Then
			if !errors.Is(err, context.Canceled) || len(s.Requests()) != 0 {
				t.Fatalf("cancelled Token: %v", err)
			}
		})
	}
}

func TestToken_when_response_is_untrusted(t *testing.T) {
	for _, variant := range []string{"rejected", "malformed", "empty"} {
		t.Run(variant, func(t *testing.T) {
			// Given
			const secret = "never-echo-response-secret"
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				switch variant {
				case "rejected":
					w.WriteHeader(http.StatusUnauthorized)
					if _, err := w.Write([]byte(secret)); err != nil {
						t.Error(err)
					}
				case "malformed":
					if _, err := w.Write([]byte(`{"expires_in":"` + secret + `"}`)); err != nil {
						t.Error(err)
					}
				case "empty":
					if _, err := w.Write([]byte(`{}`)); err != nil {
						t.Error(err)
					}
				}
			}))
			t.Cleanup(s.Close)
			r, err := auth.Resolve(auth.Config{Server: s.URL, ClientID: "acme:robot", ClientSecret: secret, Getenv: emptyEnv, HTTP: s.Client()})
			if err != nil {
				t.Fatal(err)
			}
			// When
			_, err = r.TokenSource.Token(context.Background())
			// Then
			if err == nil || strings.Contains(err.Error(), secret) {
				t.Fatalf("unsafe response error: %v", err)
			}
			if variant == "rejected" {
				var status *auth.HTTPStatusError
				if !errors.As(err, &status) || status.StatusCode != http.StatusUnauthorized {
					t.Fatal("HTTP error lost status")
				}
			}
		})
	}
}

func TestToken_when_refresh_response_rotates_token(t *testing.T) {
	// Given
	s := fakeapi.New(t)
	original := exchangeToken(t)
	s.AddRefreshToken(original)
	s.AddRefreshToken("rotated")
	now := time.Now()
	var calls atomic.Int64
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if calls.Add(1) == 1 {
			if _, err := w.Write([]byte(`{"access_token":"first","expires_in":120,"refresh_token":"rotated"}`)); err != nil {
				t.Error(err)
			}
			return
		}
		s.ServeHTTP(w, req)
	}))
	t.Cleanup(proxy.Close)
	r, err := auth.Resolve(auth.Config{Server: proxy.URL, Tenant: "acme", Token: original, Getenv: emptyEnv, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.TokenSource.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	s.RecordBodies(true)
	// When
	_, err = r.TokenSource.Token(context.Background())
	// Then
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(s.Requests()[0].Body), `"refresh_token":"rotated"`) {
		t.Fatal("refresh token rotation lost")
	}
}

func TestToken_when_token_endpoint_rate_limits(t *testing.T) {
	// Given a token endpoint that rate limits the first two requests (as Wayfinder's anonymous
	// per-IP limiter does) and then issues a token.
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		if _, err := w.Write([]byte(`{"access_token":"issued","expires_in":3600}`)); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(s.Close)
	r, err := auth.Resolve(auth.Config{Server: s.URL, ClientID: "acme:robot", ClientSecret: "secret", Getenv: emptyEnv, HTTP: s.Client()})
	if err != nil {
		t.Fatal(err)
	}
	// When
	got, err := r.TokenSource.Token(context.Background())
	// Then the request is retried until the token is issued.
	if err != nil || got != "issued" || calls.Load() != 3 {
		t.Fatalf("token = %q, err = %v, calls = %d", got, err, calls.Load())
	}
}
