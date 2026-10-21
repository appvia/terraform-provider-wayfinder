// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// reenable preserves enabled features and plans an update for disabled features.
type reenable struct{}

func (reenable) Description(context.Context) string {
	return "Keeps enabled features true; a disabled feature becomes unknown to plan an in-place re-enable."
}

func (m reenable) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (reenable) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	if req.StateValue.ValueBool() {
		resp.PlanValue = types.BoolValue(true)
	} else {
		resp.PlanValue = types.BoolUnknown()
	}
}
