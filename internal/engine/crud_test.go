// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func rawValue(t *testing.T, e testEnv, values map[string]any) tftypes.Value {
	t.Helper()
	m, err := convert.NewMapping(e.k)
	if err != nil {
		t.Fatal(err)
	}
	typ := m.TFType()
	a := map[string]tftypes.Value{}
	for name, child := range typ.AttributeTypes {
		a[name] = tftypes.NewValue(child, nil)
	}
	a["name"] = tftypes.NewValue(tftypes.String, "example")
	for name, value := range values {
		a[name] = tftypes.NewValue(typ.AttributeTypes[name], value)
	}
	return tftypes.NewValue(typ, a)
}

func directResource(e testEnv) *kindResource {
	r := newResource(e.k, testSchema(e.k), pollOptions{initial: time.Millisecond, maximum: 2 * time.Millisecond})
	r.client = e.c
	return r
}

func createDirect(t *testing.T, e testEnv, v tftypes.Value) resource.CreateResponse {
	t.Helper()
	r := directResource(e)
	resp := resource.CreateResponse{State: tfsdk.State{Raw: v, Schema: r.schema}}
	// The framework's private-state concrete type is internal; instantiate its
	// exported response field without importing or duplicating that type.
	f := reflect.ValueOf(&resp).Elem().FieldByName("Private")
	f.Set(reflect.New(f.Type().Elem()))
	r.Create(t.Context(), resource.CreateRequest{Config: tfsdk.Config{Raw: v, Schema: r.schema}, Plan: tfsdk.Plan{Raw: v, Schema: r.schema}}, &resp)
	return resp
}

func assertDiagnostics(t *testing.T, d diag.Diagnostics) {
	t.Helper()
	if d.HasError() {
		t.Fatalf("diagnostics: %v", d)
	}
}

func TestPrivateGeneration_whenRefreshSeesOutsideEdit(t *testing.T) {
	// Given a successful create and a later outside change.
	e := newTestEnv(t, "Workspace")
	v := rawValue(t, e, map[string]any{"display_name": "First", "cloud_provider": "aws"})
	created := createDirect(t, e, v)
	assertDiagnostics(t, created.Diagnostics)
	b, d := created.Private.GetKey(t.Context(), "generation")
	assertDiagnostics(t, d)
	if string(b) != "1" {
		t.Fatalf("created generation = %s", b)
	}
	e.fake.Mutate(e.ref("", "", ""), func(map[string]any) {})
	r := directResource(e)
	read := resource.ReadResponse{State: created.State, Private: created.Private}
	// When ordinary refresh reads the newer generation.
	r.Read(t.Context(), resource.ReadRequest{State: created.State, Private: created.Private}, &read)
	// Then the baseline stays unchanged and the next update sends no PUT.
	assertDiagnostics(t, read.Diagnostics)
	b, d = read.Private.GetKey(t.Context(), "generation")
	assertDiagnostics(t, d)
	if string(b) != "1" {
		t.Fatalf("refresh overwrote generation: %s", b)
	}
	updated := resource.UpdateResponse{State: read.State, Private: read.Private}
	r.Update(t.Context(), resource.UpdateRequest{State: read.State, Private: read.Private, Config: tfsdk.Config{Raw: v, Schema: r.schema}, Plan: tfsdk.Plan{Raw: v, Schema: r.schema}}, &updated)
	if !updated.Diagnostics.HasError() || !strings.Contains(updated.Diagnostics[0].Detail(), "changed outside Terraform") {
		t.Fatalf("diagnostics = %v", updated.Diagnostics)
	}
	if n := e.fake.CountRequests(http.MethodPut, ""); n != 0 {
		t.Fatalf("PUT count = %d", n)
	}
	t.Log("stale generation: PUT request count = 0; refresh preserved generation 1")
}

func TestPrivateGeneration_whenImportedAndUpdated(t *testing.T) {
	// Given a resource whose import has no generation baseline.
	e := newTestEnv(t, "Workspace")
	e.fake.Seed(e.ref("", "", ""), map[string]any{"spec": map[string]any{"displayName": "First", "cloudProvider": "aws"}})
	r := directResource(e)
	v := rawValue(t, e, map[string]any{"display_name": "First", "cloud_provider": "aws"})
	resp := resource.ReadResponse{State: tfsdk.State{Raw: v, Schema: r.schema}}
	f := reflect.ValueOf(&resp).Elem().FieldByName("Private")
	f.Set(reflect.New(f.Type().Elem()))
	// When the first post-import read runs.
	r.Read(t.Context(), resource.ReadRequest{State: resp.State}, &resp)
	// Then it records the generation, and a successful update replaces it.
	assertDiagnostics(t, resp.Diagnostics)
	b, d := resp.Private.GetKey(t.Context(), "generation")
	assertDiagnostics(t, d)
	if string(b) != "1" {
		t.Fatalf("import generation = %s", b)
	}
	updated := resource.UpdateResponse{State: resp.State, Private: resp.Private}
	r.Update(t.Context(), resource.UpdateRequest{State: resp.State, Private: resp.Private, Config: tfsdk.Config{Raw: v, Schema: r.schema}, Plan: tfsdk.Plan{Raw: v, Schema: r.schema}}, &updated)
	assertDiagnostics(t, updated.Diagnostics)
	b, d = updated.Private.GetKey(t.Context(), "generation")
	assertDiagnostics(t, d)
	if string(b) != "2" {
		t.Fatalf("update generation = %s", b)
	}
}

