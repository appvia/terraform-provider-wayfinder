// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestPlan_whenPersistedImportHasMatchingOptionalFields(t *testing.T) {
	// Given a status-bearing kind with the default readiness input.
	e := newTestEnv(t, "Blueprint")
	e.fake.Seed(e.ref("ws", "", "1.2.3"), map[string]any{
		"spec": map[string]any{"source": map[string]any{"repository": "https://example.com/repo", "ref": "main"}},
	})
	cfg := `resource "wayfinder_blueprint" "example" {
 name = "example"
 workspace = "ws"
 version = "1.2.3"
 source = {repository = "https://example.com/repo", ref = "main"}
}`
	// When persisting identity-only CLI import and planning with matching configuration.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: cfg, ResourceName: "wayfinder_blueprint.example", ImportState: true,
			ImportStateId: "ws/example@1.2.3", ImportStatePersist: true},
		{Config: cfg, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
	}})
	// Then the first plan is empty and no apply writes were needed to repair imported fields.
	if n := e.fake.CountRequests("POST", "") + e.fake.CountRequests("PUT", ""); n != 0 {
		t.Fatalf("adoption writes = %d", n)
	}
}
