// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakeapi_test

import (
	"net/http"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

const planCollection = "/api/v3/resources/tenants/acme/app.appvia.io/v3/appplans"

func planKind() fakeapi.KindSpec {
	return fakeapi.KindSpec{
		Kind: "AppPlan", Group: "app.appvia.io", Version: "v3", APIPath: "appplans",
		Scopes: []fakeapi.Scope{fakeapi.ScopeTenant}, Versioned: true,
	}
}

func planObject(version string) map[string]any {
	return map[string]any{
		"metadata": map[string]any{"name": "p", "version": version},
		"spec":     map[string]any{"size": "small"},
	}
}

func TestVersioned_create_get_and_update_use_version_routes(t *testing.T) {
	s := newServer(t, planKind())
	wantCode(t, call(t, s, http.MethodPost, planCollection, planObject("1.0.0")), http.StatusOK)
	wantCode(t, call(t, s, http.MethodPost, planCollection, planObject("1.1.0")), http.StatusOK)

	v1 := call(t, s, http.MethodGet, planCollection+"/p/versions/1.0.0", nil)
	upd := call(t, s, http.MethodPut, planCollection+"/p/versions/1.1.0", planObject("1.1.0"))

	wantCode(t, v1, http.StatusOK)
	if dig(v1.body, "metadata", "version") != "1.0.0" {
		t.Errorf("version = %v", dig(v1.body, "metadata", "version"))
	}
	wantCode(t, upd, http.StatusOK)
	if g := dig(upd.body, "metadata", "generation"); g != float64(2) {
		t.Errorf("generation = %v, want 2", g)
	}
	wantCode(t, call(t, s, http.MethodGet, planCollection+"/p/versions/9.9.9", nil), http.StatusNotFound)
}

func TestVersioned_unversioned_put_returns_500(t *testing.T) {
	s := newServer(t, planKind())
	wantCode(t, call(t, s, http.MethodPost, planCollection, planObject("1.0.0")), http.StatusOK)

	got := call(t, s, http.MethodPut, planCollection+"/p", planObject("1.0.0"))

	wantCode(t, got, http.StatusInternalServerError)
}

func TestVersioned_put_rejects_body_version_differing_from_url(t *testing.T) {
	s := newServer(t, planKind())
	wantCode(t, call(t, s, http.MethodPost, planCollection, planObject("1.0.0")), http.StatusOK)

	got := call(t, s, http.MethodPut, planCollection+"/p/versions/1.0.0", planObject("2.0.0"))

	wantCode(t, got, http.StatusBadRequest)
}

func TestVersioned_create_rejects_non_semver_version(t *testing.T) {
	s := newServer(t, planKind())

	got := call(t, s, http.MethodPost, planCollection, planObject("v1.0.0"))

	wantCode(t, got, http.StatusBadRequest)
	fes, _ := got.body["fieldErrors"].([]any)
	if len(fes) != 1 || dig(fes[0].(map[string]any), "field") != "metadata.version" {
		t.Errorf("body = %s", got.raw)
	}
}

func TestVersioned_delete_removes_only_that_version(t *testing.T) {
	s := newServer(t, planKind())
	wantCode(t, call(t, s, http.MethodPost, planCollection, planObject("1.0.0")), http.StatusOK)
	wantCode(t, call(t, s, http.MethodPost, planCollection, planObject("1.1.0")), http.StatusOK)

	wantCode(t, call(t, s, http.MethodDelete, planCollection+"/p/versions/1.0.0", nil), http.StatusOK)

	wantCode(t, call(t, s, http.MethodGet, planCollection+"/p/versions/1.0.0", nil), http.StatusNotFound)
	wantCode(t, call(t, s, http.MethodGet, planCollection+"/p/versions/1.1.0", nil), http.StatusOK)
}
