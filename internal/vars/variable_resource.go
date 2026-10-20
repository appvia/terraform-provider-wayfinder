// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package vars

import (
	"context"
	"encoding/json"
	"errors"
	"maps"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
)

var (
	_ resource.ResourceWithConfigure   = (*variableResource)(nil)
	_ resource.ResourceWithImportState = (*variableResource)(nil)
)

// NewVariableResource returns the wayfinder_variable resource.
func NewVariableResource() resource.Resource { return &variableResource{} }

type variableResource struct{ client *client.Client }

type variableModel struct {
	ID          types.String         `tfsdk:"id"`
	Key         types.String         `tfsdk:"key"`
	Workspace   types.String         `tfsdk:"workspace"`
	Environment types.String         `tfsdk:"environment"`
	Instance    types.String         `tfsdk:"instance"`
	Value       jsontypes.Normalized `tfsdk:"value"`
}

func (m variableModel) scoped() scopedKey {
	return keyOf(m.Key, m.Workspace, m.Environment, m.Instance)
}

func (r *variableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_variable"
}

const variableValueDescription = "Value of the variable as a JSON document; any JSON value is allowed. " +
	"Use `jsonencode`, for example `value = jsonencode(\"eu-west-2\")` for a string or " +
	"`value = jsonencode({ replicas = 3 })` for an object."

func (r *variableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := scopeAttributes()
	maps.Copy(attrs, map[string]schema.Attribute{
		"value": schema.StringAttribute{
			Required:            true,
			CustomType:          jsontypes.NormalizedType{},
			MarkdownDescription: variableValueDescription,
		},
	})
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages one variable in Wayfinder's variable and secret hierarchy, at tenant, " +
			"workspace, environment or stack instance scope. Variable values are not secret and are stored in Terraform " +
			"state; use `wayfinder_secret` for sensitive values.\n\n" +
			"Creating a variable whose key already exists at the same scope fails with an import hint " +
			"instead of overwriting it.",
		Attributes: attrs,
	}
}

func (r *variableResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFrom(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func (r *variableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m variableModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k := m.scoped()
	resp.Diagnostics.Append(ensureAbsent(ctx, r.client, k, kindVariables)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := put(ctx, r.client, k, kindVariables, json.RawMessage(m.Value.ValueString())); err != nil {
		resp.Diagnostics.AddError("Unable to create variable "+k.ID(), err.Error())
		return
	}
	m.ID = types.StringValue(k.ID())
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *variableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m variableModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k := m.scoped()
	item, err := get(ctx, r.client, k, kindVariables)
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read variable "+k.ID(), err.Error())
		return
	}
	m.ID = types.StringValue(k.ID())
	m.Value = jsontypes.NewNormalizedValue(string(item.Value))
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *variableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m variableModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k := m.scoped()
	if err := put(ctx, r.client, k, kindVariables, json.RawMessage(m.Value.ValueString())); err != nil {
		resp.Diagnostics.AddError("Unable to update variable "+k.ID(), err.Error())
		return
	}
	m.ID = types.StringValue(k.ID())
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *variableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m variableModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := remove(ctx, r.client, m.scoped(), kindVariables); err != nil {
		resp.Diagnostics.AddError("Unable to delete variable "+m.scoped().ID(), err.Error())
	}
}

func (r *variableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importScope(ctx, req, resp)
}

// importScope sets id, key, workspace, environment and instance from an import ID; Read fills the rest.
func importScope(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	k, err := parseID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	for attr, v := range map[string]types.String{
		"id":          types.StringValue(k.ID()),
		"key":         types.StringValue(k.Key),
		"workspace":   optional(k.Workspace),
		"environment": optional(k.Environment),
		"instance":    optional(k.Instance),
	} {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attr), v)...)
	}
}
