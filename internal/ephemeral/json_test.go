// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package ephemeral

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestResponseDynamic_whenJSONValue(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want attr.Value
	}{
		{"object", `{"url":"credential"}`, types.ObjectValueMust(map[string]attr.Type{"url": types.StringType}, map[string]attr.Value{"url": types.StringValue("credential")})},
		{"array", `["one",true,null]`, types.TupleValueMust([]attr.Type{types.StringType, types.BoolType, types.DynamicType}, []attr.Value{types.StringValue("one"), types.BoolValue(true), types.DynamicNull()})},
		{"nested", `{"status":{"items":[false]}}`, types.ObjectValueMust(
			map[string]attr.Type{"status": types.ObjectType{AttrTypes: map[string]attr.Type{"items": types.TupleType{ElemTypes: []attr.Type{types.BoolType}}}}},
			map[string]attr.Value{"status": types.ObjectValueMust(map[string]attr.Type{"items": types.TupleType{ElemTypes: []attr.Type{types.BoolType}}},
				map[string]attr.Value{"items": types.TupleValueMust([]attr.Type{types.BoolType}, []attr.Value{types.BoolValue(false)})})},
		)},
		{"large integer", `9007199254740993`, types.NumberValue(new(big.Float).SetInt64(9007199254740993))},
		{"fraction", `1.25`, types.NumberValue(big.NewFloat(1.25))},
		{"exponent", `1e3`, types.NumberValue(big.NewFloat(1000))},
		{"string", `"credential"`, types.StringValue("credential")},
		{"boolean", `false`, types.BoolValue(false)},
		{"null", `null`, types.DynamicNull()},
		{"empty object", `{}`, types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{})},
		{"empty array", `[]`, types.TupleValueMust([]attr.Type{}, []attr.Value{})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a JSON response, when converted without filtering.
			got, diags := responseDynamic(t.Context(), json.RawMessage(tt.raw), nil)
			// Then both its type and value survive conversion.
			if diags.HasError() {
				t.Fatal(diags)
			}

			if tt.want.IsNull() {
				if !got.IsNull() {
					t.Fatalf("expected null, got %s", got)
				}

				return
			}

			if !got.UnderlyingValue().Equal(tt.want) {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestResponseDynamic_whenExportPathsSelected(t *testing.T) {
	tests := []struct {
		name    string
		exports []string
		want    string
	}{
		{"present", []string{"url"}, `{"url":"credential"}`},
		{"missing", []string{"missing"}, `{"missing":null}`},
		{"nested", []string{"status.token"}, `{"status":{"token":"nested-secret"}}`},
		{"nested missing", []string{"absent.token"}, `{"absent":{"token":null}}`},
		{"null intermediate", []string{"nothing.token"}, `{"nothing":{"token":null}}`},
		{"scalar intermediate", []string{"url.token"}, `{"url":{"token":null}}`},
		{"siblings", []string{"status.token", "status.ready"}, `{"status":{"token":"nested-secret","ready":true}}`},
		{"parent first", []string{"status", "status.token"}, `{"status":{"token":"nested-secret","ready":true,"private":"omitted"}}`},
		{"parent last", []string{"status.token", "status"}, `{"status":{"token":"nested-secret","ready":true,"private":"omitted"}}`},
		{"duplicate", []string{"url", "url"}, `{"url":"credential"}`},
		{"empty selection", []string{}, `{}`},
		{"array retained", []string{"items"}, `{"items":[1,"two"]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a response containing both selected and unselected secrets.
			raw := json.RawMessage(`{"url":"credential","status":{"token":"nested-secret","ready":true,"private":"omitted"},"nothing":null,"items":[1,"two"]}`)
			// When exported paths are selected.
			got, diags := responseDynamic(t.Context(), raw, tt.exports)
			// Then only the requested nested tree remains.
			if diags.HasError() {
				t.Fatal(diags)
			}

			want, diags := responseDynamic(t.Context(), json.RawMessage(tt.want), nil)
			if diags.HasError() {
				t.Fatal(diags)
			}

			if !got.Equal(want) {
				t.Fatalf("got %s, want %s", got, want)
			}
		})
	}
}

func TestResponseDynamic_whenResponseMalformed(t *testing.T) {
	for _, raw := range []string{"", "secret-value", `{"token":"secret-value"`, `{} {"token":"secret-value"}`, `1e9999999999`} {
		t.Run(raw, func(t *testing.T) {
			// Given malformed JSON or an unrepresentable number, when converted.
			_, diags := responseDynamic(t.Context(), json.RawMessage(raw), nil)
			// Then a specific, body-free diagnostic is returned.
			assertDiagnostic(t, diags, "Invalid action response", "", "secret-value")
		})
	}
}
