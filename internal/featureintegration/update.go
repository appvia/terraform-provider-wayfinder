// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration

import (
	"context"

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
	removed := make([]string, 0, len(priorParams))

	for key := range priorParams {
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

	if needsEnable && !r.enablePlanned(ctx, &plan, &resp.Diagnostics) {
		return
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
}
