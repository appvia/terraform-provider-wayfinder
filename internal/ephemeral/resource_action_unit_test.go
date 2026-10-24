// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package ephemeral

import (
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

func actionModel() resourceActionModel {
	return resourceActionModel{
		Type: types.StringValue("integrations/v3/Webhook"), Name: types.StringValue("hook"),
		Action: types.StringValue("ingesturl"), Workspace: types.StringNull(), Environment: types.StringNull(),
		Version: types.StringNull(), ResponseExportValues: types.ListNull(types.StringType), Output: types.DynamicNull(),
	}
}

func actionConfig(t *testing.T, m resourceActionModel) tfsdk.Config {
	t.Helper()

	var response ephemeral.SchemaResponse
	NewResourceAction().Schema(t.Context(), ephemeral.SchemaRequest{}, &response)

	result := tfsdk.EphemeralResultData{Schema: response.Schema}
	if diags := result.Set(t.Context(), &m); diags.HasError() {
		t.Fatal(diags)
	}

	return tfsdk.Config{Schema: response.Schema, Raw: result.Raw}
}

func TestResourceAction_whenSchemaRequested(t *testing.T) {
	// Given a new resource, when its schema is requested.
	resource := NewResourceAction()

	var response ephemeral.SchemaResponse
	resource.Schema(t.Context(), ephemeral.SchemaRequest{}, &response)

	// Then the exact eight attributes expose no method or body.
	names := []string{"action", "environment", "name", "output", "response_export_values", "type", "version", "workspace"}
	if len(response.Schema.Attributes) != len(names) {
		t.Fatalf("unexpected schema attributes: %v", response.Schema.Attributes)
	}

	t.Log("Schema dump (no method/body attributes):")

	for _, name := range names {
		attribute, ok := response.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing attribute %q", name)
		}

		required := slices.Contains([]string{"type", "name", "action"}, name)
		computed := name == "output"

		if attribute.IsRequired() != required || attribute.IsComputed() != computed || attribute.IsOptional() != (!required && !computed) || attribute.IsSensitive() != computed {
			t.Errorf("incorrect flags for %s", name)
		}

		wantType := attr.Type(types.StringType)
		if name == "output" {
			wantType = types.DynamicType
		}

		if name == "response_export_values" {
			wantType = types.ListType{ElemType: types.StringType}
		}

		if !attribute.GetType().Equal(wantType) {
			t.Errorf("incorrect type for %s: %s", name, attribute.GetType())
		}

		t.Logf("%s: %s required=%t optional=%t computed=%t sensitive=%t", name, attribute.GetType(), attribute.IsRequired(), attribute.IsOptional(), attribute.IsComputed(), attribute.IsSensitive())
	}

	var metadata ephemeral.MetadataResponse
	resource.Metadata(t.Context(), ephemeral.MetadataRequest{ProviderTypeName: "wayfinder"}, &metadata)

	if metadata.TypeName != "wayfinder_resource_action" {
		t.Fatalf("wrong type name: %s", metadata.TypeName)
	}
}

