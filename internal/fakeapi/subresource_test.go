// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakeapi_test

import (
	"net/http"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

// clientRef converts a fakeapi.Ref (which carries the tenant) into the client.Ref the hook takes.
func clientRef(r fakeapi.Ref, versioned bool) client.Ref {
	return client.Ref{
		Group: r.Group, Version: r.Version, APIPath: r.APIPath, Workspace: r.Workspace,
		Environment: r.Environment, Name: r.Name, ObjectVersion: r.ObjectVersion, Versioned: versioned,
	}
}

func TestSubresource_whenSet(t *testing.T) {
	s := newServer(t, identityKind())
	ref := identityRef("id1")
	s.SetSubresource(clientRef(ref, false), "secrets", http.StatusOK, map[string]any{"token": "abc"})

	got := call(t, s, http.MethodGet, identityCollection+"/id1/secrets", nil)

	wantCode(t, got, http.StatusOK)

	if dig(got.body, "token") != "abc" {
		t.Errorf("body = %s", got.raw)
	}

	if n := s.CountRequests(http.MethodGet, identityCollection+"/id1/secrets"); n != 1 {
		t.Errorf("logged requests = %d, want 1", n)
	}

	// Another action and another object stay unset.
	wantCode(t, call(t, s, http.MethodGet, identityCollection+"/id1/other", nil), http.StatusNotFound)
	wantCode(t, call(t, s, http.MethodGet, identityCollection+"/id2/secrets", nil), http.StatusNotFound)
	// Only GET is routed.
	wantCode(t, call(t, s, http.MethodPost, identityCollection+"/id1/secrets", nil), http.StatusNotFound)
}

func TestSubresource_whenUnset(t *testing.T) {
	s := newServer(t, identityKind())

	got := call(t, s, http.MethodGet, identityCollection+"/id1/secrets", nil)

	wantCode(t, got, http.StatusNotFound)

	reqs := s.Requests()
	if len(reqs) != 1 || reqs[0].Path != identityCollection+"/id1/secrets" || reqs[0].Status != http.StatusNotFound {
		t.Errorf("requests = %+v", reqs)
	}
}

func TestSubresource_whenSetDoesNotLeakAcrossServers(t *testing.T) {
	a := newServer(t, identityKind())
	b := newServer(t, identityKind())

	a.SetSubresource(clientRef(identityRef("id1"), false), "secrets", http.StatusOK, map[string]any{"token": "abc"})

	wantCode(t, call(t, b, http.MethodGet, identityCollection+"/id1/secrets", nil), http.StatusNotFound)
}

func TestSubresource_whenSetAtWorkspaceScope(t *testing.T) {
	s := newServer(t, identityKind())
	ref := identityRef("id1")
	s.SetSubresource(clientRef(ref, false), "secrets", http.StatusOK, map[string]any{"scope": "workspace"})

	wantCode(t, call(t, s, http.MethodGet, identityCollection+"/id1/secrets", nil), http.StatusOK)

	// Same name at tenant scope is a different object.
	tenantRef := ref
	tenantRef.Workspace = ""

	wantCode(t, call(t, s, http.MethodGet, "/api/v3/resources/tenants/acme/integrations.appvia.io/v3/externalidentities/id1/secrets", nil), http.StatusNotFound)

	s.SetSubresource(clientRef(tenantRef, false), "secrets", http.StatusOK, map[string]any{"scope": "tenant"})
	got := call(t, s, http.MethodGet, "/api/v3/resources/tenants/acme/integrations.appvia.io/v3/externalidentities/id1/secrets", nil)

	wantCode(t, got, http.StatusOK)

	if dig(got.body, "scope") != "tenant" {
		t.Errorf("body = %s", got.raw)
	}
}

func TestSubresource_whenSetAtEnvironmentScope(t *testing.T) {
	k := identityKind()
	k.Scopes = append(k.Scopes, fakeapi.ScopeEnvironment)
	s := newServer(t, k)
	ref := identityRef("id1")
	ref.Environment = "dev"
	s.SetSubresource(clientRef(ref, false), "secrets", http.StatusOK, map[string]any{"scope": "env"})

	got := call(t, s, http.MethodGet, "/api/v3/resources/tenants/acme/workspaces/ws1/envs/dev/integrations.appvia.io/v3/externalidentities/id1/secrets", nil)
	wantCode(t, got, http.StatusOK)

	if dig(got.body, "scope") != "env" {
		t.Errorf("body = %s", got.raw)
	}
}

func TestSubresource_whenSetOnVersionedKind(t *testing.T) {
	s := newServer(t, planKind())
	ref := fakeapi.Ref{Tenant: tenant, Group: "app.appvia.io", Version: "v3", APIPath: "appplans", Name: "p", ObjectVersion: "1.0.0"}
	s.SetSubresource(clientRef(ref, true), "secrets", http.StatusOK, map[string]any{"v": "1.0.0"})

	got := call(t, s, http.MethodGet, planCollection+"/p/versions/1.0.0/secrets", nil)

	wantCode(t, got, http.StatusOK)

	if dig(got.body, "v") != "1.0.0" {
		t.Errorf("body = %s", got.raw)
	}

	wantCode(t, call(t, s, http.MethodGet, planCollection+"/p/versions/1.1.0/secrets", nil), http.StatusNotFound)
}

// A response configured without an object version must not be served at the unversioned path:
// client.Ref.ItemPath rejects a versioned kind without a version, so the route does not exist.
func TestSubresource_whenVersionedKindMissingVersion(t *testing.T) {
	s := newServer(t, planKind())
	ref := fakeapi.Ref{Tenant: tenant, Group: "app.appvia.io", Version: "v3", APIPath: "appplans", Name: "p"}
	s.SetSubresource(clientRef(ref, true), "secrets", http.StatusOK, map[string]any{"token": "unexpected"})

	wantCode(t, call(t, s, http.MethodGet, planCollection+"/p/secrets", nil), http.StatusNotFound)
}

func TestSubresource_whenPostSet(t *testing.T) {
	s := newServer(t, identityKind())
	s.SetSubresourceMethod(clientRef(identityRef("id1"), false), http.MethodPost, "rotate", http.StatusOK, map[string]any{"url": "https://ingest.example/x"})

	got := call(t, s, http.MethodPost, identityCollection+"/id1/rotate", nil)

	wantCode(t, got, http.StatusOK)

	if dig(got.body, "url") != "https://ingest.example/x" {
		t.Errorf("body = %s", got.raw)
	}

	if n := s.CountRequests(http.MethodPost, identityCollection+"/id1/rotate"); n != 1 {
		t.Errorf("logged POST requests = %d, want 1", n)
	}
}

func TestSubresource_whenPostUnsetAndGetSet(t *testing.T) {
	s := newServer(t, identityKind())
	s.SetSubresource(clientRef(identityRef("id1"), false), "rotate", http.StatusOK, map[string]any{"get": true})

	wantCode(t, call(t, s, http.MethodPost, identityCollection+"/id1/rotate", nil), http.StatusNotFound)
}

func TestSubresource_whenPostSetDoesNotAffectGet(t *testing.T) {
	s := newServer(t, identityKind())
	s.SetSubresourceMethod(clientRef(identityRef("id1"), false), http.MethodPost, "rotate", http.StatusOK, map[string]any{"url": "u"})

	wantCode(t, call(t, s, http.MethodGet, identityCollection+"/id1/rotate", nil), http.StatusNotFound)
}
