package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	grantClient   = "client_credentials"
	grantExchange = "urn:ietf:params:oauth:grant-type:token-exchange"
	grantRefresh  = "refresh_token"
)

// JSON fields/omitempty match pinned pkg/apiserver/types/types.go:65-84.
// Grants mirror pkg/wfclient/refresh.go:87-138 and federated.go:57-64;
// JSON decoding is asserted by pkg/wfclient/refresh_test.go:53-54 (not form data).
type tokenRequest struct {
	GrantType        string `json:"grant_type,omitempty"`
	RefreshToken     string `json:"refresh_token,omitempty"`
	ClientID         string `json:"client_id,omitempty"`
	ClientSecret     string `json:"client_secret,omitempty"`
	SubjectToken     string `json:"subject_token,omitempty"`
	SubjectTokenType string `json:"subject_token_type,omitempty"`
	ServiceAccount   string `json:"serviceaccount,omitempty"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Expires      int64  `json:"expires"`
}

type cachedSource struct {
	mu       sync.Mutex
	server   string
	http     *http.Client
	now      func() time.Time
	request  tokenRequest
	identity *federatedIdentity
	token    string
	expires  time.Time
}

var errEmptyToken = errors.New("authentication response contained no access token")

func (s *cachedSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("get access token: %w", err)
	}
	if s.token != "" && s.now().Add(time.Minute).Before(s.expires) {
		return s.token, nil
	}
	payload := s.request
	if s.identity != nil {
		subject, err := s.subjectToken(ctx)
		if err != nil {
			return "", fmt.Errorf("resolve federated identity: %w", err)
		}
		payload.SubjectToken = subject
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode token request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.server+"/api/v3/general/login/token", bytes.NewReader(body))
	if err != nil {
		return "", &safeError{operation: "create token request", cause: err}
	}
	req.Header.Set("Content-Type", "application/json")
	var issued tokenResponse
	if err := s.decode(req, &issued); err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}
	if issued.AccessToken == "" {
		return "", errEmptyToken
	}
	expires := s.now().Add(time.Duration(issued.ExpiresIn) * time.Second)
	switch {
	case issued.Expires > 0:
		expires = time.Unix(issued.Expires, 0)
	case issued.ExpiresIn > 0:
	default:
		claims, err := parseClaims(issued.AccessToken)
		if err == nil && claims.Exp > 0 {
			expires = time.Unix(claims.Exp, 0)
		}
	}
	s.token, s.expires = issued.AccessToken, expires
	if payload.GrantType == grantRefresh && issued.RefreshToken != "" {
		s.request.RefreshToken = issued.RefreshToken
	}
	return s.token, nil
}

// HTTPStatusError deliberately excludes response bodies, which may echo credentials.
type HTTPStatusError struct{ StatusCode int }

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("HTTP %d %s", e.StatusCode, http.StatusText(e.StatusCode))
}

// Keep error identity without rendering transport URLs or decoder input (secrets).
type safeError struct {
	operation string
	cause     error
}

func (e *safeError) Error() string { return e.operation + " failed" }
func (e *safeError) Unwrap() error { return e.cause }

func (s *cachedSource) decode(req *http.Request, out any) (err error) {
	resp, err := s.http.Do(req)
	if err != nil {
		return &safeError{operation: "authentication HTTP request", cause: err}
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = errors.Join(err, &safeError{operation: "close authentication response", cause: closeErr})
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return &HTTPStatusError{StatusCode: resp.StatusCode}
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return &safeError{operation: "decode authentication response", cause: err}
	}
	return nil
}
