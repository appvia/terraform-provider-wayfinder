// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func (r *featureResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m featureModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)

	if resp.Diagnostics.HasError() {
		return
	}

	keys := paramKeys(m.Params)
	if r.enablePlanned(ctx, &m, &resp.Diagnostics) != nil {
		// Track a successful enable even when a coercion diagnostic taints this resource.
		resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
		resp.Diagnostics.Append(storeParamKeys(ctx, resp.Private, keys)...)
	}
}

// enablePlanned returns the enabled object, including when Wayfinder coerced
// a parameter and the returned state must be saved alongside an error.
func (r *featureResource) enablePlanned(ctx context.Context, m *featureModel, d *diag.Diagnostics) map[string]any {
	params := make(map[string]json.RawMessage, len(m.Params.Elements()))
	for key, value := range m.Params.Elements() {
		params[key] = json.RawMessage(value.(jsontypes.Normalized).ValueString())
	}

	obj, err := enable(ctx, r.client, m.Name.ValueString(), params)
	if err != nil {
		addAPIError(d, err, m.Name.ValueString())
		return nil
	}

	m.setObject(obj)
	projected, diags := projectParams(m.Params, obj, paramKeys(m.Params))
	d.Append(diags...)

	mismatch := false

	for key, value := range projected.Elements() {
		got := value.(jsontypes.Normalized).ValueString()
		want := params[key]

		var actual, planned any
		if err := json.Unmarshal([]byte(got), &actual); err != nil {
			d.AddAttributeError(path.Root("params").AtMapKey(key), "Unable to decode stored parameter", err.Error())
			continue
		}

		if err := json.Unmarshal(want, &planned); err != nil {
			d.AddAttributeError(path.Root("params").AtMapKey(key), "Unable to decode planned parameter", err.Error())
			continue
		}

		if !reflect.DeepEqual(actual, planned) {
			mismatch = true

			d.AddAttributeError(path.Root("params").AtMapKey(key), "Wayfinder stored a different value",
				fmt.Sprintf("Wayfinder stored %s for %q, so the configured %s would never match. Set params[%q] = jsonencode(%s).",
					got, key, want, key, got))
		}
	}

	if mismatch {
		m.Params = projected
	}

	return obj
}
