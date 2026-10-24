// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package ephemeral

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

var (
	_ ephemeral.EphemeralResourceWithConfigure      = (*resourceAction)(nil)
	_ ephemeral.EphemeralResourceWithValidateConfig = (*resourceAction)(nil)
)

// NewResourceAction returns the GET-only wayfinder_resource_action ephemeral resource.
func NewResourceAction() ephemeral.EphemeralResource { return &resourceAction{} }

type resourceAction struct{ client *client.Client }

type resourceActionModel struct {
	Type                 types.String  `tfsdk:"type"`
	Name                 types.String  `tfsdk:"name"`
	Workspace            types.String  `tfsdk:"workspace"`
	Environment          types.String  `tfsdk:"environment"`
	Version              types.String  `tfsdk:"version"`
	Action               types.String  `tfsdk:"action"`
	ResponseExportValues types.List    `tfsdk:"response_export_values"`
	Output               types.Dynamic `tfsdk:"output"`
}

func (r *resourceAction) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_action"
}

func (r *resourceAction) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Wayfinder object sub-resource using GET on every plan and apply. " +
			"For a webhook's ingest URL use `wayfinder_webhook_ingest_url`. The sensitive output is ephemeral " +
			"and is never stored in state or plan files. Requires Terraform/OpenTofu >= 1.11.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Required: true, MarkdownDescription: "Object type in `group/version/Kind` format, for example `integrations/v3/Webhook`.",
			},
			"name": schema.StringAttribute{
				Required: true, MarkdownDescription: "Object name at the selected scope.",
			},
			"workspace": schema.StringAttribute{
				Optional: true, MarkdownDescription: "Workspace scope. Omit for tenant scope.",
			},
			"environment": schema.StringAttribute{
				Optional: true, MarkdownDescription: "Environment scope; requires `workspace`. The type must support the selected scope.",
			},
			"version": schema.StringAttribute{
				Optional: true, MarkdownDescription: "Object version, required for versioned types and forbidden for other types.",
			},
			"action": schema.StringAttribute{
				Required: true, MarkdownDescription: "Sub-resource segment matching `^[a-z][a-z0-9-]*$`. Always read using GET.",
			},
			"response_export_values": schema.ListAttribute{
				Optional: true, ElementType: types.StringType,
				MarkdownDescription: "Dot-separated object paths to export, preserving nested structure. Missing paths have null leaves. Omit to return the full JSON response.",
			},
			"output": schema.DynamicAttribute{
				Computed: true, Sensitive: true, MarkdownDescription: "Sensitive JSON response, optionally restricted by `response_export_values`.",
			},
		},
	}
}

func (r *resourceAction) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}

	r.client = c
}

func (r *resourceAction) ValidateConfig(ctx context.Context, req ephemeral.ValidateConfigRequest, resp *ephemeral.ValidateConfigResponse) {
	var m resourceActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, diags := m.validate(ctx)
	resp.Diagnostics.Append(diags...)
}

