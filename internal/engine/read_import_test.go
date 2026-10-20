// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestRead_whenImportHydratesOptionalFieldsAndSetsGeneration(t *testing.T) {
	// Given identity-only imported state and a server optional field.
	e := newTestEnv(t, "Integration")
	e.fake.Seed(e.ref("ws", "", ""), map[string]any{"spec": map[string]any{"endpoint": "https://example", "timeoutSeconds": 30}})
	r := directResource(e)
	state := tfsdk.State{Raw: rawValue(t, e, nil), Schema: r.schema}
	imported := resource.ImportStateResponse{State: state}
	r.ImportState(t.Context(), resource.ImportStateRequest{ID: "ws/example"}, &imported)
	assertDiagnostics(t, imported.Diagnostics)
	resp := resource.ReadResponse{State: imported.State}
	f := reflect.ValueOf(&resp).Elem().FieldByName("Private")
	f.Set(reflect.New(f.Type().Elem()))
	// When the first read has no private generation.
	r.Read(t.Context(), resource.ReadRequest{State: imported.State}, &resp)
	// Then the optional field and generation baseline are adopted.
	assertDiagnostics(t, resp.Diagnostics)

	var attrs map[string]tftypes.Value
	if err := resp.State.Raw.As(&attrs); err != nil {
		t.Fatal(err)
	}

	if !attrs["timeout_seconds"].Equal(tftypes.NewValue(tftypes.Number, 30)) {
		t.Fatalf("timeout_seconds = %s", attrs["timeout_seconds"])
	}

	b, d := resp.Private.GetKey(t.Context(), "generation")
	assertDiagnostics(t, d)

	if string(b) != "1" {
		t.Fatalf("generation = %s", b)
	}
}

func TestRead_whenCreatedResourceIgnoresNewServerDefault(t *testing.T) {
	// Given Create-established private state and a new out-of-band optional default.
	e := newTestEnv(t, "Integration")
	v := rawValue(t, e, map[string]any{"workspace": "ws", "endpoint": "https://example", "api_key_wo": "secret"})
	created := createDirect(t, e, v)
	assertDiagnostics(t, created.Diagnostics)
	e.fake.Mutate(e.ref("ws", "", ""), func(obj map[string]any) {
		obj["spec"].(map[string]any)["timeoutSeconds"] = 30
	})
	r := directResource(e)
	resp := resource.ReadResponse{State: created.State, Private: created.Private}
	// When refreshing a resource with an existing generation baseline.
	r.Read(t.Context(), resource.ReadRequest{State: created.State, Private: created.Private}, &resp)
	// Then the unconfigured default stays null and the baseline stays at Create's value.
	assertDiagnostics(t, resp.Diagnostics)

	var attrs map[string]tftypes.Value
	if err := resp.State.Raw.As(&attrs); err != nil {
		t.Fatal(err)
	}

	if !attrs["timeout_seconds"].IsNull() {
		t.Fatalf("timeout_seconds = %s", attrs["timeout_seconds"])
	}

	b, d := resp.Private.GetKey(t.Context(), "generation")
	assertDiagnostics(t, d)

	if string(b) != "1" {
		t.Fatalf("generation = %s", b)
	}
}
