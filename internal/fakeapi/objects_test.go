// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakeapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

func TestStore_whenIntegerExceedsFloat64Precision(t *testing.T) {
	// Given an object holding an integer above 2^53.
	s := newServer(t, identityKind())
	obj := identityObject("big")
	obj["spec"].(map[string]any)["limit"] = int64(9007199254740993)

	// When it is created and read back.
	wantCode(t, call(t, s, http.MethodPost, identityCollection, obj), http.StatusOK)

	got := call(t, s, http.MethodGet, identityCollection+"/big", nil)

	// Then the exact digits are served.
	wantCode(t, got, http.StatusOK)

	if !bytes.Contains(got.raw, []byte("9007199254740993")) || bytes.Contains(got.raw, []byte("9007199254740992")) {
		t.Errorf("body = %s, want exact 9007199254740993", got.raw)
	}
}

func TestSeed_whenIntegerExceedsFloat64Precision(t *testing.T) {
	// Given a seeded object holding an integer above 2^53.
	s := newServer(t, identityKind())
	obj := identityObject("seeded")
	obj["spec"].(map[string]any)["limit"] = int64(9007199254740993)

	// When it is seeded and read back through the test accessor.
	s.Seed(identityRef("seeded"), obj)

	stored, _ := s.Object(identityRef("seeded"))

	// Then the stored value keeps its exact digits.
	if got := dig(stored, "spec", "limit"); got != json.Number("9007199254740993") {
		t.Errorf("spec.limit = %v (%T), want json.Number 9007199254740993", got, got)
	}
}

func TestCreate_then_get_returns_generation_1_server_fields_and_blanked_secrets(t *testing.T) {
	// Given
	s := newServer(t, identityKind())
	wantCode(t, call(t, s, http.MethodPost, identityCollection, identityObject("id1")), http.StatusOK)

	// When
	got := call(t, s, http.MethodGet, identityCollection+"/id1", nil)

	// Then
	wantCode(t, got, http.StatusOK)

	if g := dig(got.body, "metadata", "generation"); g != float64(1) {
		t.Errorf("generation = %v, want 1", g)
	}

	if dig(got.body, "metadata", "uid") == "" || dig(got.body, "metadata", "uid") == nil {
		t.Error("uid not set")
	}

	if dig(got.body, "metadata", "creationTimestamp") == nil {
		t.Error("creationTimestamp not set")
	}

	if l := dig(got.body, "metadata", "labels", fakeapi.ManagedByLabel); l != "true" {
		t.Errorf("server-added label = %v", l)
	}

	if d := dig(got.body, "spec", "mode"); d != "auto" {
		t.Errorf("server default spec.mode = %v", d)
	}

	if tok := dig(got.body, "spec", "bearerToken", "token"); tok != "" {
		t.Errorf("sensitive string returned as %q, want blank", tok)
	}

	headers, _ := dig(got.body, "spec", "headers").(map[string]any)
	if len(headers) != 2 || headers["a"] != "" || headers["b"] != "" {
		t.Errorf("sensitive map = %v, want keys a,b with blank values", headers)
	}

	stored, _ := s.Object(identityRef("id1"))
	if dig(stored, "spec", "bearerToken", "token") != secretTok {
		t.Error("stored secret lost")
	}
}

func TestCreate_returns_409_with_already_exists_header_when_object_exists(t *testing.T) {
	s := newServer(t, identityKind())
	wantCode(t, call(t, s, http.MethodPost, identityCollection, identityObject("id1")), http.StatusOK)

	got := call(t, s, http.MethodPost, identityCollection, identityObject("id1"))

	wantCode(t, got, http.StatusConflict)

	if h := got.header.Get(fakeapi.HeaderObjectAlreadyExists); h != "true" {
		t.Errorf("%s = %q", fakeapi.HeaderObjectAlreadyExists, h)
	}
}

func TestUpdate_returns_404_when_object_missing(t *testing.T) {
	s := newServer(t, identityKind())

	got := call(t, s, http.MethodPut, identityCollection+"/nope", identityObject("nope"))

	wantCode(t, got, http.StatusNotFound)
}

