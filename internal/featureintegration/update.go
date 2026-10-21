// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func (r *featureResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state featureModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	plannedParams, priorParams := plan.Params.Elements(), state.Params.Elements()
	plannedKeys := paramKeys(plan.Params)
	keys, diags := managedParamKeys(ctx, req.Private, state.Params)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	removed := make([]string, 0, len(keys))

	for _, key := range keys {
		if _, ok := plannedParams[key]; !ok {
			removed = append(removed, key)
		}
	}

	needsEnable := plan.Enabled.IsUnknown()

	for key, value := range plannedParams {
		prior, ok := priorParams[key]
		if !ok {
			needsEnable = true
			continue
		}

		equal, diags := value.(jsontypes.Normalized).StringSemanticEquals(ctx, prior.(jsontypes.Normalized))
		resp.Diagnostics.Append(diags...)

		if !equal {
			needsEnable = true
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if needsEnable {
		obj := r.enablePlanned(ctx, &plan, &resp.Diagnostics)
		if obj == nil {
			return
		}

		keys = append(keys, plannedKeys...)
		slices.Sort(keys)
		keys = slices.Compact(keys)

		// Save the observed write before removal can fail; old keys remain owned
		// until the removal PUT succeeds.
		partial := plan
		params, diags := projectParams(plan.Params, obj, keys)
		resp.Diagnostics.Append(diags...)

		partial.Params = params
		resp.Diagnostics.Append(resp.State.Set(ctx, &partial)...)
		resp.Diagnostics.Append(storeParamKeys(ctx, resp.Private, keys)...)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	if len(removed) > 0 {
		ref := integrationRef(plan.Name.ValueString())

		obj, err := r.client.Get(ctx, ref)
		if err != nil {
			addAPIError(&resp.Diagnostics, err, plan.Name.ValueString())
			return
		}

		spec := obj["spec"].(map[string]any)
		params, _ := spec["params"].(map[string]any)

		for _, key := range removed {
			delete(params, key)
		}

		if len(params) == 0 {
			delete(spec, "params")
		}

		obj, err = r.client.Update(ctx, ref, obj)
		if err != nil {
			addAPIError(&resp.Diagnostics, err, plan.Name.ValueString())
			return
		}

		plan.setObject(obj)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(storeParamKeys(ctx, resp.Private, plannedKeys)...)
}
