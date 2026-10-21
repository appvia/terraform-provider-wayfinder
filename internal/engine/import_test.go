// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"regexp"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func modifiedRegex() *regexp.Regexp {
	return regexp.MustCompile(`(?s)changed outside Terraform.*re-import`)
}

func TestImport_whenIDSelectsScopeAndVersion(t *testing.T) {
	for _, tc := range []struct {
		kind, id string
		invalid  bool
	}{
		{"Workspace", "example", false},
		{"Integration", "ws/example", false},
		{"Integration", "ws/dev/example", false},
		{"Blueprint", "ws/example@1.2.3", false},
		{"Integration", "a/b/c/d", true},
		{"Workspace", "ws/example", true},
		{"Blueprint", "ws/example", true},
	} {
		t.Run(tc.id, func(t *testing.T) {
			// Given an import ID and the manifest scope contract.
			e := newTestEnv(t, tc.kind)
			r := directResource(e)
			resp := resource.ImportStateResponse{State: tfsdk.State{Raw: tftypes.NewValue(rawValue(t, e, nil).Type(), nil), Schema: r.schema}}
			// When importing.
			r.ImportState(t.Context(), resource.ImportStateRequest{ID: tc.id}, &resp)
			// Then scope/version are retained, or invalid IDs have an error.
			if resp.Diagnostics.HasError() != tc.invalid {
				t.Fatalf("diagnostics = %v", resp.Diagnostics)
			}

			if !tc.invalid {
				i, err := identityOf(e.k, resp.State.Raw)
				if err != nil || i.ID() != tc.id {
					t.Fatalf("identity = %v, %v", i, err)
				}
			}
		})
	}
}

func importDirect(t *testing.T, e testEnv, id string) resource.ImportStateResponse {
	t.Helper()

	r := directResource(e)
	resp := resource.ImportStateResponse{State: tfsdk.State{Raw: tftypes.NewValue(rawValue(t, e, nil).Type(), nil), Schema: r.schema}}
	r.ImportState(t.Context(), resource.ImportStateRequest{ID: id}, &resp)

	return resp
}

func TestImport_whenBareFeatureNameMissing(t *testing.T) {
	// Given an Integration kind and no object for a bare feature name.
	e := newFeatureEnv(t)
	// When importing it.
	resp := importDirect(t, e, "azure")
	// Then the error points at the wayfinder.-prefixed name and the feature resource.
	got := resp.Diagnostics.Errors()
	if len(got) != 1 || got[0].Summary() != "Cannot import non-existent integration" ||
		!strings.Contains(got[0].Detail(), `"wayfinder.azure"`) || !strings.Contains(got[0].Detail(), "wayfinder_feature_integration") {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
}

func TestImport_whenBareFeatureNameExists(t *testing.T) {
	// Given an Integration whose bare name exists.
	e := newFeatureEnv(t)
	e.fake.Seed(fakeapi.Ref{Tenant: "acme", Group: e.k.Group, Version: e.k.Version, APIPath: e.k.APIPath, Name: "azure"}, map[string]any{"metadata": map[string]any{"name": "azure"}, "spec": map[string]any{}})
	// When importing it.
	resp := importDirect(t, e, "azure")
	// Then the import succeeds without the hint.
	assertDiagnostics(t, resp.Diagnostics)
}

func TestImport_whenOtherKindMakesNoRequest(t *testing.T) {
	// Given kinds and names that never get the feature hint.
	for _, tc := range []struct{ kind, id string }{{"Workspace", "example"}, {"Integration", "wayfinder.azure"}, {"Integration", "ws/example"}} {
		t.Run(tc.kind+"/"+tc.id, func(t *testing.T) {
			e := newTestEnv(t, tc.kind)
			if tc.kind == "Integration" {
				e = newFeatureEnv(t)
			}
			// When importing.
			resp := importDirect(t, e, tc.id)
			// Then no request is made and the import succeeds.
			assertDiagnostics(t, resp.Diagnostics)

			if n := len(e.fake.Requests()); n != 0 {
				t.Fatalf("requests = %d", n)
			}
		})
	}
}
