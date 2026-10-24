// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package webhookrotation_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/webhookrotation"
)

func rotatedModel() model {
	return model{
		ID: types.StringValue("hook"), Name: types.StringValue("hook"),
		Webhook: types.StringValue("hook"), Workspace: types.StringNull(),
		Triggers:   types.MapValueMust(types.StringType, map[string]attr.Value{"n": types.StringValue("1")}),
		RotationID: types.StringValue("0123456789abcdef0123456789abcdef"),
		RotatedAt:  types.StringValue("2026-10-24T10:00:00.123456789Z"),
	}
}

func TestRead_whenWebhookExists(t *testing.T) {
	t.Parallel()
	// Given a rotation of an existing webhook.
	e := newEnv(t)
	e.seedWebhook("", "hook")
	r, s := e.resource(t)
	prior := stateOf(t, s, rotatedModel())

	// When it is read.
	resp := resource.ReadResponse{State: prior}
	r.Read(t.Context(), resource.ReadRequest{State: prior}, &resp)

	// Then state is unchanged and only the webhook object is fetched.
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	if !resp.State.Raw.Equal(prior.Raw) {
		t.Fatalf("state = %s, want unchanged %s", resp.State.Raw, prior.Raw)
	}

	requests := e.fake.Requests()
	if len(requests) != 1 || requests[0].Method != http.MethodGet || requests[0].Path != e.item(t, "", "hook") {
		t.Fatalf("requests = %+v, want one GET of the webhook", requests)
	}
}

func TestRead_whenWebhookMissing(t *testing.T) {
	t.Parallel()
	// Given a rotation whose webhook was deleted.
	e := newEnv(t)
	r, s := e.resource(t)
	prior := stateOf(t, s, rotatedModel())

	// When it is read.
	resp := resource.ReadResponse{State: prior}
	r.Read(t.Context(), resource.ReadRequest{State: prior}, &resp)

	// Then the resource is removed from state without rotating.
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	if !resp.State.Raw.IsNull() {
		t.Fatalf("state = %s, want removed", resp.State.Raw)
	}

	if got := e.fake.CountRequests(http.MethodPost, ""); got != 0 {
		t.Fatalf("POST requests = %d, want 0", got)
	}
}

func TestDelete_whenRotated(t *testing.T) {
	t.Parallel()
	// Given a rotation in state.
	e := newEnv(t)
	r, s := e.resource(t)
	prior := stateOf(t, s, rotatedModel())

	// When it is destroyed.
	resp := resource.DeleteResponse{State: prior}
	r.Delete(t.Context(), resource.DeleteRequest{State: prior}, &resp)

	// Then no API request is made, so the current URL keeps working.
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	if requests := e.fake.Requests(); len(requests) != 0 {
		t.Fatalf("requests = %+v, want none", requests)
	}
}

func TestImport_whenIDValid(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ id, workspace string }{{"hook", ""}, {"team/hook", "team"}} {
		t.Run(tc.id, func(t *testing.T) {
			t.Parallel()
			// Given an import ID.
			e := newEnv(t)
			r, s := e.resource(t)

			// When it is imported.
			resp := resource.ImportStateResponse{State: nullState(s)}
			r.(resource.ResourceWithImportState).ImportState(t.Context(), resource.ImportStateRequest{ID: tc.id}, &resp)

			// Then the webhook is adopted without rotation details and without any request.
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}

			want := stateOf(t, s, model{
				ID: types.StringValue(tc.id), Name: types.StringValue("hook"), Webhook: types.StringValue("hook"),
				Workspace: optional(tc.workspace), Triggers: types.MapNull(types.StringType),
				RotationID: types.StringNull(), RotatedAt: types.StringNull(),
			})
			if !resp.State.Raw.Equal(want.Raw) {
				t.Fatalf("state = %s, want %s", resp.State.Raw, want.Raw)
			}

			if requests := e.fake.Requests(); len(requests) != 0 {
				t.Fatalf("requests = %+v, want none", requests)
			}
		})
	}
}

func TestImport_whenIDInvalid(t *testing.T) {
	t.Parallel()

	for _, id := range []string{"", "a/b/c", "/hook", "team/", ".", "team/..", `a\b`} {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			// Given a malformed import ID.
			e := newEnv(t)
			r, s := e.resource(t)

			// When it is imported.
			resp := resource.ImportStateResponse{State: nullState(s)}
			r.(resource.ResourceWithImportState).ImportState(t.Context(), resource.ImportStateRequest{ID: id}, &resp)

			// Then import fails without state.
			if !strings.Contains(diagText(resp.Diagnostics), "Invalid import ID") || !resp.State.Raw.IsNull() {
				t.Fatalf("diagnostics = %s, state = %s; want invalid import ID and no state", diagText(resp.Diagnostics), resp.State.Raw)
			}
		})
	}
}

func TestConfigure_whenProviderDataVaries(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		data    any
		wantErr bool
	}{{"nil", nil, false}, {"wrong_type", "client", true}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var resp resource.ConfigureResponse
			webhookrotation.NewResource().(resource.ResourceWithConfigure).Configure(t.Context(),
				resource.ConfigureRequest{ProviderData: tc.data}, &resp)

			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Fatalf("diagnostics = %s, want error %t", diagText(resp.Diagnostics), tc.wantErr)
			}
		})
	}
}

func TestSegmentValidator_whenValueVaries(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		value   types.String
		wantErr bool
	}{
		{"name", types.StringValue("hook-1"), false},
		{"null", types.StringNull(), false},
		{"unknown", types.StringUnknown(), false},
		{"empty", types.StringValue(""), true},
		{"dot", types.StringValue("."), true},
		{"dotdot", types.StringValue(".."), true},
		{"slash", types.StringValue("a/b"), true},
		{"backslash", types.StringValue(`a\b`), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var resp validator.StringResponse
			webhookrotation.SegmentValidator.ValidateString(t.Context(),
				validator.StringRequest{Path: path.Root("webhook"), ConfigValue: tc.value}, &resp)

			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Fatalf("diagnostics = %s, want error %t", diagText(resp.Diagnostics), tc.wantErr)
			}
		})
	}
}
