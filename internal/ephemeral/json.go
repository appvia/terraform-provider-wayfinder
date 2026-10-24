// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package ephemeral

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// responseDynamic converts JSON without losing integer precision. A nil exports
// slice selects the whole response; an empty slice selects an empty object.
func responseDynamic(ctx context.Context, raw json.RawMessage, exports []string) (types.Dynamic, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !json.Valid(raw) {
		diags.AddError("Invalid action response", "Wayfinder did not return a valid JSON value.")
		return types.DynamicNull(), diags
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		diags.AddError("Invalid action response", "Wayfinder did not return a valid JSON value.")
		return types.DynamicNull(), diags
	}

	if exports != nil {
		value = filterResponse(value, exports)
	}

	converted, diags := jsonValue(ctx, value)
	if diags.HasError() {
		return types.DynamicNull(), diags
	}

	if converted.IsNull() {
		return types.DynamicNull(), diags
	}

	return types.DynamicValue(converted), diags
}

// filterResponse projects dot-separated object paths without mutating the source.
// Selecting a parent includes its whole value, regardless of path order.
func filterResponse(value any, paths []string) any {
	if slices.Contains(paths, "") {
		return value
	}

	children := make(map[string][]string)

	for _, path := range paths {
		head, tail, _ := strings.Cut(path, ".")
		children[head] = append(children[head], tail)
	}

	object, _ := value.(map[string]any)
	selected := make(map[string]any, len(children))

	for head, tails := range children {
		selected[head] = filterResponse(object[head], tails)
	}

	return selected
}

// jsonValue only receives the variants produced by a UseNumber JSON decoder.
func jsonValue(ctx context.Context, value any) (attr.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch v := value.(type) {
	case nil:
		return types.DynamicNull(), diags
	case string:
		return types.StringValue(v), diags
	case bool:
		return types.BoolValue(v), diags
	case json.Number:
		number, _, err := big.ParseFloat(string(v), 10, 512, big.ToNearestEven)
		if err != nil {
			diags.AddError("Invalid action response", "Wayfinder returned a number Terraform cannot represent.")
			return nil, diags
		}

		return types.NumberValue(number), diags
	case map[string]any:
		attributeTypes := make(map[string]attr.Type, len(v))
		attributes := make(map[string]attr.Value, len(v))

		for name, child := range v {
			converted, childDiags := jsonValue(ctx, child)
			diags.Append(childDiags...)

			if diags.HasError() {
				return nil, diags
			}

			attributeTypes[name], attributes[name] = converted.Type(ctx), converted
		}

		return types.ObjectValue(attributeTypes, attributes)
	case []any:
		elementTypes := make([]attr.Type, len(v))
		elements := make([]attr.Value, len(v))

		for i, child := range v {
			converted, childDiags := jsonValue(ctx, child)
			diags.Append(childDiags...)

			if diags.HasError() {
				return nil, diags
			}

			elementTypes[i], elements[i] = converted.Type(ctx), converted
		}

		return types.TupleValue(elementTypes, elements)
	default:
		diags.AddError("Invalid action response", "Wayfinder returned an unsupported JSON value.")
		return nil, diags
	}
}
