// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package webhookrotation_test

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/webhookrotation"
)

var rotationIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

func TestCreate_whenTenantScope(t *testing.T) {
	t.Parallel()
	// Given a tenant-scoped webhook whose rotate response carries the canary URL.
	e := newEnv(t)
	e.allowRotate("", "hook")
	e.fake.RecordBodies(true)
	r, s := e.resource(t)
	before := time.Now().UTC()

	// When the rotation is created.
	resp := create(t, r, s, "", "hook")

	// Then exactly one POST rotate request is sent and the state records the rotation only.
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	requests := e.fake.Requests()
	if len(requests) != 1 || e.fake.CountRequests(http.MethodPost, e.rotatePath(t, "", "hook")) != 1 {
		t.Fatalf("requests = %+v, want one POST %s", requests, e.rotatePath(t, "", "hook"))
	}

	if got := string(requests[0].Body); got != "{}" {
		t.Fatalf("rotate request body = %q, want {}", got)
	}

	m := stateModel(t, resp.State)
	if m.ID.ValueString() != "hook" || m.Name.ValueString() != "hook" || !m.Workspace.IsNull() {
		t.Fatalf("id/name/workspace = %s/%s/%s, want hook/hook/null", m.ID, m.Name, m.Workspace)
	}

	if !rotationIDPattern.MatchString(m.RotationID.ValueString()) {
		t.Fatalf("rotation_id = %q, want 32 lowercase hex characters", m.RotationID.ValueString())
	}

	rotatedAt, err := time.Parse(time.RFC3339Nano, m.RotatedAt.ValueString())
	if err != nil || rotatedAt.Before(before) || rotatedAt.After(time.Now().UTC()) {
		t.Fatalf("rotated_at = %q (%v), want an RFC 3339 time during the create", m.RotatedAt.ValueString(), err)
	}

	if strings.Contains(resp.State.Raw.String(), canaryWord) {
		t.Fatal("state contains the rotated URL")
	}
}

func TestCreate_whenWorkspaceScope(t *testing.T) {
	t.Parallel()
	// Given a webhook in workspace team.
	e := newEnv(t)
	e.allowRotate("team", "hook")
	r, s := e.resource(t)

	// When the rotation is created.
	resp := create(t, r, s, "team", "hook")

	// Then the workspace rotate path is used and the ID carries the workspace.
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	path := e.rotatePath(t, "team", "hook")
	if !strings.Contains(path, "/workspaces/team/") || e.fake.CountRequests(http.MethodPost, path) != 1 {
		t.Fatalf("requests = %+v, want one POST %s", e.fake.Requests(), path)
	}

	if m := stateModel(t, resp.State); m.ID.ValueString() != "team/hook" || m.Workspace.ValueString() != "team" {
		t.Fatalf("id/workspace = %s/%s, want team/hook/team", m.ID, m.Workspace)
	}
}

func TestCreate_whenCreatedTwice(t *testing.T) {
	t.Parallel()
	// Given one webhook.
	e := newEnv(t)
	e.allowRotate("", "hook")
	r, s := e.resource(t)

	// When it is rotated twice in quick succession.
	first := stateModel(t, create(t, r, s, "", "hook").State).RotationID.ValueString()
	second := stateModel(t, create(t, r, s, "", "hook").State).RotationID.ValueString()

	// Then each rotation has its own rotation_id.
	if first == second || !rotationIDPattern.MatchString(first) || !rotationIDPattern.MatchString(second) {
		t.Fatalf("rotation IDs %q and %q, want two distinct 32-character hex values", first, second)
	}

	if got := e.fake.CountRequests(http.MethodPost, e.rotatePath(t, "", "hook")); got != 2 {
		t.Fatalf("rotate requests = %d, want 2", got)
	}
}

// Not parallel: it swaps the package-level rotation ID source.
func TestCreate_whenRotationIDFails(t *testing.T) {
	// Given a rotation ID source that cannot read randomness.
	webhookrotation.SetNewRotationID(t, func() (string, error) { return "", errors.New("entropy unavailable") })
	e := newEnv(t)
	e.allowRotate("", "hook")
	r, s := e.resource(t)

	// When the rotation is created.
	resp := create(t, r, s, "", "hook")

	// Then create fails without saving the resource.
	if !strings.Contains(diagText(resp.Diagnostics), "Unable to generate rotation ID") {
		t.Fatalf("diagnostics = %s, want rotation ID error", diagText(resp.Diagnostics))
	}

	if !resp.State.Raw.IsNull() {
		t.Fatalf("state = %s, want none", resp.State.Raw)
	}
}

func TestCreate_whenRotateFails(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		status int
		want   []string
	}{
		{"not_found", http.StatusNotFound, []string{"Webhook not found", `"hook"`}},
		{"forbidden", http.StatusForbidden, []string{"Permission denied", "webhooks/rotate"}},
		{"server_error", http.StatusInternalServerError, []string{"Wayfinder request failed", "returned HTTP 500"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given a rotate error whose body holds a secret.
			e := newEnv(t)
			e.answerRotate("", "hook", tc.status, map[string]string{"message": "secret-body", "url": leakCanary})
			r, s := e.resource(t)

			// When the rotation is created.
			resp := create(t, r, s, "", "hook")

			// Then the diagnostic is specific, omits the body and nothing is saved.
			text := diagText(resp.Diagnostics)
			for _, want := range tc.want {
				if !strings.Contains(text, want) {
					t.Errorf("diagnostics = %s, want %q", text, want)
				}
			}

			if strings.Contains(text, "secret-body") || strings.Contains(text, canaryWord) {
				t.Errorf("diagnostics leak the response body: %s", text)
			}

			if !resp.State.Raw.IsNull() {
				t.Errorf("state = %s, want none", resp.State.Raw)
			}

			if got := e.fake.CountRequests(http.MethodPost, ""); got != 1 {
				t.Errorf("POST requests = %d, want 1 (no retry)", got)
			}
		})
	}
}