func TestUpdate_returns_409_object_modified_when_generation_is_stale(t *testing.T) {
	s := newServer(t, identityKind())
	wantCode(t, call(t, s, http.MethodPost, identityCollection, identityObject("id1")), http.StatusOK)
	s.Mutate(identityRef("id1"), func(obj map[string]any) { obj["spec"].(map[string]any)["mode"] = "ui-edit" })

	stale := identityObject("id1")
	stale["metadata"].(map[string]any)["generation"] = 1

	got := call(t, s, http.MethodPut, identityCollection+"/id1", stale)

	wantCode(t, got, http.StatusConflict)

	if h := got.header.Get(fakeapi.HeaderObjectModified); h != "true" {
		t.Errorf("%s = %q", fakeapi.HeaderObjectModified, h)
	}

	if got.body["message"] != "OBJECT_MODIFIED" {
		t.Errorf("message = %v", got.body["message"])
	}

	if n := s.CountRequests(http.MethodPut, ""); n != 1 {
		t.Errorf("PUT count = %d", n)
	}
}

func TestUpdate_backfills_blank_sensitive_string_and_bumps_generation(t *testing.T) {
	s := newServer(t, identityKind())
	wantCode(t, call(t, s, http.MethodPost, identityCollection, identityObject("id1")), http.StatusOK)
	fresh := call(t, s, http.MethodGet, identityCollection+"/id1", nil).body

	got := call(t, s, http.MethodPut, identityCollection+"/id1", fresh)

	wantCode(t, got, http.StatusOK)

	if g := dig(got.body, "metadata", "generation"); g != float64(2) {
		t.Errorf("generation = %v, want 2", g)
	}

	stored, _ := s.Object(identityRef("id1"))
	if dig(stored, "spec", "bearerToken", "token") != secretTok {
		t.Errorf("blank secret not backfilled: %v", dig(stored, "spec", "bearerToken", "token"))
	}
}

func TestUpdate_merges_sensitive_map_key_wise(t *testing.T) {
	s := newServer(t, identityKind())
	wantCode(t, call(t, s, http.MethodPost, identityCollection, identityObject("id1")), http.StatusOK)
	obj := identityObject("id1")
	// a: blank -> keep stored; b: omitted -> dropped; c: new value -> stored.
	obj["spec"].(map[string]any)["headers"] = map[string]any{"a": "", "c": "vc"}

	wantCode(t, call(t, s, http.MethodPut, identityCollection+"/id1", obj), http.StatusOK)

	stored, _ := s.Object(identityRef("id1"))

	headers := dig(stored, "spec", "headers").(map[string]any)
	if len(headers) != 2 || headers["a"] != "va" || headers["c"] != "vc" {
		t.Errorf("merged map = %v, want {a:va c:vc}", headers)
	}
}

func TestUpdate_rotates_non_blank_sensitive_map_value(t *testing.T) {
	s := newServer(t, identityKind())
	wantCode(t, call(t, s, http.MethodPost, identityCollection, identityObject("id1")), http.StatusOK)
	obj := identityObject("id1")
	obj["spec"].(map[string]any)["headers"] = map[string]any{"a": "rotated", "b": ""}

	wantCode(t, call(t, s, http.MethodPut, identityCollection+"/id1", obj), http.StatusOK)

	stored, _ := s.Object(identityRef("id1"))

	headers := dig(stored, "spec", "headers").(map[string]any)
	if headers["a"] != "rotated" || headers["b"] != "vb" {
		t.Errorf("map = %v, want {a:rotated b:vb}", headers)
	}
}

func TestDelete_returns_404_when_object_missing(t *testing.T) {
	s := newServer(t, identityKind())

	got := call(t, s, http.MethodDelete, identityCollection+"/nope", nil)

	wantCode(t, got, http.StatusNotFound)
}

func TestDelete_removes_object_immediately_without_finalizers(t *testing.T) {
	s := newServer(t, identityKind())
	wantCode(t, call(t, s, http.MethodPost, identityCollection, identityObject("id1")), http.StatusOK)

	wantCode(t, call(t, s, http.MethodDelete, identityCollection+"/id1", nil), http.StatusOK)

	wantCode(t, call(t, s, http.MethodGet, identityCollection+"/id1", nil), http.StatusNotFound)
}

