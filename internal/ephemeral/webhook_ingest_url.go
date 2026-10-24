// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package ephemeral

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

var (
	_ ephemeral.EphemeralResourceWithConfigure      = (*webhookIngestURL)(nil)
	_ ephemeral.EphemeralResourceWithValidateConfig = (*webhookIngestURL)(nil)
)

// NewWebhookIngestURL returns the curated webhook credential reader.
func NewWebhookIngestURL() ephemeral.EphemeralResource { return &webhookIngestURL{} }

// Reuse provider-client configuration; schema, validation and Open are curated.
type webhookIngestURL struct{ resourceAction }

type webhookIngestURLModel struct {
	Name      types.String `tfsdk:"name"`
	Workspace types.String `tfsdk:"workspace"`
	URL       types.String `tfsdk:"url"`
}

func (r *webhookIngestURL) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_ingest_url"
}

func (r *webhookIngestURL) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a webhook's ingest URL on every plan and apply. The URL is a credential: " +
			"it is sensitive, ephemeral and never stored in state or plan files. It exists only after " +
			"the webhook is rotated with `wayfinder_webhook_rotation` or `wf rotate webhook <name>`, and requires the `webhooks/ingesturl` permission (granted with rotate). " +
			"Requires Terraform/OpenTofu >= 1.11.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true, MarkdownDescription: "Webhook name at the selected scope.",
			},
			"workspace": schema.StringAttribute{
				Optional: true, MarkdownDescription: "Workspace scope. Omit for tenant scope.",
			},
			"url": schema.StringAttribute{
				Computed: true, Sensitive: true, MarkdownDescription: "Webhook ingest URL. Treat this URL as a credential.",
			},
		},
	}
}

func (m webhookIngestURLModel) validate() diag.Diagnostics {
	var diags diag.Diagnostics

	for _, input := range []struct {
		name  string
		value types.String
	}{{"name", m.Name}, {"workspace", m.Workspace}} {
		if !input.value.IsUnknown() && (invalidSegment(input.value.ValueString()) || (!input.value.IsNull() && input.value.ValueString() == "")) {
			diags.AddAttributeError(path.Root(input.name), "Invalid resource reference", input.name+" must be a non-empty path segment, not . or .. .")
		}
	}

	return diags
}

func (r *webhookIngestURL) ValidateConfig(ctx context.Context, req ephemeral.ValidateConfigRequest, resp *ephemeral.ValidateConfigResponse) {
	var m webhookIngestURLModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)

	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(m.validate()...)
	}
}

func (r *webhookIngestURL) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var m webhookIngestURLModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(m.validate()...)

	if resp.Diagnostics.HasError() {
		return
	}

	m.URL = types.StringUnknown()
	unknown := m.Name.IsUnknown() || m.Workspace.IsUnknown()

	if unknown || r.client == nil {
		if req.ClientCapabilities.DeferralAllowed {
			reason := ephemeral.DeferredReasonEphemeralResourceConfigUnknown
			if r.client == nil {
				reason = ephemeral.DeferredReasonProviderConfigUnknown
			}

			resp.Deferred = &ephemeral.Deferred{Reason: reason}
		} else if r.client == nil && !unknown {
			resp.Diagnostics.AddError("Wayfinder client not configured", "Configure the Wayfinder provider before reading a webhook ingest URL.")
			return
		}

		resp.Diagnostics.Append(resp.Result.Set(ctx, &m)...)

		return
	}

	embedded, err := manifest.LoadEmbedded()
	if err != nil {
		resp.Diagnostics.AddError("Unable to load Wayfinder manifest", err.Error())
		return
	}

	k, ok := embedded.FindKind("integrations", "Webhook")
	if !ok {
		resp.Diagnostics.AddError("Webhook type unavailable", "The embedded Wayfinder manifest does not contain integrations/Webhook.")
		return
	}

	ref := client.Ref{
		Group: k.Group, Version: k.Version, APIPath: k.APIPath, Versioned: k.Versioned,
		Name: m.Name.ValueString(), Workspace: m.Workspace.ValueString(),
	}
	raw, diags := getAction(ctx, r.client, *k, ref, "ingesturl")
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || result.URL == "" {
		resp.Diagnostics.AddError("Invalid webhook ingest URL response", "Wayfinder must return a non-empty string in the url field.")
		return
	}

	m.URL = types.StringValue(result.URL)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &m)...)
}
