// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"strconv"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func (r *kindResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.prepared(&resp.Diagnostics) {
		return
	}
	i, err := identityOf(r.kind, req.Plan.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource identity", err.Error())
		return
	}
	duration, d := r.operationTimeout(ctx, req.Plan.Raw, "create")
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	include := map[string]bool{}
	v, err := writeValues(secretInputs{config: req.Config.Raw, plan: req.Plan.Raw, creating: true, include: include})
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource configuration", err.Error())
		return
	}
	obj, err := convert.ToObject(r.mapping, v, convert.ToOpts{IncludeWriteOnly: include})
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource configuration", err.Error())
		return
	}
	ref := reference(r.kind, i)
	obj, err = r.client.Create(ctx, ref, obj)
	if err != nil {
		r.apiDiagnostics(&resp.Diagnostics, err, i.ID())
		return
	}
	s, err := inspectObject(obj)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Wayfinder metadata", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, "generation", []byte(strconv.FormatInt(s.Metadata.Generation, 10)))...)
	state, err := convert.FromObject(r.mapping, obj, req.Plan.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read created resource", err.Error())
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
