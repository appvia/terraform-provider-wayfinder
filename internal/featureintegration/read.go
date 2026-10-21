// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
)

func (r *featureResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m featureModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)

	if resp.Diagnostics.HasError() {
		return
	}

	obj, err := r.client.Get(ctx, integrationRef(m.Name.ValueString()))
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		addAPIError(&resp.Diagnostics, err, m.Name.ValueString())
		return
	}

	m.setObject(obj)
	params, diags := projectParams(m.Params, obj)
	resp.Diagnostics.Append(diags...)

	m.Params = params
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *featureResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	name := strings.TrimPrefix(req.ID, "wayfinder.")
	if name == "" || strings.Contains(name, "/") {
		resp.Diagnostics.AddError("Invalid import ID", `Expected a bare feature name such as "azure" or an object name such as "wayfinder.azure".`)
		return
	}

	obj, err := r.client.Get(ctx, integrationRef(name))
	spec, _ := obj["spec"].(map[string]any)

	if errors.Is(err, client.ErrNotFound) || (err == nil && spec["enabled"] != true) {
		resp.Diagnostics.AddError("Feature integration not enabled",
			fmt.Sprintf("Feature integration %q is not enabled in this tenant, so there is nothing to import. Add the resource to your configuration and apply to enable it.", name))

		return
	}

	if err != nil {
		addAPIError(&resp.Diagnostics, err, name)
		return
	}

	m := featureModel{
		Name: types.StringValue(name), KeepEnabledOnDestroy: types.BoolValue(false),
		Params: types.MapNull(jsontypes.NormalizedType{}),
	}
	m.setObject(obj)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