func TestRead_whenObjectMissingRemovesState(t *testing.T) {
	// Given state for an object absent from Wayfinder.
	e := newTestEnv(t, "Workspace")
	r := directResource(e)
	v := rawValue(t, e, nil)
	resp := resource.ReadResponse{State: tfsdk.State{Raw: v, Schema: r.schema}}
	// When refreshing.
	r.Read(t.Context(), resource.ReadRequest{State: resp.State}, &resp)
	// Then Terraform state is null.
	assertDiagnostics(t, resp.Diagnostics)
	if !resp.State.Raw.IsNull() {
		t.Fatal("resource was not removed")
	}
}

func TestDelete_whenFinalizersRemainWaitsFor404(t *testing.T) {
	// Given an asynchronously deleted object.
	e := newTestEnv(t, "Workspace")
	k := kindSpec(e.k)
	k.FinalizerGETs = 3
	e.fake.RegisterKind(k)
	e.fake.Seed(e.ref("", "", ""), map[string]any{})
	r := directResource(e)
	resp := resource.DeleteResponse{}
	// When deleting.
	r.Delete(t.Context(), resource.DeleteRequest{State: tfsdk.State{Raw: rawValue(t, e, nil), Schema: r.schema}}, &resp)
	// Then one DELETE and four GETs prove finalizer completion, without mutation retries.
	assertDiagnostics(t, resp.Diagnostics)
	if n := e.fake.CountRequests(http.MethodDelete, ""); n != 1 {
		t.Fatalf("DELETE = %d", n)
	}
	if n := e.fake.CountRequests(http.MethodGet, ""); n != 4 {
		t.Fatalf("GET = %d", n)
	}
	t.Log("finalizer deletion: DELETE count = 1, GET count = 4 (final GET 404)")
}

func TestWait_whenTerminalStatusOrTimeout(t *testing.T) {
	for _, tc := range []struct {
		status, detail         string
		wantError, wantWarning bool
	}{
		{"Success", "", false, false},
		{"Complete", "", false, false},
		{"ApprovalRequired", "", false, true},
		{"Warning", "", false, true},
		{"ActionRequired", "", false, true},
		{"Failure", "failed detail", true, false},
		{"Error", "error detail", true, false},
		{"Pending", "", true, false},
	} {
		t.Run(tc.status, func(t *testing.T) {
			// Given a status script and a short operation deadline.
			e := newTestEnv(t, "Workspace")
			k := kindSpec(e.k)
			k.StatusScript = []fakeapi.StatusStep{{Status: "Pending"}, {Status: tc.status, Message: "controller message", Detail: tc.detail}}
			e.fake.RegisterKind(k)
			// When creating through the resource lifecycle.
			resp := createDirect(t, e, rawValue(t, e, map[string]any{"display_name": "First", "cloud_provider": "aws", "timeouts": map[string]tftypes.Value{"create": tftypes.NewValue(tftypes.String, "20ms"), "update": tftypes.NewValue(tftypes.String, nil), "delete": tftypes.NewValue(tftypes.String, nil)}}))
			// Then readiness has the requested severity and retains failure detail.
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics = %v", resp.Diagnostics)
			}
			if tc.wantWarning && (len(resp.Diagnostics) != 1 || resp.Diagnostics[0].Severity() != diag.SeverityWarning) {
				t.Fatalf("warning = %v", resp.Diagnostics)
			}
			if tc.detail != "" && !strings.Contains(resp.Diagnostics[0].Detail(), tc.detail) {
				t.Fatalf("missing detail: %v", resp.Diagnostics)
			}
			if tc.status == "Pending" && !strings.Contains(resp.Diagnostics[0].Detail(), "Pending") {
				t.Fatalf("missing last status: %v", resp.Diagnostics)
			}
		})
	}
}

func TestWait_whenOperationTokenNotAcknowledged(t *testing.T) {
	// Given cancellation before the first poll.
	e := newTestEnv(t, "Workspace")
	r := directResource(e)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	// When waiting with cancellation.
	_, d := r.wait(ctx, reference(e.k, convert.Identity{Name: "example"}), false)
	// Then cancellation is an error rather than successful readiness.
	if !d.HasError() {
		t.Fatal("cancelled wait succeeded")
	}
}
