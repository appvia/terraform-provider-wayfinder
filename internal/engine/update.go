// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"strconv"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func (r *kindResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.prepared(&resp.Diagnostics) {
		return
	}

	i, err := identityOf(r.kind, req.Plan.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource identity", err.Error())
		return
	}

	duration, d := r.operationTimeout(ctx, req.Plan.Raw, "update")
	resp.Diagnostics.Append(d...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	ref := reference(r.kind, i)

	fresh, err := r.client.Get(ctx, ref)
	if err != nil {
		r.apiDiagnostics(&resp.Diagnostics, err, i.ID())
		return
	}

	s, err := inspectObject(fresh)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Wayfinder metadata", err.Error())
		return
	}

	baseline, d := req.Private.GetKey(ctx, "generation")
	resp.Diagnostics.Append(d...)

	if resp.Diagnostics.HasError() {
		return
	}

	if string(baseline) != strconv.FormatInt(s.Metadata.Generation, 10) {
		r.apiDiagnostics(&resp.Diagnostics, client.ErrModified, i.ID())
		return
	}

	include := map[string]bool{}

	v, err := writeValues(secretInputs{config: req.Config.Raw, plan: req.Plan.Raw, state: req.State.Raw, include: include})
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource configuration", err.Error())
		return
	}

	obj, err := convert.ToObject(r.mapping, v, convert.ToOpts{Base: fresh, Prior: req.State.Raw, IncludeWriteOnly: include})
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource configuration", err.Error())
		return
	}

	obj, err = r.client.Update(ctx, ref, obj)
	if err != nil {
		r.apiDiagnostics(&resp.Diagnostics, err, i.ID())
		return
	}

	s, err = inspectObject(obj)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Wayfinder metadata", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.Private.SetKey(ctx, "generation", []byte(strconv.FormatInt(s.Metadata.Generation, 10)))...)

	state, err := convert.FromObject(r.mapping, obj, req.Plan.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read updated resource", err.Error())
		return
	}

	resp.State.Raw = state
	if r.shouldWait(req.Plan.Raw) {
		obj, d = r.wait(ctx, ref, false)
		resp.Diagnostics.Append(d...)

		if obj != nil {
			state, err = convert.FromObject(r.mapping, obj, req.Plan.Raw)
			if err != nil {
				resp.Diagnostics.AddError("Unable to read resource status", err.Error())
				return
			}

			resp.State.Raw = state
		}
	}
}
