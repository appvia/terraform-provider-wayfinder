// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package manifestresource

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/engine"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Update follows D-CRUD: fresh GET, generation compare against the private baseline, then PUT
// of the fresh object with the manifest overlaid. Nothing is sent when the fresh object
// already matches the manifest (for example when only wait_for_ready or timeouts changed).
func (r *manifestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.ready(&resp.Diagnostics, true) {
		return
	}
	var plan, state model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	want, ok := r.desiredFrom(plan.Manifest, &resp.Diagnostics)
	if !ok {
		return
	}
	if want.importID() != state.ID.ValueString() {
		resp.Diagnostics.AddError("Object identity changed", fmt.Sprintf("The manifest now identifies %s instead of %s. Terraform could not plan the replacement because manifest was unknown at plan time; apply again.", want.importID(), state.ID.ValueString()))
		return
	}
	duration, d := plan.Timeouts.Update(ctx, defaultTimeout)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	ref, target := engine.Reference(want.kind, want.identity), errorTarget(want.kind, want.identity)
	fresh, err := r.client.Get(ctx, ref)
	if err != nil {
		apiError(&resp.Diagnostics, err, target, nil)
		return
	}
	generation, err := engine.Generation(fresh)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Wayfinder metadata", err.Error())
		return
	}
	baseline, d := req.Private.GetKey(ctx, generationKey)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if string(baseline) != strconv.FormatInt(generation, 10) {
		engine.AddAPIError(&resp.Diagnostics, client.ErrModified, target)
		return
	}
	js, err := encodeJSON(want.object)
	if err != nil {
		resp.Diagnostics.AddError("Invalid manifest", err.Error())
		return
	}
	if plan.ObjectJSON.IsUnknown() {
		plan.ObjectJSON = jsontypes.NewNormalizedValue(js)
	}
	wo, planned, ok := r.writeOnlyFrom(ctx, req.Config, want, &resp.Diagnostics)
	if !ok {
		return
	}
	stored, _ := versionsOf(state.ManifestWOVersion)
	selected, paths := selectWriteOnly(wo, planned, stored)
	var merged map[string]any
	send := want.object
	if selected != nil {
		var problems []woProblem
		merged, problems = mergeWriteOnly(want.kind, want.object, selected)
		addProblems(&resp.Diagnostics, problems)
		if resp.Diagnostics.HasError() {
			return
		}
		send = merged
	}
	var prior map[string]any
	if !state.Manifest.IsNull() {
		if p, err := parseManifest(r.manifest, state.Manifest.ValueString()); err == nil {
			prior = p.object
		}
	}
	if wo != nil {
		prior = withoutWriteOnly(prior, wo)
	}
	candidate := overlay(fresh, send, prior)
	if len(paths) > 0 {
		candidate = deepCopy(candidate).(map[string]any)
		for _, p := range paths {
			if v, ok := lookup(merged, p); ok {
				setAt(candidate, p, deepCopy(v))
			}
		}
	}
	if reflect.DeepEqual(normalized(candidate), normalized(fresh)) {
		if plan.Status.IsUnknown() {
			plan.Status = statusValue(fresh)
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		return
	}
	obj, err := r.client.Update(ctx, ref, candidate)
	if err != nil {
		apiError(&resp.Diagnostics, err, target, wo)
		return
	}
	setGeneration(ctx, resp.Private, obj, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Status = statusValue(obj)
	if shouldWait(want.kind, plan) {
		ready, d := r.waiter(want.kind, want.identity).Ready(ctx, ref)
		resp.Diagnostics.Append(d...)
		if ready != nil {
			plan.Status = statusValue(ready)
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}
