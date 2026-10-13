package fakeapi

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Grant and token types of POST /api/v3/general/login/token (pkg/apiserver/types/types.go:86-97).
const (
	GrantRefreshToken      = "refresh_token"
	GrantTokenExchange     = "urn:ietf:params:oauth:grant-type:token-exchange"
	GrantClientCredentials = "client_credentials"
	SubjectTokenTypeJWT    = "urn:ietf:params:oauth:token-type:jwt"
)

type authState struct {
	static      map[string]bool      // tokens accepted forever (DefaultToken, AcceptToken)
	minted      map[string]time.Time // token -> expiry
	clients     map[string]string    // client id -> secret
	federated   map[string]string    // service account ref -> subject token
	refresh     map[string]bool
	mints       map[string]int // grant type -> count
	ttl         time.Duration
	mintCounter int
}

func newAuthState() authState {
	return authState{
		static:    map[string]bool{DefaultToken: true},
		minted:    map[string]time.Time{},
		clients:   map[string]string{},
		federated: map[string]string{},
		refresh:   map[string]bool{},
		mints:     map[string]int{},
		ttl:       time.Hour,
	}
}

func (s *Server) authorized(r *http.Request) bool {
	tok, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || tok == "" {
		return false
	}
	if s.auth.static[tok] {
		return true
	}
	exp, ok := s.auth.minted[tok]
	return ok && now().Before(exp)
}

// tokenRequest is types.AccessTokenRequest (pkg/apiserver/types/types.go:65-84).
type tokenRequest struct {
	GrantType        string `json:"grant_type"`
	RefreshToken     string `json:"refresh_token"`
	ClientID         string `json:"client_id"`
	ClientSecret     string `json:"client_secret"`
	SubjectToken     string `json:"subject_token"`
	SubjectTokenType string `json:"subject_token_type"`
	ServiceAccount   string `json:"serviceaccount"`
}

// handleToken mirrors loginHandler.exchangeIDToken (pkg/apiserver/handlers/login.go:374-520):
// unknown grant or missing inputs -> 400 validation error; bad credentials -> opaque 401 with no
// body; success -> types.AccessTokenResponse. Request shape per pkg/wfclient/refresh.go:90-135,
// pkg/wfclient/federated.go:42-57 and the stub in pkg/wfclient/refresh_test.go:40-60.
// ASSUMPTION: the ttl query parameter is ignored; SetTokenTTL controls expiry. No refresh token
// is returned (the real handler only returns one for upstream user sessions).
func (s *Server) handleToken(w http.ResponseWriter, r *http.Request, body []byte) {
	var req tokenRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON body")
		return
	}
	var valid bool
	switch req.GrantType {
	case GrantClientCredentials:
		id, secret := req.ClientID, req.ClientSecret
		if id == "" || secret == "" {
			// Basic auth is accepted too (login.go:488-493).
			id, secret, _ = r.BasicAuth()
		}
		if id == "" || secret == "" {
			writeValidation(w, "invalid client credentials", []FieldError{{Field: "client_id", ErrCode: "required", Message: "client_id and client_secret are required"}})
			return
		}
		want, ok := s.auth.clients[id]
		valid = ok && want == secret
	case GrantTokenExchange:
		if req.SubjectTokenType != SubjectTokenTypeJWT {
			writeValidation(w, "invalid token exchange", []FieldError{{Field: "subject_token_type", ErrCode: "invalidValue", Message: "subject_token_type must be " + SubjectTokenTypeJWT}})
			return
		}
		want, ok := s.auth.federated[req.ServiceAccount]
		valid = ok && req.SubjectToken != "" && want == req.SubjectToken
	case GrantRefreshToken:
		if req.RefreshToken == "" {
			writeValidation(w, "invalid refresh", []FieldError{{Field: "refresh_token", ErrCode: "required", Message: "refresh token is required"}})
			return
		}
		valid = s.auth.refresh[req.RefreshToken]
	default:
		writeValidation(w, "invalid refresh", []FieldError{{Field: "grant_type", ErrCode: "invalidValue", Message: "invalid grant type"}})
		return
	}
	if !valid {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, s.mint(req.GrantType))
}

// mint issues an unsigned JWT-shaped token carrying exp, so clients that inspect expiry work.
func (s *Server) mint(grant string) map[string]any {
	s.auth.mintCounter++
	s.auth.mints[grant]++
	issued := now()
	exp := issued.Add(s.auth.ttl)
	enc := base64.RawURLEncoding
	header := enc.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	claims, _ := json.Marshal(map[string]any{"sub": "fake", "iat": issued.Unix(), "exp": exp.Unix(), "jti": s.auth.mintCounter})
	tok := header + "." + enc.EncodeToString(claims) + "." + enc.EncodeToString([]byte("fake"))
	s.auth.minted[tok] = exp
	return map[string]any{
		"access_token": tok,
		"token_type":   "Bearer",
		"expires_in":   int(s.auth.ttl.Seconds()),
		"expires":      exp.Unix(),
	}
}

func (s *Server) withAuth(fn func(a *authState)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.auth)
}

// AcceptToken adds a static bearer token that never expires.
func (s *Server) AcceptToken(token string) { s.withAuth(func(a *authState) { a.static[token] = true }) }

// RevokeToken stops accepting a static or minted token.
func (s *Server) RevokeToken(token string) {
	s.withAuth(func(a *authState) { delete(a.static, token); delete(a.minted, token) })
}

// AddClientCredential registers a service-account client credential for the client_credentials grant.
func (s *Server) AddClientCredential(clientID, secret string) {
	s.withAuth(func(a *authState) { a.clients[clientID] = secret })
}

// AddFederatedIdentity registers the subject token accepted for a service account
// (fully-qualified ref, e.g. "tenant:workspace:name") in the token-exchange grant.
func (s *Server) AddFederatedIdentity(serviceAccount, subjectToken string) {
	s.withAuth(func(a *authState) { a.federated[serviceAccount] = subjectToken })
}

// AddRefreshToken registers a token accepted by the refresh_token grant.
func (s *Server) AddRefreshToken(token string) {
	s.withAuth(func(a *authState) { a.refresh[token] = true })
}

// SetTokenTTL sets the lifetime (and expires_in) of tokens minted from now on.
func (s *Server) SetTokenTTL(ttl time.Duration) { s.withAuth(func(a *authState) { a.ttl = ttl }) }

// ExpireMintedTokens makes every token minted so far invalid, as if it had expired.
func (s *Server) ExpireMintedTokens() { s.withAuth(func(a *authState) { clear(a.minted) }) }

// Mints returns how many tokens were minted for a grant type.
func (s *Server) Mints(grant string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.auth.mints[grant]
}
