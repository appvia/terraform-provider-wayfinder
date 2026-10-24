// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package ephemeral

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestResourceAction_whenOpened(t *testing.T) {
	tests := []struct {
		name, raw, want string
		exports         types.List
	}{
		{"full response", `{"url":"credential","status":true}`, `{"url":"credential","status":true}`, types.ListNull(types.StringType)},
		{"filtered", `{"url":"credential","ignored":"private"}`, `{"url":"credential"}`, types.ListValueMust(types.StringType, []attr.Value{types.StringValue("url")})},
		{"missing path", `{"url":"credential"}`, `{"status":{"token":null}}`, types.ListValueMust(types.StringType, []attr.Value{types.StringValue("status.token")})},
		{"empty selection", `{"url":"credential"}`, `{}`, types.ListValueMust(types.StringType, []attr.Value{})},
		{"null response", `null`, `null`, types.ListNull(types.StringType)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a configured resource and real HTTP response.
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != "/api/v3/resources/tenants/acme/integrations/v3/webhooks/hook/ingesturl" {
					t.Errorf("unexpected request %s %s", req.Method, req.URL)
				}

				if _, err := io.WriteString(w, tt.raw); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()

			resource := NewResourceAction().(*resourceAction)

			var configured ephemeral.ConfigureResponse
			resource.Configure(t.Context(), ephemeral.ConfigureRequest{ProviderData: actionClient(server)}, &configured)

			if configured.Diagnostics.HasError() {
				t.Fatal(configured.Diagnostics)
			}

			m := actionModel()
			m.ResponseExportValues, m.Output = tt.exports, types.DynamicUnknown()
			config := actionConfig(t, m)
			response := ephemeral.OpenResponse{Result: tfsdk.EphemeralResultData(config)}
			// When Open reads the action.
			resource.Open(t.Context(), ephemeral.OpenRequest{Config: config}, &response)
			// Then result data contains the selected value, not an unknown or a stub.
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}

			var result resourceActionModel
			if diags := response.Result.Get(t.Context(), &result); diags.HasError() {
				t.Fatal(diags)
			}

			want, diags := responseDynamic(t.Context(), json.RawMessage(tt.want), nil)
			if diags.HasError() || !result.Output.Equal(want) {
				t.Fatalf("output %s, want %s, diagnostics %v", result.Output, want, diags)
			}
		})
	}
}

func TestResourceAction_whenInputsUnknown(t *testing.T) {
	tests := []struct {
		name string
		set  func(*resourceActionModel)
	}{
		{"type", func(m *resourceActionModel) { m.Type = types.StringUnknown() }},
		{"name", func(m *resourceActionModel) { m.Name = types.StringUnknown() }},
		{"workspace", func(m *resourceActionModel) { m.Workspace = types.StringUnknown() }},
		{"environment", func(m *resourceActionModel) { m.Environment = types.StringUnknown() }},
		{"version", func(m *resourceActionModel) { m.Version = types.StringUnknown() }},
		{"action", func(m *resourceActionModel) { m.Action = types.StringUnknown() }},
		{"export list", func(m *resourceActionModel) { m.ResponseExportValues = types.ListUnknown(types.StringType) }},
		{"export element", func(m *resourceActionModel) {
			m.ResponseExportValues = types.ListValueMust(types.StringType, []attr.Value{types.StringUnknown()})
		}},
	}
	for _, tt := range tests {
		for _, allowed := range []bool{false, true} {
			t.Run(tt.name+map[bool]string{false: "/without deferral", true: "/with deferral"}[allowed], func(t *testing.T) {
				// Given unknown configuration and a server that must not be called.
				server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
					t.Error("unknown input reached the server")
				}))
				defer server.Close()

				resource := &resourceAction{client: actionClient(server)}
				m := actionModel()
				tt.set(&m)
				config := actionConfig(t, m)
				response := ephemeral.OpenResponse{Result: tfsdk.EphemeralResultData(config)}
				// When Open receives either framework capability setting.
				resource.Open(t.Context(), ephemeral.OpenRequest{Config: config, ClientCapabilities: ephemeral.OpenClientCapabilities{DeferralAllowed: allowed}}, &response)
				// Then output remains unknown without a null success or API request.
				if response.Diagnostics.HasError() {
					t.Fatal(response.Diagnostics)
				}

				var result resourceActionModel
				if diags := response.Result.Get(t.Context(), &result); diags.HasError() {
					t.Fatal(diags)
				}

				if !result.Output.IsUnknown() {
					t.Fatalf("expected unknown output, got %s", result.Output)
				}

				if allowed {
					if response.Deferred == nil || response.Deferred.Reason != ephemeral.DeferredReasonEphemeralResourceConfigUnknown {
						t.Fatalf("incorrect deferral: %v", response.Deferred)
					}
				} else if response.Deferred != nil {
					t.Fatalf("deferral returned without client support")
				}
			})
		}
	}
}

func TestResourceAction_whenClientUnavailable(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		t.Run(map[bool]string{false: "without deferral", true: "with deferral"}[allowed], func(t *testing.T) {
			// Given known configuration but no configured provider client.
			config := actionConfig(t, actionModel())
			response := ephemeral.OpenResponse{Result: tfsdk.EphemeralResultData(config)}
			// When Open receives the request.
			NewResourceAction().Open(t.Context(), ephemeral.OpenRequest{Config: config, ClientCapabilities: ephemeral.OpenClientCapabilities{DeferralAllowed: allowed}}, &response)
			// Then provider deferral is used only when permitted, otherwise an error.
			if !allowed {
				assertDiagnostic(t, response.Diagnostics, "Wayfinder client not configured", "", "")
				return
			}

			if response.Diagnostics.HasError() || response.Deferred == nil || response.Deferred.Reason != ephemeral.DeferredReasonProviderConfigUnknown {
				t.Fatalf("deferral %v, diagnostics %v", response.Deferred, response.Diagnostics)
			}

			var result resourceActionModel
			if diags := response.Result.Get(t.Context(), &result); diags.HasError() {
				t.Fatal(diags)
			}

			if !result.Output.IsUnknown() {
				t.Fatalf("deferred output must remain unknown")
			}
		})
	}
}
