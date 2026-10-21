// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration

import (
	"context"
	"encoding/json"
	"maps"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const managedParamKeysKey = "managed_param_keys"

// The framework's concrete private-state type is internal to its module.
type paramKeyState interface {
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
	SetKey(context.Context, string, []byte) diag.Diagnostics
}

func paramKeys(params types.Map) []string {
	return slices.Sorted(maps.Keys(params.Elements()))
}

func managedParamKeys(ctx context.Context, private paramKeyState, prior types.Map) ([]string, diag.Diagnostics) {
	data, diags := private.GetKey(ctx, managedParamKeysKey)
	if diags.HasError() {
		return nil, diags
	}

	var keys []string
	if len(data) > 0 {
		if err := json.Unmarshal(data, &keys); err != nil {
			diags.AddError("Unable to read parameter ownership", err.Error())
			return nil, diags
		}
	}

	// Seed older state from its prior projection, never from remote parameters.
	keys = append(keys, paramKeys(prior)...)
	slices.Sort(keys)

	return slices.Compact(keys), diags
}

func storeParamKeys(ctx context.Context, private paramKeyState, keys []string) diag.Diagnostics {
	data, err := json.Marshal(keys)
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Unable to save parameter ownership", err.Error())}
	}

	return private.SetKey(ctx, managedParamKeysKey, data)
}
