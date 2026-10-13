package engine

import (
	"context"

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
	for name, value := range map[string]string{"id": i.ID(), "name": i.Name, "workspace": i.Workspace, "environment": i.Environment, "version": i.Version} {
		if value != "" {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(name), value)...)
		}
	}
}
