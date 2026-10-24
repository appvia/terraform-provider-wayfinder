// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package ephemeral

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

type staticToken string

func (s staticToken) Token(context.Context) (string, error) { return string(s), nil }

func actionClient(server *httptest.Server) *client.Client {
	return &client.Client{BaseURL: server.URL, Tenant: "acme", HTTP: server.Client(), TokenSource: staticToken("test-token")}
}

func webhookKind() manifest.Kind {
	return manifest.Kind{Group: "integrations", Version: "v3", Kind: "Webhook", APIPath: "webhooks", Scopes: []string{"tenant", "workspace"}}
}

func TestGetAction_whenScopeSelected(t *testing.T) {
	tests := []struct {
		name, workspace, environment, version, objectName, suffix string
	}{
		{"tenant", "", "", "", "hook", "/integrations/v3/webhooks/hook/ingesturl"},
		{"workspace", "team", "", "", "hook", "/workspaces/team/integrations/v3/webhooks/hook/ingesturl"},
		{"environment", "team", "dev", "", "hook", "/workspaces/team/envs/dev/integrations/v3/webhooks/hook/ingesturl"},
		{"versioned", "team", "", "1.2", "hook", "/workspaces/team/integrations/v3/webhooks/hook/versions/1.2/ingesturl"},
		{"escaped", "team space", "", "", "hook?#%", "/workspaces/team%20space/integrations/v3/webhooks/hook%3F%23%25/ingesturl"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a real HTTP endpoint at the expected scoped action path.
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.EscapedPath() != "/api/v3/resources/tenants/acme"+tt.suffix || req.URL.RawQuery != "" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL)
				}

				body, err := io.ReadAll(req.Body)
				if err != nil || len(body) != 0 {
					t.Errorf("GET body = %q, err = %v", body, err)
				}

				if req.Header.Get("Authorization") != "Bearer test-token" {
					t.Errorf("missing provider authentication")
				}

				if _, err := io.WriteString(w, `{"url":"test-credential"}`); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()

			k := webhookKind()
			ref := client.Ref{Group: k.Group, Version: k.Version, APIPath: k.APIPath, Name: tt.objectName,
				Workspace: tt.workspace, Environment: tt.environment, Versioned: tt.version != "", ObjectVersion: tt.version}
			// When opening the action.
			raw, diags := getAction(t.Context(), actionClient(server), k, ref, "ingesturl")
			// Then the response is returned without changing its contents.
			if diags.HasError() || string(raw) != `{"url":"test-credential"}` {
				t.Fatalf("response %s, diagnostics %v", raw, diags)
			}
		})
	}
}

func TestGetAction_whenRequestFails(t *testing.T) {
	tests := []struct {
		name, body, summary, hint string
		status                    int
	}{
		{"missing", `{"message":"secret-body","detail":"secret-body"}`, "Wayfinder action not found", "wf rotate webhook hook", 404},
		{"forbidden", `secret-body`, "Wayfinder action permission denied", "webhooks/ingesturl", 403},
		{"validation", `{"message":"secret-body","fieldErrors":[{"field":"token","message":"secret-body"}]}`, "Wayfinder request failed", "GET /api/", 400},
		{"server error", `secret-body`, "Wayfinder request failed", "HTTP 500", 500},
		{"malformed success", `secret-body`, "Wayfinder request failed", "GET /api/", 200},
		{"trailing success", `{} {"token":"secret-body"}`, "Wayfinder request failed", "GET /api/", 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a server whose response body contains a credential.
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)

				if _, err := io.WriteString(w, tt.body); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()

			k := webhookKind()
			ref := client.Ref{Group: k.Group, Version: k.Version, APIPath: k.APIPath, Name: "hook"}
			// When reading the action.
			raw, diags := getAction(t.Context(), actionClient(server), k, ref, "ingesturl")
			// Then the actionable error never includes the body.
			assertDiagnostic(t, diags, tt.summary, tt.hint, "secret-body")

			if raw != nil {
				t.Fatalf("failed action returned a body")
			}
		})
	}
}

func TestGetAction_whenPathMalformed(t *testing.T) {
	tests := []struct{ name, action, objectName, workspace, environment, version string }{
		{"action traversal", "../rotate", "hook", "", "", ""},
		{"action slash", "ingest/url", "hook", "", "", ""},
		{"action uppercase", "Rotate", "hook", "", "", ""},
		{"name traversal", "ingesturl", "../hook", "", "", ""},
		{"name slash", "ingesturl", "a/b", "", "", ""},
		{"name parent", "ingesturl", "..", "", "", ""},
		{"workspace traversal", "ingesturl", "hook", "..", "", ""},
		{"environment traversal", "ingesturl", "hook", "team", "../dev", ""},
		{"version traversal", "ingesturl", "hook", "", "", "../v1"},
		{"missing name", "ingesturl", "", "", "", ""},
		{"environment without workspace", "ingesturl", "hook", "", "dev", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given invalid path input and a server that must not be called.
			server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Error("invalid configuration reached the server")
			}))
			defer server.Close()

			k := webhookKind()
			ref := client.Ref{Group: k.Group, Version: k.Version, APIPath: k.APIPath, Name: tt.objectName,
				Workspace: tt.workspace, Environment: tt.environment, ObjectVersion: tt.version, Versioned: tt.version != ""}
			// When the shared core receives the malformed reference.
			_, diags := getAction(t.Context(), actionClient(server), k, ref, tt.action)
			// Then it rejects the exact invalid input class.
			summary := "Invalid resource reference"
			if strings.HasPrefix(tt.name, "action") {
				summary = "Invalid action"
			}

			assertDiagnostic(t, diags, summary, "", "")
		})
	}
}

func assertDiagnostic(t *testing.T, diags diag.Diagnostics, summary, hint, forbidden string) {
	t.Helper()

	if len(diags) != 1 || diags[0].Severity() != diag.SeverityError || diags[0].Summary() != summary {
		t.Fatalf("expected one %q error, got %v", summary, diags)
	}

	detail := diags[0].Detail()
	if hint != "" && !strings.Contains(detail, hint) {
		t.Errorf("diagnostic missing %q: %s", hint, detail)
	}

	if forbidden != "" && strings.Contains(fmt.Sprint(diags), forbidden) {
		t.Errorf("diagnostic contains a response body")
	}
}
