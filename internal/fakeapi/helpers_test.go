// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakeapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

const (
	tenant    = "acme"
	workspace = "ws1"
	secretTok = "s3cr3t-token"
)

// identityKind models an ExternalIdentity-like kind (R18): a +encrypted string and a +encrypted map.
func identityKind() fakeapi.KindSpec {
	return fakeapi.KindSpec{
		Kind:             "ExternalIdentity",
		Group:            "integrations.appvia.io",
		Version:          "v3",
		APIPath:          "externalidentities",
		Scopes:           []fakeapi.Scope{fakeapi.ScopeTenant, fakeapi.ScopeWorkspace},
		HasStatus:        true,
		SensitiveStrings: []string{"spec.bearerToken.token"},
		SensitiveMaps:    []string{"spec.headers"},
		Defaults:         map[string]any{"spec.mode": "auto"},
	}
}

func identityRef(name string) fakeapi.Ref {
	k := identityKind()
	return fakeapi.Ref{Tenant: tenant, Workspace: workspace, Group: k.Group, Version: k.Version, APIPath: k.APIPath, Name: name}
}

const identityCollection = "/api/v3/resources/tenants/acme/workspaces/ws1/integrations.appvia.io/v3/externalidentities"

func identityObject(name string) map[string]any {
	return map[string]any{
		"apiVersion": "integrations.appvia.io/v3",
		"kind":       "ExternalIdentity",
		"metadata":   map[string]any{"name": name, "workspace": workspace},
		"spec": map[string]any{
			"bearerToken": map[string]any{"token": secretTok},
			"headers":     map[string]any{"a": "va", "b": "vb"},
		},
	}
}

func newServer(t *testing.T, kinds ...fakeapi.KindSpec) *fakeapi.Server {
	t.Helper()
	s := fakeapi.New(t)
	for _, k := range kinds {
		s.RegisterKind(k)
	}
	return s
}

type response struct {
	code   int
	header http.Header
	body   map[string]any
	raw    []byte
}

func call(t *testing.T, s *fakeapi.Server, method, path string, body any) response {
	t.Helper()
	return callWithToken(t, s, fakeapi.DefaultToken, method, path, body)
}

func callWithToken(t *testing.T, s *fakeapi.Server, token, method, path string, body any) response {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, s.URL+path, rd)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	out := response{code: resp.StatusCode, header: resp.Header, raw: raw}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out.body)
	}
	return out
}

// dig returns the value at a dotted path in a decoded JSON object.
func dig(obj map[string]any, path ...string) any {
	var cur any = obj
	for _, p := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[p]
	}
	return cur
}

func wantCode(t *testing.T, r response, code int) {
	t.Helper()
	if r.code != code {
		t.Fatalf("status = %d, want %d; body=%s", r.code, code, r.raw)
	}
}
