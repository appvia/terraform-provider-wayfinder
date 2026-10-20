// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"regexp"
	"testing"

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
