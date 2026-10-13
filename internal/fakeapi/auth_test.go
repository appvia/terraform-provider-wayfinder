package fakeapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

const tokenPath = "/api/v3/general/login/token"

func TestAuth_returns_401_for_missing_or_unknown_bearer_token(t *testing.T) {
	s := newServer(t, identityKind())

	missing := callWithToken(t, s, "", http.MethodGet, identityCollection+"/x", nil)
	wrong := callWithToken(t, s, "nope", http.MethodGet, identityCollection+"/x", nil)

	wantCode(t, missing, http.StatusUnauthorized)
	wantCode(t, wrong, http.StatusUnauthorized)
}

func TestToken_client_credentials_mints_usable_token_and_wrong_secret_is_401(t *testing.T) {
	s := newServer(t, identityKind())
	s.AddClientCredential("acme:robot", "right")
	s.SetTokenTTL(90 * time.Second)

	ok := callWithToken(t, s, "", http.MethodPost, tokenPath, map[string]any{"grant_type": "client_credentials", "client_id": "acme:robot", "client_secret": "right"})
	bad := callWithToken(t, s, "", http.MethodPost, tokenPath, map[string]any{"grant_type": "client_credentials", "client_id": "acme:robot", "client_secret": "wrong"})

	wantCode(t, ok, http.StatusOK)
	wantCode(t, bad, http.StatusUnauthorized)
	tok, _ := ok.body["access_token"].(string)
	if tok == "" || ok.body["expires_in"] != float64(90) || ok.body["token_type"] != "Bearer" {
		t.Fatalf("token response = %s", ok.raw)
	}
	wantCode(t, callWithToken(t, s, tok, http.MethodGet, identityCollection+"/x", nil), http.StatusNotFound)
	if n := s.Mints(fakeapi.GrantClientCredentials); n != 1 {
		t.Errorf("mints = %d, want 1", n)
	}
}

func TestToken_exchange_federated_identity(t *testing.T) {
	s := newServer(t)
	s.AddFederatedIdentity("acme:ws1:deployer", "oidc-jwt")
	body := func(subject, typ string) map[string]any {
		return map[string]any{
			"grant_type": fakeapi.GrantTokenExchange, "subject_token": subject,
			"subject_token_type": typ, "serviceaccount": "acme:ws1:deployer",
		}
	}

	ok := callWithToken(t, s, "", http.MethodPost, tokenPath, body("oidc-jwt", fakeapi.SubjectTokenTypeJWT))
	wrongSubject := callWithToken(t, s, "", http.MethodPost, tokenPath, body("other", fakeapi.SubjectTokenTypeJWT))
	wrongType := callWithToken(t, s, "", http.MethodPost, tokenPath, body("oidc-jwt", "urn:x"))

	wantCode(t, ok, http.StatusOK)
	wantCode(t, wrongSubject, http.StatusUnauthorized)
	wantCode(t, wrongType, http.StatusBadRequest)
	if s.Mints(fakeapi.GrantTokenExchange) != 1 {
		t.Errorf("mints = %d", s.Mints(fakeapi.GrantTokenExchange))
	}
}

func TestToken_refresh_grant(t *testing.T) {
	s := newServer(t)
	s.AddRefreshToken("refresh-1")

	ok := callWithToken(t, s, "", http.MethodPost, tokenPath, map[string]any{"grant_type": fakeapi.GrantRefreshToken, "refresh_token": "refresh-1"})
	empty := callWithToken(t, s, "", http.MethodPost, tokenPath, map[string]any{"grant_type": fakeapi.GrantRefreshToken})
	unknown := callWithToken(t, s, "", http.MethodPost, tokenPath, map[string]any{"grant_type": fakeapi.GrantRefreshToken, "refresh_token": "x"})
	badGrant := callWithToken(t, s, "", http.MethodPost, tokenPath, map[string]any{"grant_type": "password"})

	wantCode(t, ok, http.StatusOK)
	wantCode(t, empty, http.StatusBadRequest)
	wantCode(t, unknown, http.StatusUnauthorized)
	wantCode(t, badGrant, http.StatusBadRequest)
}

func TestToken_expired_minted_tokens_are_rejected(t *testing.T) {
	s := newServer(t, identityKind())
	s.AddClientCredential("acme:robot", "right")
	tok := callWithToken(t, s, "", http.MethodPost, tokenPath, map[string]any{"grant_type": "client_credentials", "client_id": "acme:robot", "client_secret": "right"}).body["access_token"].(string)

	s.ExpireMintedTokens()

	wantCode(t, callWithToken(t, s, tok, http.MethodGet, identityCollection+"/x", nil), http.StatusUnauthorized)
}

func TestServerInfo_returns_version_release(t *testing.T) {
	s := newServer(t)
	s.SetServerVersion("v3.3.2")

	got := callWithToken(t, s, "", http.MethodGet, "/api/v3/general/serverinfo", nil)

	wantCode(t, got, http.StatusOK)
	if dig(got.body, "version", "release") != "v3.3.2" {
		t.Errorf("body = %s", got.raw)
	}
}
