// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/featureintegration"
)

func TestValidate_whenParamElementNull(t *testing.T) {
	for _, tc := range []struct {
		name    string
		value   types.Map
		wantErr bool
	}{
		{
			name: "null_element", wantErr: true,
			value: types.MapValueMust(jsontypes.NormalizedType{}, map[string]attr.Value{
				"nullable": jsontypes.NewNormalizedNull(),
			}),
		},
		{
			name: "unknown_element",
			value: types.MapValueMust(jsontypes.NormalizedType{}, map[string]attr.Value{
				"nullable": jsontypes.NewNormalizedUnknown(),
			}),
		},
		{
			name: "json_null",
			value: types.MapValueMust(jsontypes.NormalizedType{}, map[string]attr.Value{
				"nullable": jsontypes.NewNormalizedValue("null"),
			}),
		},
		{name: "null_map", value: types.MapNull(jsontypes.NormalizedType{})},
		{name: "unknown_map", value: types.MapUnknown(jsontypes.NormalizedType{})},
		{name: "empty_map", value: types.MapValueMust(jsontypes.NormalizedType{}, map[string]attr.Value{})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given the validators registered on the params map.
			r := featureintegration.NewResource()

			var schemaResp resource.SchemaResponse
			r.Schema(t.Context(), resource.SchemaRequest{}, &schemaResp)
			params := schemaResp.Schema.Attributes["params"].(schema.MapAttribute)

			var resp validator.MapResponse
			// When validating a configured map value.
			for _, v := range params.Validators {
				v.ValidateMap(t.Context(), validator.MapRequest{
					Path: path.Root("params"), ConfigValue: tc.value,
				}, &resp)
			}
			// Then only a Terraform null element is rejected, at its map-key path.
			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Fatalf("diagnostics = %v, want error = %t", resp.Diagnostics, tc.wantErr)
			}

			if tc.wantErr {
				errs := resp.Diagnostics.Errors()
				if len(errs) != 1 {
					t.Fatalf("diagnostics = %v, want one error", resp.Diagnostics)
				}

				got, ok := errs[0].(diag.DiagnosticWithPath)
				if !ok || !got.Path().Equal(path.Root("params").AtMapKey("nullable")) {
					t.Fatalf("diagnostic = %v, want params[\"nullable\"]", errs[0])
				}
			}
		})
	}
}
