// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package ephemeral

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
)

func webhookConfig(t *testing.T, m webhookIngestURLModel) tfsdk.Config {
	t.Helper()

	var response ephemeral.SchemaResponse
	NewWebhookIngestURL().Schema(t.Context(), ephemeral.SchemaRequest{}, &response)

	result := tfsdk.EphemeralResultData{Schema: response.Schema}
	if diags := result.Set(t.Context(), &m); diags.HasError() {
		t.Fatal(diags)
	}

	return tfsdk.Config{Schema: response.Schema, Raw: result.Raw}
}

func TestWebhookIngestURL_whenSchemaRequested(t *testing.T) {
	// Given a curated resource, when its schema is requested.
	var response ephemeral.SchemaResponse
	NewWebhookIngestURL().Schema(t.Context(), ephemeral.SchemaRequest{}, &response)
	// Then it exposes exactly three string attributes, including no method/body.
	if len(response.Schema.Attributes) != 3 {
		t.Fatal("expected exactly name, workspace and url")
	}

	for _, name := range []string{"name", "workspace", "url"} {
		attribute, ok := response.Schema.Attributes[name]
		if !ok || !attribute.GetType().Equal(types.StringType) ||
			attribute.IsRequired() != (name == "name") || attribute.IsOptional() != (name == "workspace") ||
			attribute.IsComputed() != (name == "url") || attribute.IsSensitive() != (name == "url") {
			t.Fatalf("incorrect curated schema for %s", name)
		}
	}
}

func TestWebhookIngestURL_whenInputsUnknown(t *testing.T) {
	for _, input := range []string{"name", "workspace"} {
		t.Run(input, func(t *testing.T) {
			for _, allowed := range []bool{false, true} {
				// Given one unknown input and a client that must not be used.
				m := webhookIngestURLModel{Name: types.StringValue("hook"), Workspace: types.StringNull(), URL: types.StringUnknown()}
				if input == "name" {
					m.Name = types.StringUnknown()
				} else {
					m.Workspace = types.StringUnknown()
				}

				config := webhookConfig(t, m)
				r := NewWebhookIngestURL().(*webhookIngestURL)
				r.client = &client.Client{}
				response := ephemeral.OpenResponse{Result: tfsdk.EphemeralResultData(config)}
				// When opened with either client capability.
				r.Open(t.Context(), ephemeral.OpenRequest{
					Config: config, ClientCapabilities: ephemeral.OpenClientCapabilities{DeferralAllowed: allowed},
				}, &response)
				// Then the URL remains unknown and deferral follows capability.
				if response.Diagnostics.HasError() {
					t.Fatal(response.Diagnostics)
				}

				var result webhookIngestURLModel
				if diags := response.Result.Get(t.Context(), &result); diags.HasError() {
					t.Fatal(diags)
				}

				if !result.URL.IsUnknown() || (response.Deferred != nil) != allowed {
					t.Fatal("unknown URL or deferral capability was not preserved")
				}

				if allowed && response.Deferred.Reason != ephemeral.DeferredReasonEphemeralResourceConfigUnknown {
					t.Fatal("incorrect deferral reason for unknown input")
				}
			}
		})
	}
}

func TestWebhookIngestURL_whenConfigurationInvalid(t *testing.T) {
	for _, input := range []string{"name", "workspace"} {
		for _, value := range []string{"", ".", "..", "../hook", "a/b", `a\b`} {
			t.Run(input+"_"+value, func(t *testing.T) {
				// Given an invalid path segment in an otherwise valid config.
				m := webhookIngestURLModel{Name: types.StringValue("hook"), Workspace: types.StringNull(), URL: types.StringUnknown()}
				if input == "name" {
					m.Name = types.StringValue(value)
				} else {
					m.Workspace = types.StringValue(value)
				}

				var response ephemeral.ValidateConfigResponse
				// When validated, then the precise input error is diagnosed.
				NewWebhookIngestURL().(ephemeral.EphemeralResourceWithValidateConfig).ValidateConfig(t.Context(),
					ephemeral.ValidateConfigRequest{Config: webhookConfig(t, m)}, &response)
				assertDiagnostic(t, response.Diagnostics, "Invalid resource reference", input, "")
			})
		}
	}
}
