package vars

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
)

var (
	_ resource.ResourceWithConfigure   = (*secretResource)(nil)
	_ resource.ResourceWithImportState = (*secretResource)(nil)
)

// NewSecretResource returns the wayfinder_secret resource.
func NewSecretResource() resource.Resource { return &secretResource{} }

type secretResource struct{ client *client.Client }

// secretModel never carries the secret: value_wo is write-only and always null in plan and
// state; the value is read from configuration only, when it must be sent.
type secretModel struct {
	ID             types.String         `tfsdk:"id"`
	Key            types.String         `tfsdk:"key"`
	Workspace      types.String         `tfsdk:"workspace"`
	Environment    types.String         `tfsdk:"environment"`
	ValueWO        jsontypes.Normalized `tfsdk:"value_wo"`
	ValueWOVersion types.String         `tfsdk:"value_wo_version"`
}

func (m secretModel) scoped() scopedKey { return keyOf(m.Key, m.Workspace, m.Environment) }

func (r *secretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *secretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := scopeAttributes()
	attrs["value_wo"] = schema.StringAttribute{
		Required:   true,
		WriteOnly:  true,
		Sensitive:  true,
		CustomType: jsontypes.NormalizedType{},
		MarkdownDescription: "Value of the secret as a JSON document, for example " +
			"`value_wo = jsonencode(var.db_password)` or `value_wo = jsonencode({ user = \"app\", password = var.db_password })`. " +
			"Write-only: it is never stored in Terraform state or plan, and Wayfinder never returns it. " +
			"It is sent when the secret is created and whenever `value_wo_version` changes; changing only " +
			"`value_wo` does not update the secret. Requires Terraform or OpenTofu 1.11 or later.",
	}
	attrs["value_wo_version"] = schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Change this value to send a new `value_wo` to Wayfinder.",
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages one secret in Wayfinder's variable and secret hierarchy, at tenant, " +
			"workspace or environment scope. The secret value is write-only: Terraform sends it to " +
			"Wayfinder but never stores or reads it back, so changes made outside Terraform to the value " +
			"are not detected; only the existence of the secret is.\n\n" +
			"Creating a secret whose key already exists at the same scope fails with an import hint " +
			"instead of overwriting it. After `terraform import`, set `value_wo` and `value_wo_version` " +
			"in configuration; the value is sent the next time `value_wo_version` changes.",
		Attributes: attrs,
	}
}

func (r *secretResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := clientFrom(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

// configValue reads the write-only value from configuration (it is absent from plan and state).
func configValue(ctx context.Context, config tfsdk.Config) (json.RawMessage, diag.Diagnostics) {
	var v jsontypes.Normalized
	diags := config.GetAttribute(ctx, path.Root("value_wo"), &v)
	if diags.HasError() {
		return nil, diags
	}
	if v.IsNull() || v.IsUnknown() {
		diags.AddAttributeError(path.Root("value_wo"), "Missing secret value", "value_wo must be set to a known value.")
		return nil, diags
	}
	return json.RawMessage(v.ValueString()), diags
}

func (r *secretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m secretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	value, diags := configValue(ctx, req.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	k := m.scoped()
	resp.Diagnostics.Append(ensureAbsent(ctx, r.client, k, kindSecrets)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := put(ctx, r.client, k, kindSecrets, value); err != nil {
		resp.Diagnostics.AddError("Unable to create secret "+k.ID(), err.Error())
		return
	}
	m.ID = types.StringValue(k.ID())
	m.ValueWO = jsontypes.NewNormalizedNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

// Read checks existence only; the secret value is never requested or stored.
func (r *secretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m secretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k := m.scoped()
	_, err := get(ctx, r.client, k, kindSecrets)
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read secret "+k.ID(), err.Error())
		return
	}
	m.ID = types.StringValue(k.ID())
	m.ValueWO = jsontypes.NewNormalizedNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

// Update sends value_wo only when value_wo_version changed; otherwise it makes no API call.
func (r *secretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state secretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k := plan.scoped()
	if !plan.ValueWOVersion.Equal(state.ValueWOVersion) {
		value, diags := configValue(ctx, req.Config)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := put(ctx, r.client, k, kindSecrets, value); err != nil {
			resp.Diagnostics.AddError("Unable to update secret "+k.ID(), err.Error())
			return
		}
	}
	plan.ID = types.StringValue(k.ID())
	plan.ValueWO = jsontypes.NewNormalizedNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *secretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m secretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := remove(ctx, r.client, m.scoped(), kindSecrets); err != nil {
		resp.Diagnostics.AddError("Unable to delete secret "+m.scoped().ID(), err.Error())
	}
}

// ImportState imports existence only; the value is never imported.
func (r *secretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importScope(ctx, req, resp)
}
