// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package webhookrotation

import (
	"context"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ImportState adopts a webhook without rotating it; the following Read checks it exists.
func (r *rotationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) > 2 || slices.ContainsFunc(parts, invalidSegment) {
		resp.Diagnostics.AddError("Invalid import ID",
			`Expected "<webhook>" or "<workspace>/<webhook>", where each part is a non-empty name other than . or .. and without \.`)

		return
	}

	webhook := types.StringValue(parts[len(parts)-1])
	workspace := types.StringNull()

	if len(parts) == 2 {
		workspace = types.StringValue(parts[0])
	}

	m := rotationModel{
		ID: types.StringValue(req.ID), Name: webhook, Webhook: webhook, Workspace: workspace,
		Triggers:   types.MapNull(types.StringType),
		RotationID: types.StringNull(), RotatedAt: types.StringNull(),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