func (m resourceActionModel) validate(ctx context.Context) (manifest.Kind, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !m.Action.IsUnknown() && !actionPattern.MatchString(m.Action.ValueString()) {
		diags.AddAttributeError(path.Root("action"), "Invalid action", "Action must match ^[a-z][a-z0-9-]*$.")
	}

	for _, input := range []struct {
		name  string
		value types.String
	}{{"name", m.Name}, {"workspace", m.Workspace}, {"environment", m.Environment}, {"version", m.Version}} {
		if !input.value.IsUnknown() && (invalidSegment(input.value.ValueString()) || (!input.value.IsNull() && input.value.ValueString() == "")) {
			diags.AddAttributeError(path.Root(input.name), "Invalid resource reference", input.name+" must be a non-empty path segment, not . or .. .")
		}
	}

	if !m.Environment.IsUnknown() && m.Environment.ValueString() != "" && !m.Workspace.IsUnknown() && m.Workspace.ValueString() == "" {
		diags.AddAttributeError(path.Root("environment"), "Environment requires workspace", "Set workspace when selecting an environment.")
	}

	var exports []types.String
	if !m.ResponseExportValues.IsUnknown() {
		diags.Append(m.ResponseExportValues.ElementsAs(ctx, &exports, false)...)
	}

	for i, export := range exports {
		if !export.IsUnknown() && slices.Contains(strings.Split(export.ValueString(), "."), "") {
			diags.AddAttributeError(path.Root("response_export_values").AtListIndex(i), "Invalid response export path", "Use non-empty, dot-separated object keys.")
		}
	}

	if m.Type.IsUnknown() {
		return manifest.Kind{}, diags
	}

	parts := strings.Split(m.Type.ValueString(), "/")
	if len(parts) != 3 || slices.Contains(parts, "") {
		diags.AddAttributeError(path.Root("type"), "Invalid resource type", "Expected group/version/Kind, for example integrations/v3/Webhook.")
		return manifest.Kind{}, diags
	}

	embedded, err := manifest.LoadEmbedded()
	if err != nil {
		diags.AddError("Unable to load Wayfinder manifest", err.Error())
		return manifest.Kind{}, diags
	}

	k, ok := embedded.FindKind(parts[0], parts[2])
	if !ok || k.Version != parts[1] {
		diags.AddAttributeError(path.Root("type"), "Unknown resource type", "Expected a type from the embedded Wayfinder manifest in group/version/Kind format, for example integrations/v3/Webhook.")
		return manifest.Kind{}, diags
	}

	if !m.Workspace.IsUnknown() && !m.Environment.IsUnknown() {
		scope := "tenant"
		if m.Workspace.ValueString() != "" {
			scope = "workspace"
		}

		if m.Environment.ValueString() != "" {
			scope = "environment"
		}

		if !slices.Contains(k.Scopes, scope) {
			diags.AddError("Unsupported resource scope", fmt.Sprintf("%s does not support %s scope. Supported scopes: %s.", k.Kind, scope, strings.Join(k.Scopes, ", ")))
		}
	}

	if !m.Version.IsUnknown() {
		switch {
		case k.Versioned && m.Version.ValueString() == "":
			diags.AddAttributeError(path.Root("version"), "Object version required", k.Kind+" is versioned; set version.")
		case !k.Versioned && !m.Version.IsNull():
			diags.AddAttributeError(path.Root("version"), "Object version not supported", k.Kind+" is not versioned; omit version.")
		}
	}

	return *k, diags
}

func (m resourceActionModel) hasUnknownInputs() bool {
	for _, value := range []attr.Value{m.Type, m.Name, m.Workspace, m.Environment, m.Version, m.Action, m.ResponseExportValues} {
		if value.IsUnknown() {
			return true
		}
	}

	for _, value := range m.ResponseExportValues.Elements() {
		if value.IsUnknown() {
			return true
		}
	}

	return false
}

func (r *resourceAction) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var m resourceActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)

	if resp.Diagnostics.HasError() {
		return
	}

	k, diags := m.validate(ctx)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	m.Output = types.DynamicUnknown()
	// v1.19.0 prepopulates Result from Config and only permits Deferred when
	// DeferralAllowed. Keep output unknown if a caller supplies unknown inputs.
	if m.hasUnknownInputs() || r.client == nil {
		if req.ClientCapabilities.DeferralAllowed {
			reason := ephemeral.DeferredReasonEphemeralResourceConfigUnknown
			if r.client == nil {
				reason = ephemeral.DeferredReasonProviderConfigUnknown
			}

			resp.Deferred = &ephemeral.Deferred{Reason: reason}
		} else if r.client == nil && !m.hasUnknownInputs() {
			resp.Diagnostics.AddError("Wayfinder client not configured", "Configure the Wayfinder provider before reading an action.")
			return
		}

		resp.Diagnostics.Append(resp.Result.Set(ctx, &m)...)

		return
	}

	ref := client.Ref{
		Group: k.Group, Version: k.Version, APIPath: k.APIPath, Versioned: k.Versioned,
		Name: m.Name.ValueString(), Workspace: m.Workspace.ValueString(),
		Environment: m.Environment.ValueString(), ObjectVersion: m.Version.ValueString(),
	}
	raw, diags := getAction(ctx, r.client, k, ref, m.Action.ValueString())
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var exports []string
	if !m.ResponseExportValues.IsNull() {
		resp.Diagnostics.Append(m.ResponseExportValues.ElementsAs(ctx, &exports, false)...)

		if exports == nil {
			exports = []string{}
		}
	}

	m.Output, diags = responseDynamic(ctx, raw, exports)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &m)...)
}
