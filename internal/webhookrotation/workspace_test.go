// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package webhookrotation_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCreate_whenWorkspaceResolvesInvalid(t *testing.T) {
	t.Parallel()

	for _, workspace := range []string{"", ".", "..", "a/b", `a\b`} {
		t.Run(workspace, func(t *testing.T) {
			t.Parallel()
			// Given a non-null workspace resolved at apply and a tenant webhook of the same name.
			e := newEnv(t)
			e.allowRotate("", "hook")
			r, s := e.resource(t)
			m := rotatedModel()
			m.Workspace = types.StringValue(workspace)
			m.ID, m.Name = types.StringUnknown(), types.StringUnknown()
			m.RotationID, m.RotatedAt = types.StringUnknown(), types.StringUnknown()
			plan := tfsdk.Plan{Schema: s, Raw: stateOf(t, s, m).Raw}

			// When Create receives the resolved plan, bypassing plan-time validation.
			resp := resource.CreateResponse{State: nullState(s)}
			r.Create(t.Context(), resource.CreateRequest{Plan: plan}, &resp)

			// Then it rejects the workspace without saving state or contacting either scope.
			if !resp.Diagnostics.HasError() {
				t.Error("want an error for the invalid resolved workspace")
			}

			if !resp.State.Raw.IsNull() {
				t.Errorf("state = %s, want none", resp.State.Raw)
			}

			if requests := e.fake.Requests(); len(requests) != 0 {
				t.Errorf("requests = %+v, want none", requests)
			}
		})
	}
}

func TestRead_whenWorkspaceResolvesInvalid(t *testing.T) {
	t.Parallel()

	for _, workspace := range []string{"", ".", "..", "a/b", `a\b`} {
		t.Run(workspace, func(t *testing.T) {
			t.Parallel()
			// Given a stored non-null invalid workspace and a tenant webhook of the same name.
			e := newEnv(t)
			e.seedWebhook("", "hook")
			r, s := e.resource(t)
			m := rotatedModel()
			m.Workspace = types.StringValue(workspace)
			prior := stateOf(t, s, m)

			// When Read receives the state directly.
			resp := resource.ReadResponse{State: prior}
			r.Read(t.Context(), resource.ReadRequest{State: prior}, &resp)

			// Then it rejects the workspace without contacting either scope or changing state.
			if !resp.Diagnostics.HasError() {
				t.Error("want an error for the invalid stored workspace")
			}

			if !resp.State.Raw.Equal(prior.Raw) {
				t.Errorf("state = %s, want unchanged %s", resp.State.Raw, prior.Raw)
			}

			if requests := e.fake.Requests(); len(requests) != 0 {
				t.Errorf("requests = %+v, want none", requests)
			}
		})
	}
}
