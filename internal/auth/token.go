// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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

	issued, err := s.issue(ctx, payload)
	if err != nil {
		return "", err
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

// issue posts the token request, retrying while the endpoint is rate limited: Wayfinder limits
// unauthenticated requests per client IP, and every Terraform command mints a token.
func (s *cachedSource) issue(ctx context.Context, payload tokenRequest) (tokenResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return tokenResponse{}, fmt.Errorf("encode token request: %w", err)
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.server+"/api/v3/general/login/token", bytes.NewReader(body))
		if err != nil {
			return tokenResponse{}, &safeError{operation: "create token request", cause: err}
		}

		req.Header.Set("Content-Type", "application/json")

		var issued tokenResponse

		err = s.decode(req, &issued)

		var status *HTTPStatusError
		if err == nil || attempt >= maxRateLimitRetries || !errors.As(err, &status) || status.StatusCode != http.StatusTooManyRequests {
			if err != nil {
				return tokenResponse{}, fmt.Errorf("authentication failed: %w", err)
			}

			return issued, nil
		}

		if err := wait(ctx, status.retryAfter(attempt)); err != nil {
			return tokenResponse{}, fmt.Errorf("authentication failed: %w", err)
		}
	}
}

const (
	maxRateLimitRetries  = 6
	rateLimitBaseDelay   = 250 * time.Millisecond
	maxAuthResponseBytes = 1 << 20
)

var errResponseTooLarge = errors.New("response too large")

func readLimited(r io.Reader, max int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}

	if int64(len(b)) > max {
		return nil, fmt.Errorf("%w: exceeds %d bytes", errResponseTooLarge, max)
	}

	return b, nil
}

func wait(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-t.C:
		return nil
	}
}

// HTTPStatusError deliberately excludes response bodies, which may echo credentials.
type HTTPStatusError struct {
	StatusCode int
	// RetryAfter is the delay a 429 response asked for, or nil when it named none.
	RetryAfter *time.Duration
}

// retryAfter is the server-requested delay, else exponential backoff from rateLimitBaseDelay.
func (e *HTTPStatusError) retryAfter(attempt int) time.Duration {
	if e.RetryAfter != nil {
		return *e.RetryAfter
	}

	return rateLimitBaseDelay << attempt
}

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
		return &HTTPStatusError{StatusCode: resp.StatusCode, RetryAfter: retryAfterHeader(resp.Header.Get("Retry-After"))}
	}

	body, err := readLimited(resp.Body, maxAuthResponseBytes)
	if errors.Is(err, errResponseTooLarge) {
		return &safeError{operation: fmt.Sprintf("read authentication response (exceeds %d bytes)", maxAuthResponseBytes), cause: err}
	}

	if err != nil {
		return &safeError{operation: "read authentication response", cause: err}
	}

	if err := json.Unmarshal(body, out); err != nil {
		return &safeError{operation: "decode authentication response", cause: err}
	}

	return nil
}

// retryAfterHeader parses a delta-seconds Retry-After value.
func retryAfterHeader(v string) *time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || secs < 0 {
		return nil
	}

	d := time.Duration(secs) * time.Second

	return &d
}
