// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUpdate_whenOnlyDisableOnDestroyChanged(t *testing.T) {
	for _, tc := range []struct {
		name  string
		prior types.Map
		plan  types.Map
	}{
		{
			name: "no_params", prior: types.MapNull(jsontypes.NormalizedType{}),
			plan: types.MapNull(jsontypes.NormalizedType{}),
		},
		{
			name: "equivalent_json",
			prior: types.MapValueMust(jsontypes.NormalizedType{}, map[string]attr.Value{
				"options": jsontypes.NewNormalizedValue(`{ "z": true, "a": 1 }`),
			}),
			plan: types.MapValueMust(jsontypes.NormalizedType{}, map[string]attr.Value{
				"options": jsontypes.NewNormalizedValue(`{"a":1,"z":true}`),
			}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given known enabled state, optionally with semantically equal JSON.
			e := newEnv(t)

			r, state := lifecycleResource(t, e, "azure")
			if diags := state.SetAttribute(t.Context(), path.Root("params"), tc.prior); diags.HasError() {
				t.Fatal(diags)
			}

			plan := state
			if diags := plan.SetAttribute(t.Context(), path.Root("params"), tc.plan); diags.HasError() {
				t.Fatal(diags)
			}

			if diags := plan.SetAttribute(t.Context(), path.Root("keep_enabled_on_destroy"), types.BoolValue(true)); diags.HasError() {
				t.Fatal(diags)
			}

			resp := resource.UpdateResponse{State: state}
			// When applying only the retention flag change.
			r.Update(t.Context(), resource.UpdateRequest{
				State: state, Plan: tfsdk.Plan(plan),
				Config: tfsdk.Config(plan),
			}, &resp)
			// Then the new value is saved without even a GET.
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}

			var keep types.Bool
			if diags := resp.State.GetAttribute(t.Context(), path.Root("keep_enabled_on_destroy"), &keep); diags.HasError() {
				t.Fatal(diags)
			}

			if !keep.ValueBool() || e.fake.CountRequests("", "") != 0 {
				t.Fatalf("retention = %v, requests = %+v", keep, e.fake.Requests())
			}
		})
	}
}
