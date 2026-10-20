// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package manifestresource

import (
	"context"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/engine"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const defaultTimeout = 30 * time.Minute

func (r *manifestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.ready(&resp.Diagnostics, true) {
		return
	}

	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	want, ok := r.desiredFrom(plan.Manifest, &resp.Diagnostics)
	if !ok {
		return
	}

	duration, d := plan.Timeouts.Create(ctx, defaultTimeout)
	resp.Diagnostics.Append(d...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	js, err := encodeJSON(want.object)
	if err != nil {
		resp.Diagnostics.AddError("Invalid manifest", err.Error())
		return
	}

	wo, _, ok := r.writeOnlyFrom(ctx, req.Config, want, &resp.Diagnostics)
	if !ok {
		return
	}

	send := want.object

	if wo != nil {
		var problems []woProblem

		send, problems = mergeWriteOnly(want.kind, want.object, wo)
		addProblems(&resp.Diagnostics, problems)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	ref := engine.Reference(want.kind, want.identity)

	obj, err := r.client.Create(ctx, ref, send)
	if err != nil {
		apiError(&resp.Diagnostics, err, errorTarget(want.kind, want.identity), wo)
		return
	}

	setGeneration(ctx, resp.Private, obj, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(want.importID())
	if plan.ObjectJSON.IsUnknown() {
		plan.ObjectJSON = jsontypes.NewNormalizedValue(js)
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
