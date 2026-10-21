// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

// Package featureintegration manages tenant-scoped built-in Wayfinder features through
// the enable action and the Integration resource API.
//
// ProviderData contract: the provider's Configure must set ResourceData and DataSourceData to a
// *client.Client whose Tenant is the provider tenant. A nil ProviderData (early validation calls)
// is tolerated; any other type is reported as an error.
package featureintegration

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
)

var (
	_ resource.ResourceWithConfigure   = (*featureResource)(nil)
	_ resource.ResourceWithImportState = (*featureResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*featureResource)(nil)
)

// NewResource returns the wayfinder_feature_integration resource.
func NewResource() resource.Resource { return &featureResource{} }

type featureResource struct{ client *client.Client }

type featureModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Params               types.Map    `tfsdk:"params"`
	KeepEnabledOnDestroy types.Bool   `tfsdk:"keep_enabled_on_destroy"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	Type                 types.String `tfsdk:"type"`
}

func (r *featureResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_integration"
}

func (r *featureResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Enables a built-in Wayfinder feature integration, such as `azure`, `github` or `ai`, " +
			"at tenant scope. Use `wayfinder_integration` to create custom integrations instead. Enabling a feature " +
			"is idempotent; applying after it is disabled outside Terraform re-enables it. Destroy disables the " +
			"feature unless `keep_enabled_on_destroy` is true.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "Integration object name, in the form `wayfinder.<name>`.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{bareNameValidator{}},
				MarkdownDescription: "Bare feature name, for example `azure`, not `wayfinder.azure`.",
			},
			"params": schema.MapAttribute{
				Optional:    true,
				ElementType: jsontypes.NormalizedType{},
				Validators:  []validator.Map{paramValuesValidator{}},
				MarkdownDescription: "Feature parameters as JSON-encoded values, for example " +
					"`enable_wayfinder = jsonencode(false)`. Only keys set here are managed; removing a key " +
					"deletes it in Wayfinder. Use the JSON type the feature expects. For a JSON null, use " +
					"`jsonencode(null)`; Terraform null map elements are not accepted.",
			},
			"keep_enabled_on_destroy": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				MarkdownDescription: "Destroy disables the feature by default. Set to true to leave it enabled, " +
					"and apply this value before destroying. A `removed` block with `lifecycle { destroy = false }` " +
					"can also drop one resource from state without disabling it.",
			},
			"enabled": schema.BoolAttribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{reenable{}},
				MarkdownDescription: "Whether the feature is enabled in Wayfinder.",
			},
			"type": schema.StringAttribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "Integration type reported by Wayfinder (`feature`).",
			},
		},
	}
}

func (r *featureResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
