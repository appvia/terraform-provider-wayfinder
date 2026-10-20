// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

// Package client is a minimal REST client for the Wayfinder resource API (plan R8-R10).
//
// It never sends query parameters on resource routes (no apply, force or withSensitiveData),
// retries only GET requests, and logs only method, path and status (never bodies or tokens).
package client

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
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	// maxRetries is the number of GET retries after the first attempt (D-UA).
	maxRetries = 5
	// DefaultRetryBaseDelay is the first backoff delay when RetryBaseDelay is zero.
	DefaultRetryBaseDelay = 500 * time.Millisecond
	maxBackoff            = 30 * time.Second
	userAgentProduct      = "terraform-provider-wayfinder"
	// maxResponseBytes caps every response body read.
	maxResponseBytes = 32 << 20
)

var errResponseTooLarge = errors.New("response too large")

// TokenSource supplies the bearer token for each request.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// Client talks to one Wayfinder API server on behalf of one tenant.
type Client struct {
	// BaseURL is the API endpoint, e.g. https://api.on.wayfinder.run (no path).
	BaseURL     string
	Tenant      string
	TokenSource TokenSource
	// HTTP defaults to http.DefaultClient when nil.
	HTTP *http.Client
	// UserAgent defaults to UserAgent("dev") when empty.
	UserAgent string
	// RetryBaseDelay is the first GET retry backoff (doubled per attempt, capped at 30s) used
	// when the response carries no Retry-After header. Zero means DefaultRetryBaseDelay.
	RetryBaseDelay time.Duration
}

// UserAgent returns the provider User-Agent for a provider version (D-UA).
func UserAgent(version string) string { return userAgentProduct + "/" + version }

// Create POSTs obj to the collection of ref and returns the stored object.
func (c *Client) Create(ctx context.Context, ref Ref, obj map[string]any) (map[string]any, error) {
	path, err := ref.CollectionPath(c.Tenant)
	if err != nil {
		return nil, err
	}
	return c.object(ctx, http.MethodPost, path, obj)
}

// Get reads the object identified by ref. A missing object yields an error matching ErrNotFound.
func (c *Client) Get(ctx context.Context, ref Ref) (map[string]any, error) {
	path, err := ref.ItemPath(c.Tenant)
	if err != nil {
		return nil, err
	}
	return c.object(ctx, http.MethodGet, path, nil)
}

// Update PUTs obj to the item route of ref. obj should carry metadata.generation so a
// concurrent change yields an error matching ErrModified.
func (c *Client) Update(ctx context.Context, ref Ref, obj map[string]any) (map[string]any, error) {
	path, err := ref.ItemPath(c.Tenant)
	if err != nil {
		return nil, err
	}
	return c.object(ctx, http.MethodPut, path, obj)
}

// Delete requests deletion of the object identified by ref. Deletion may complete
// asynchronously (finalizers); callers poll Get until ErrNotFound.
func (c *Client) Delete(ctx context.Context, ref Ref) error {
	path, err := ref.ItemPath(c.Tenant)
	if err != nil {
		return err
	}
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) object(ctx context.Context, method, path string, obj map[string]any) (map[string]any, error) {
	// A nil map in an interface is non-nil to DoJSON and would be sent as a "null" body;
	// CDNs in front of Wayfinder reject a GET that carries a body.
	var body any
	if obj != nil {
		body = obj
	}
	out := map[string]any{}
	if err := c.DoJSON(ctx, method, path, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DoJSON sends one authenticated request to path (already escaped, beginning with "/"),
// JSON-encoding body when non-nil and decoding a non-empty response into out when non-nil.
// Non-2xx responses map to *ValidationError or *APIError. Only GET is retried.
func (c *Client) DoJSON(ctx context.Context, method, path string, body, out any) error {
	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding %s %s request: %w", method, path, err)
		}
		payload = b
	}
	for attempt := 0; ; attempt++ {
		resp, respBody, err := c.send(ctx, method, path, payload)
		if err != nil {
			return err
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if out == nil || len(bytes.TrimSpace(respBody)) == 0 {
				return nil
			}
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("decoding %s %s response: %w", method, path, err)
			}
			return nil
		}
		if method != http.MethodGet || attempt >= maxRetries || !retryable(resp.StatusCode) {
			return decodeError(method, path, resp, respBody)
		}
		if err := sleep(ctx, c.retryDelay(resp, attempt)); err != nil {
			return fmt.Errorf("retrying %s %s: %w", method, path, err)
		}
	}
}

// send performs one attempt and returns the response with its fully read body.
func (c *Client) send(ctx context.Context, method, path string, payload []byte) (*http.Response, []byte, error) {
	token, err := c.TokenSource.Token(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("obtaining Wayfinder token: %w", err)
	}
	var rd io.Reader
	if payload != nil {
		rd = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, "/")+path, rd)
	if err != nil {
		return nil, nil, fmt.Errorf("building %s %s request: %w", method, path, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", c.userAgent())
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("wayfinder API %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := readLimited(resp.Body, maxResponseBytes)
	tflog.Debug(ctx, "wayfinder API request", map[string]any{"method": method, "path": path, "status": resp.StatusCode})
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s %s response: %w", method, path, err)
	}
	return resp, respBody, nil
}

// readLimited reads at most max bytes from r; a longer body yields errResponseTooLarge.
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

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) userAgent() string {
	if c.UserAgent != "" {
		return c.UserAgent
	}
	return UserAgent("dev")
}

func retryable(status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// retryDelay honours Retry-After (delta-seconds or HTTP-date), else exponential backoff.
func (c *Client) retryDelay(resp *http.Response, attempt int) time.Duration {
	if ra := strings.TrimSpace(resp.Header.Get("Retry-After")); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil && secs >= 0 {
			return time.Duration(secs) * time.Second
		}
		if t, err := http.ParseTime(ra); err == nil {
			return max(time.Until(t), 0)
		}
	}
	base := c.RetryBaseDelay
	if base <= 0 {
		base = DefaultRetryBaseDelay
	}
	return min(base<<attempt, maxBackoff)
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-t.C:
		return nil
	}
}
