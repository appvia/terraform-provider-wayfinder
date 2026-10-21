// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"errors"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func (r *kindResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	i, err := convert.ParseImportID(r.kind, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	if hint := notFoundHint(r.kind, i); hint != "" {
		if !r.prepared(&resp.Diagnostics) {
			return
		}

		if _, err := r.client.Get(ctx, reference(r.kind, i)); err != nil {
			if errors.Is(err, client.ErrNotFound) {
				resp.Diagnostics.AddError("Cannot import non-existent integration", err.Error()+"\n\n"+hint)
				return
			}

			r.apiDiagnostics(&resp.Diagnostics, err, i.ID())

			return
		}
	}

	for name, value := range map[string]string{"id": i.ID(), "name": i.Name, "workspace": i.Workspace, "environment": i.Environment, "version": i.Version} {
		if value != "" {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(name), value)...)
		}
	}
}