func TestDelete_with_finalizers_marks_deletion_and_disappears_after_n_gets(t *testing.T) {
	k := identityKind()
	k.FinalizerGETs = 2
	s := newServer(t, k)
	wantCode(t, call(t, s, http.MethodPost, identityCollection, identityObject("id1")), http.StatusOK)

	del := call(t, s, http.MethodDelete, identityCollection+"/id1", nil)

	wantCode(t, del, http.StatusOK)

	for i := range 2 {
		got := call(t, s, http.MethodGet, identityCollection+"/id1", nil)
		wantCode(t, got, http.StatusOK)

		if dig(got.body, "metadata", "deletionTimestamp") == nil {
			t.Fatalf("GET %d: deletionTimestamp not set", i)
		}
	}

	wantCode(t, call(t, s, http.MethodGet, identityCollection+"/id1", nil), http.StatusNotFound)
}

func TestValidation_hook_returns_400_validation_error_body(t *testing.T) {
	k := identityKind()
	k.Validate = func(_ fakeapi.Operation, _ map[string]any) []fakeapi.FieldError {
		return []fakeapi.FieldError{{Field: "spec.mode", ErrCode: "invalidValue", Message: "bad mode"}}
	}
	s := newServer(t, k)

	got := call(t, s, http.MethodPost, identityCollection, identityObject("id1"))

	wantCode(t, got, http.StatusBadRequest)

	fes, _ := got.body["fieldErrors"].([]any)
	if len(fes) != 1 || dig(fes[0].(map[string]any), "field") != "spec.mode" || got.body["message"] == "" {
		t.Errorf("validation body = %s", got.raw)
	}

	if _, found := s.Object(identityRef("id1")); found {
		t.Error("invalid object was stored")
	}
}

func TestMutate_bumps_generation_like_an_out_of_band_ui_edit(t *testing.T) {
	s := newServer(t, identityKind())
	s.Seed(identityRef("id1"), identityObject("id1"))

	s.Mutate(identityRef("id1"), func(obj map[string]any) { obj["spec"].(map[string]any)["mode"] = "edited" })

	got := call(t, s, http.MethodGet, identityCollection+"/id1", nil)
	if g := dig(got.body, "metadata", "generation"); g != float64(2) {
		t.Errorf("generation = %v, want 2", g)
	}

	if m := dig(got.body, "spec", "mode"); m != "edited" {
		t.Errorf("spec.mode = %v", m)
	}
}

func TestRequests_logs_method_path_status_without_bodies_by_default(t *testing.T) {
	s := newServer(t, identityKind())
	call(t, s, http.MethodPost, identityCollection, identityObject("id1"))

	reqs := s.Requests()

	if len(reqs) != 1 || reqs[0].Method != http.MethodPost || reqs[0].Path != identityCollection || reqs[0].Status != http.StatusOK {
		t.Fatalf("requests = %+v", reqs)
	}

	if reqs[0].Body != nil {
		t.Error("body recorded without opt-in")
	}
}

func TestUnregistered_kind_or_scope_returns_404(t *testing.T) {
	s := newServer(t, identityKind())

	envScoped := call(t, s, http.MethodGet, "/api/v3/resources/tenants/acme/workspaces/ws1/envs/dev/integrations.appvia.io/v3/externalidentities/x", nil)
	unknown := call(t, s, http.MethodGet, "/api/v3/resources/tenants/acme/other.appvia.io/v3/things/x", nil)

	wantCode(t, envScoped, http.StatusNotFound)
	wantCode(t, unknown, http.StatusNotFound)
}

func TestConcurrent_creates_are_all_stored_without_races(t *testing.T) {
	s := newServer(t, identityKind())

	var wg sync.WaitGroup

	for i := range 20 {
		wg.Go(func() { call(t, s, http.MethodPost, identityCollection, identityObject("c"+strconv.Itoa(i))) })
	}

	wg.Wait()

	if n := s.CountRequests(http.MethodPost, identityCollection); n != 20 {
		t.Errorf("POST count = %d", n)
	}

	for i := range 20 {
		if _, ok := s.Object(identityRef("c" + strconv.Itoa(i))); !ok {
			t.Errorf("object c%d missing", i)
		}
	}
}
