package manifestresource

import (
	"context"
	"errors"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *manifestResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var text types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("manifest"), &text)...)
	if resp.Diagnostics.HasError() || text.IsNull() || text.IsUnknown() || !r.ready(&resp.Diagnostics, false) {
		return
	}
	r.desiredFrom(text, &resp.Diagnostics)
}

// desiredFrom parses a known manifest and rejects secrets, reporting problems on manifest.
func (r *manifestResource) desiredFrom(text types.String, d *diag.Diagnostics) (desired, bool) {
	want, err := parseManifest(r.manifest, text.ValueString())
	if err != nil {
		summary := "Invalid manifest"
		var ke *kindError
		if errors.As(err, &ke) {
			summary = "Unsupported kind"
		}
		d.AddAttributeError(path.Root("manifest"), summary, err.Error())
		return desired{}, false
	}
	secrets := secretValues(want.kind, want.object)
	for _, s := range secrets {
		d.AddAttributeError(path.Root("manifest"), "Secret in manifest", secretMessage(want.kind, s))
	}
	return want, len(secrets) == 0
}

func (r *manifestResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || !r.ready(&resp.Diagnostics, false) {
		return
	}
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Status = types.ObjectUnknown(statusTypes)
	if plan.Manifest.IsUnknown() {
		plan.ID, plan.ObjectJSON = types.StringUnknown(), jsontypes.NewNormalizedUnknown()
		resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
		return
	}
	want, ok := r.desiredFrom(plan.Manifest, &resp.Diagnostics)
	if !ok {
		return
	}
	js, err := encodeJSON(want.object)
	if err != nil {
		resp.Diagnostics.AddError("Invalid manifest", err.Error())
		return
	}
	plan.ID, plan.ObjectJSON = types.StringValue(want.importID()), jsontypes.NewNormalizedValue(js)
	if !req.State.Raw.IsNull() {
		var state model
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		prior, err := decodeJSON(state.ObjectJSON.ValueString())
		switch {
		case state.ID.ValueString() != want.importID():
			resp.RequiresReplace = append(resp.RequiresReplace, path.Root("manifest"))
		case err == nil && reflect.DeepEqual(prior, want.object):
			plan.ObjectJSON, plan.Status = state.ObjectJSON, state.Status
		case err == nil && matches(prior, want.object):
			plan.Status = state.Status
		}
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
}
