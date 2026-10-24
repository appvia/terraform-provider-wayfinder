// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

// Package webhookrotation rotates a Wayfinder webhook's ingest URL through the webhook rotate
// sub-resource. The URL returned by the rotation is a credential and is never stored.
//
// ProviderData contract: the provider's Configure must set ResourceData to a *client.Client
// whose Tenant is the provider tenant. A nil ProviderData (early validation calls) is
// tolerated; any other type is reported as an error.
package webhookrotation

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
)

var (
	_ resource.ResourceWithConfigure   = (*rotationResource)(nil)
	_ resource.ResourceWithImportState = (*rotationResource)(nil)
)

// NewResource returns the wayfinder_webhook_rotation resource.
func NewResource() resource.Resource { return &rotationResource{} }

type rotationResource struct{ client *client.Client }

type rotationModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Webhook    types.String `tfsdk:"webhook"`
	Workspace  types.String `tfsdk:"workspace"`
	Triggers   types.Map    `tfsdk:"triggers"`
	RotationID types.String `tfsdk:"rotation_id"`
	RotatedAt  types.String `tfsdk:"rotated_at"`
}

func (r *rotationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_rotation"
}

func (r *rotationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	keep := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Rotates a webhook's ingest URL. Creating this resource, or changing `webhook`, " +
			"`workspace` or `triggers`, issues a new URL and immediately invalidates the previous one. The URL " +
			"is never stored; read it with the `wayfinder_webhook_ingest_url` ephemeral resource using " +
			"`name = wayfinder_webhook_rotation.<label>.name`. Destroying this resource leaves the current URL " +
			"working. Requires the `webhooks/rotate` permission.",
		Attributes: map[string]schema.Attribute{
			"webhook": schema.StringAttribute{
				Required:            true,
				PlanModifiers:       replace,
				Validators:          []validator.String{segmentValidator{}},
				MarkdownDescription: "Name of the webhook to rotate. Changing it rotates the newly named webhook.",
			},
			"workspace": schema.StringAttribute{
				Optional:            true,
				PlanModifiers:       replace,
				Validators:          []validator.String{segmentValidator{}},
				MarkdownDescription: "Workspace of the webhook. Omit for a tenant-scoped webhook.",
			},
			"triggers": schema.MapAttribute{
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()},
				MarkdownDescription: "Arbitrary values that rotate the URL again whenever any of them change, " +
					"for example `{ rotation = time_rotating.webhook.id }`.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				PlanModifiers:       keep,
				MarkdownDescription: "`<webhook>` for a tenant-scoped webhook, or `<workspace>/<webhook>`.",
			},
			"name": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: keep,
				MarkdownDescription: "Webhook name, equal to `webhook`. Pass it to `wayfinder_webhook_ingest_url` " +
					"so the URL is read only after this rotation.",
			},
			"rotation_id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: keep,
				MarkdownDescription: "Random value, unrelated to the URL, that changes on every rotation. Use it as " +
					"the `*_wo_version` of write-only attributes that receive the URL.",
			},
			"rotated_at": schema.StringAttribute{
				Computed:            true,
				PlanModifiers:       keep,
				MarkdownDescription: "Time of the last rotation (RFC 3339, UTC). Informational only.",
			},
		},
	}
}

func (r *rotationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
