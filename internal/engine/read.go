// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"errors"
	"strconv"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func (r *kindResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.prepared(&resp.Diagnostics) {
		return
	}

	i, err := identityOf(r.kind, req.State.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource identity", err.Error())
		return
	}

	obj, err := r.client.Get(ctx, reference(r.kind, i))
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		r.apiDiagnostics(&resp.Diagnostics, err, i.ID())
		return
	}

	baseline, d := req.Private.GetKey(ctx, "generation")
	resp.Diagnostics.Append(d...)

	if resp.Diagnostics.HasError() {
		return
	}

	state, err := convert.FromObjectOpts(r.mapping, obj, req.State.Raw, convert.FromOpts{PopulateAll: len(baseline) == 0})
	if err != nil {
		resp.Diagnostics.AddError("Unable to read resource", err.Error())
		return
	}

	resp.State.Raw = state

	if len(baseline) != 0 {
		return
	}

	s, err := inspectObject(obj)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Wayfinder metadata", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.Private.SetKey(ctx, "generation", []byte(strconv.FormatInt(s.Metadata.Generation, 10)))...)
}
