// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package manifestresource

import (
	"context"
	"errors"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/engine"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func (r *manifestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.ready(&resp.Diagnostics, true) {
		return
	}
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k, i, err := parseID(r.manifest, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("id"), "Invalid resource identity", err.Error())
		return
	}
	obj, err := r.client.Get(ctx, engine.Reference(k, i))
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		apiError(&resp.Diagnostics, err, errorTarget(k, i), nil)
		return
	}
	baseline, d := req.Private.GetKey(ctx, generationKey)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	view := importedView(obj)
	if len(baseline) != 0 && !state.Manifest.IsNull() && !state.Manifest.IsUnknown() {
		if want, err := parseManifest(r.manifest, state.Manifest.ValueString()); err == nil {
			view = projectObject(obj, want.object)
		}
	}
	js, err := encodeJSON(view)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read resource", err.Error())
		return
	}
	state.ObjectJSON, state.Status = jsontypes.NewNormalizedValue(js), statusValue(obj)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	if resp.Diagnostics.HasError() || len(baseline) != 0 {
		return
	}
	setGeneration(ctx, resp.Private, obj, &resp.Diagnostics)
}
