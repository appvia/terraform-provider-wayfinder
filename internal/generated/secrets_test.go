// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package generated_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// secretsAbsentPlan fails when a write-only literal appears in the plan JSON
// (`terraform show -json <plan>`), prior state included.
type secretsAbsentPlan []string

func (s secretsAbsentPlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	plan := *req.Plan
	// The configuration section echoes the literal expressions of the .tf files themselves,
	// which hold the fixture secrets by construction; everything Terraform computed or stored
	// (planned values, resource changes, prior state, variables) is checked.
	plan.Config = nil
	resp.Error = findSecret(plan, s, "plan JSON")
}

// secretsAbsentState fails when a write-only literal appears in the state JSON
// (`terraform show -json`).
type secretsAbsentState []string

func (s secretsAbsentState) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	resp.Error = findSecret(req.State, s, "state JSON")
}

// secretsAbsentFlatmap fails when a write-only literal appears in the testing framework's
// flatmap state (every attribute of every resource, private data excluded).
func secretsAbsentFlatmap(secrets []string) resource.TestCheckFunc {
	return func(s *terraform.State) error { return findSecret(s, secrets, "flatmap state") }
}

// importOnlyAdoptsServerKeys fails unless the plan right after a configuration-driven import
// is a no-op, or an update that changes nothing but labels/annotations (import adopts the keys
// the server added; D-DRIFT drops them), computed status, and write-only version companions
// (state written by import cannot know the configured _wo_version).
type importOnlyAdoptsServerKeys string

func (addr importOnlyAdoptsServerKeys) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	for _, rc := range req.Plan.ResourceChanges {
		if rc.Address != string(addr) || rc.Change.Actions.NoOp() {
			continue
		}
		if !rc.Change.Actions.Update() {
			resp.Error = fmt.Errorf("%s: import planned %v, want no-op or update", addr, rc.Change.Actions)
			return
		}
		before, _ := rc.Change.Before.(map[string]any)
		after, _ := rc.Change.After.(map[string]any)
		for k := range mergeKeys(before, after) {
			switch k {
			case "labels", "annotations", "status", "status_json":
				continue
			}
			if b, a := stripWriteOnly(before[k]), stripWriteOnly(after[k]); !reflect.DeepEqual(b, a) {
				resp.Error = fmt.Errorf("%s: import plans a change to %s: %v -> %v", addr, k, b, a)
				return
			}
		}
	}
}

func mergeKeys(maps ...map[string]any) map[string]bool {
	out := map[string]bool{}
	for _, m := range maps {
		for k := range m {
			out[k] = true
		}
	}
	return out
}

// stripWriteOnly drops write-only attributes and their version companions from a JSON value.
func stripWriteOnly(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, child := range t {
			if !strings.HasSuffix(k, "_wo") && !strings.HasSuffix(k, "_wo_version") {
				out[k] = stripWriteOnly(child)
			}
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, child := range t {
			out[i] = stripWriteOnly(child)
		}
		return out
	default:
		return v
	}
}
