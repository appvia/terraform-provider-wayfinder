// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

const tenant = "acme"

type staticToken string

func (s staticToken) Token(context.Context) (string, error) { return string(s), nil }

// environmentKind is a workspace-scoped kind (org/v3/environments).
func environmentKind() fakeapi.KindSpec {
	return fakeapi.KindSpec{
		Kind: "Environment", Group: "org", Version: "v3", APIPath: "environments",
		Scopes: []fakeapi.Scope{fakeapi.ScopeWorkspace}, HasStatus: true,
		Validate: func(_ fakeapi.Operation, obj map[string]any) []fakeapi.FieldError {
			spec, _ := obj["spec"].(map[string]any)
			if spec["stage"] == "bogus" {
				return []fakeapi.FieldError{{Field: "spec.stage", ErrCode: "invalidValue", Message: "stage must be one of dev, prod"}}
			}
			return nil
		},
	}
}

// planKind is a versioned tenant-scoped kind.
func planKind() fakeapi.KindSpec {
	return fakeapi.KindSpec{
		Kind: "KubeResourcePlan", Group: "kubeplans.appvia.io", Version: "v3", APIPath: "kuberesourceplans",
		Scopes: []fakeapi.Scope{fakeapi.ScopeTenant}, Versioned: true,
	}
}

func envRef(name string) client.Ref {
	return client.Ref{Group: "org", Version: "v3", APIPath: "environments", Workspace: "plat", Name: name}
}

func fakeEnvRef(name string) fakeapi.Ref {
	return fakeapi.Ref{Tenant: tenant, Workspace: "plat", Group: "org", Version: "v3", APIPath: "environments", Name: name}
}

func envObject(name string) map[string]any {
	return map[string]any{
		"apiVersion": "org/v3",
		"kind":       "Environment",
		"metadata":   map[string]any{"name": name, "workspace": "plat"},
		"spec":       map[string]any{"stage": "dev"},
	}
}

// recorded is what the wrapper saw for one request.
type recorded struct {
	Method, Path, RawQuery string
	Header                 http.Header
}

// harness fronts a fakeapi.Server with a handler that records headers and can inject
// failure responses before delegating to the fake.
type harness struct {
	t    *testing.T
	fake *fakeapi.Server
	srv  *httptest.Server

	mu      sync.Mutex
	seen    []recorded
	inject  []func(w http.ResponseWriter) bool // consumed in order; true = handled
	matchFn func(r *http.Request) bool
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{t: t, fake: fakeapi.New(t)}
	h.fake.RegisterKind(environmentKind())
	h.fake.RegisterKind(planKind())
	h.srv = httptest.NewServer(http.HandlerFunc(h.serve))
	t.Cleanup(h.srv.Close)
	return h
}

func (h *harness) serve(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	h.seen = append(h.seen, recorded{Method: r.Method, Path: r.URL.Path, RawQuery: r.URL.RawQuery, Header: r.Header.Clone()})
	var fn func(w http.ResponseWriter) bool
	if len(h.inject) > 0 && (h.matchFn == nil || h.matchFn(r)) {
		fn, h.inject = h.inject[0], h.inject[1:]
	}
	h.mu.Unlock()
	if fn != nil && fn(w) {
		return
	}
	h.fake.ServeHTTP(w, r)
}

// failWith queues n responses with the given status (and optional Retry-After) for
// requests matching method.
func (h *harness) failWith(method string, status, n int, retryAfter string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.matchFn = func(r *http.Request) bool { return r.Method == method }
	for range n {
		h.inject = append(h.inject, func(w http.ResponseWriter) bool {
			if retryAfter != "" {
				w.Header().Set("Retry-After", retryAfter)
			}
			http.Error(w, `{"message":"try later"}`, status)
			return true
		})
	}
}

func (h *harness) requests() []recorded {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]recorded(nil), h.seen...)
}

func (h *harness) count(method string) int {
	n := 0
	for _, r := range h.requests() {
		if r.Method == method {
			n++
		}
	}
	return n
}

func (h *harness) client() *client.Client {
	return &client.Client{
		BaseURL:        h.srv.URL,
		Tenant:         tenant,
		TokenSource:    staticToken(fakeapi.DefaultToken),
		HTTP:           h.srv.Client(),
		UserAgent:      client.UserAgent("1.2.3"),
		RetryBaseDelay: time.Millisecond,
	}
}
