// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package manifestresource

import (
	"context"
	"errors"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// writeOnlyFrom reads apply-time write-only values from configuration, never plan or state.
func (r *manifestResource) writeOnlyFrom(ctx context.Context, config tfsdk.Config, want desired, d *diag.Diagnostics) (wo map[string]any, versions map[string]string, ok bool) {
	var (
		text       types.String
		versionMap types.Map
	)

	d.Append(config.GetAttribute(ctx, path.Root("manifest_wo"), &text)...)
	d.Append(config.GetAttribute(ctx, path.Root("manifest_wo_version"), &versionMap)...)

	if d.HasError() {
		return nil, nil, false
	}

	versions, _ = versionsOf(versionMap)
	addProblems(d, versionsWithoutWO(text.IsNull(), versions, !versionMap.IsNull()))

	if !text.IsNull() {
		var problems []woProblem

		wo, problems = validateWriteOnly(&want, text.ValueString(), versions, !versionMap.IsNull())
		addProblems(d, problems)
	}

	return wo, versions, !d.HasError()
}

func (r *manifestResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var (
		text, wo   types.String
		versionMap types.Map
	)

	d := &resp.Diagnostics
	d.Append(req.Config.GetAttribute(ctx, path.Root("manifest"), &text)...)
	d.Append(req.Config.GetAttribute(ctx, path.Root("manifest_wo"), &wo)...)
	d.Append(req.Config.GetAttribute(ctx, path.Root("manifest_wo_version"), &versionMap)...)

	if d.HasError() {
		return
	}

	versions, versionsKnown := versionsOf(versionMap)
	if versionsKnown && wo.IsNull() {
		addProblems(d, versionsWithoutWO(true, versions, true))
	}

	var want *desired

	if !text.IsNull() && !text.IsUnknown() && r.ready(d, false) {
		if w, ok := r.desiredFrom(text, d); ok {
			want = &w
		}
	}

	if !wo.IsNull() && !wo.IsUnknown() {
		_, problems := validateWriteOnly(want, wo.ValueString(), versions, versionsKnown && !versionMap.IsNull())
		addProblems(d, problems)
	}
}

// addProblems reports write-only problems on the attribute each one names.
func addProblems(d *diag.Diagnostics, ps []woProblem) {
	for _, p := range ps {
		d.AddAttributeError(path.Root(p.attr), p.summary, p.detail)
	}
}

// versionsOf returns the manifest_wo_version keys and values: nil, true when null; nil, false
// when unknown; an unknown element value maps to "".
func versionsOf(m types.Map) (map[string]string, bool) {
	if m.IsNull() {
		return nil, true
	}

	if m.IsUnknown() {
		return nil, false
	}

	out := make(map[string]string, len(m.Elements()))
	for k, v := range m.Elements() {
		s, _ := v.(types.String)
		out[k] = s.ValueString()
	}

	return out, true
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

		if !plan.ManifestWOVersion.Equal(state.ManifestWOVersion) {
			plan.Status = types.ObjectUnknown(statusTypes)
		}
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
}