func TestResourceAction_whenConfigurationInvalid(t *testing.T) {
	tests := []struct {
		name, summary, hint string
		set                 func(*resourceActionModel)
	}{
		{"bad action", "Invalid action", "^[a-z][a-z0-9-]*$", func(m *resourceActionModel) { m.Action = types.StringValue("Rotate") }},
		{"action traversal", "Invalid action", "", func(m *resourceActionModel) { m.Action = types.StringValue("../rotate") }},
		{"action slash", "Invalid action", "", func(m *resourceActionModel) { m.Action = types.StringValue("ingest/url") }},
		{"environment without workspace", "Environment requires workspace", "", func(m *resourceActionModel) {
			m.Type, m.Environment = types.StringValue("inference/v3/AIAgent"), types.StringValue("dev")
		}},
		{"bad type", "Invalid resource type", "group/version/Kind", func(m *resourceActionModel) { m.Type = types.StringValue("Webhook") }},
		{"type path traversal", "Invalid resource type", "group/version/Kind", func(m *resourceActionModel) { m.Type = types.StringValue("../integrations/v3/Webhook") }},
		{"unknown type", "Unknown resource type", "group/version/Kind", func(m *resourceActionModel) { m.Type = types.StringValue("integrations/v3/Missing") }},
		{"wrong API version", "Unknown resource type", "group/version/Kind", func(m *resourceActionModel) { m.Type = types.StringValue("integrations/v99/Webhook") }},
		{"unsupported scope", "Unsupported resource scope", "tenant, workspace", func(m *resourceActionModel) {
			m.Workspace, m.Environment = types.StringValue("team"), types.StringValue("dev")
		}},
		{"version required", "Object version required", "", func(m *resourceActionModel) { m.Type = types.StringValue("cloudresources/v3/CloudResourcePlan") }},
		{"version forbidden", "Object version not supported", "", func(m *resourceActionModel) { m.Version = types.StringValue("v1") }},
		{"name traversal", "Invalid resource reference", "", func(m *resourceActionModel) { m.Name = types.StringValue("../hook") }},
		{"name slash", "Invalid resource reference", "", func(m *resourceActionModel) { m.Name = types.StringValue("a/b") }},
		{"name dot", "Invalid resource reference", "", func(m *resourceActionModel) { m.Name = types.StringValue(".") }},
		{"empty name", "Invalid resource reference", "", func(m *resourceActionModel) { m.Name = types.StringValue("") }},
		{"workspace traversal", "Invalid resource reference", "", func(m *resourceActionModel) { m.Workspace = types.StringValue("..") }},
		{"empty export path", "Invalid response export path", "", func(m *resourceActionModel) {
			m.ResponseExportValues = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("")})
		}},
		{"empty export component", "Invalid response export path", "", func(m *resourceActionModel) {
			m.ResponseExportValues = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("status..token")})
		}},
		{"null export path", "Invalid response export path", "", func(m *resourceActionModel) {
			m.ResponseExportValues = types.ListValueMust(types.StringType, []attr.Value{types.StringNull()})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given an otherwise valid configuration with one invalid input.
			m := actionModel()
			tt.set(&m)

			var response ephemeral.ValidateConfigResponse
			// When framework-style validation runs.
			NewResourceAction().(ephemeral.EphemeralResourceWithValidateConfig).ValidateConfig(t.Context(), ephemeral.ValidateConfigRequest{Config: actionConfig(t, m)}, &response)
			// Then the exact failure class is diagnosed, not a generic error.
			assertDiagnostic(t, response.Diagnostics, tt.summary, tt.hint, "")
		})
	}
}

func TestResourceAction_whenManifestKindSelected(t *testing.T) {
	// Given all embedded kinds, including any manifest-only entries.
	embedded, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}

	for _, k := range embedded.Kinds {
		t.Run(k.Group+"/"+k.Kind, func(t *testing.T) {
			m := actionModel()
			m.Type = types.StringValue(k.Group + "/" + k.Version + "/" + k.Kind)

			if !slices.Contains(k.Scopes, "tenant") {
				m.Workspace = types.StringValue("team")
				if !slices.Contains(k.Scopes, "workspace") {
					m.Environment = types.StringValue("dev")
				}
			}

			if k.Versioned {
				m.Version = types.StringValue("1")
			}

			// When a valid scope and object version are selected.
			got, diags := m.validate(t.Context())
			// Then resolution preserves the manifest's route metadata.
			if diags.HasError() || got.Kind != k.Kind || got.APIPath != k.APIPath || got.Versioned != k.Versioned {
				t.Fatalf("kind %v, diagnostics %v", got, diags)
			}
		})
	}
}

func TestResourceAction_whenProviderDataConfigured(t *testing.T) {
	for _, tt := range []struct {
		name string
		data any
	}{
		{"absent", nil},
		{"client", &client.Client{Tenant: "acme"}},
		{"wrong type", "not a client"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Given framework ProviderData, when Configure runs.
			resource := NewResourceAction().(*resourceAction)

			var response ephemeral.ConfigureResponse
			resource.Configure(t.Context(), ephemeral.ConfigureRequest{ProviderData: tt.data}, &response)
			// Then only *client.Client is accepted, with nil tolerated.
			if tt.name == "wrong type" {
				assertDiagnostic(t, response.Diagnostics, "Unexpected provider data", "expected *client.Client", "")
			} else if response.Diagnostics.HasError() || (tt.data != nil && resource.client != tt.data) {
				t.Fatalf("client %v, diagnostics %v", resource.client, response.Diagnostics)
			}
		})
	}
}
