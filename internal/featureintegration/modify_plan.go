// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *featureResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var configured, prior types.Map
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("params"), &configured)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("params"), &prior)...)

	if resp.Diagnostics.HasError() || configured.IsUnknown() {
		return
	}

	keys, diags := managedParamKeys(ctx, req.Private, prior)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !slices.Equal(paramKeys(configured), keys) {
		// Private changes alone cannot trigger Update. Recompute type on apply
		// without marking enabled unknown, which would request an enable POST.
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("type"), types.StringUnknown())...)
	}
}
