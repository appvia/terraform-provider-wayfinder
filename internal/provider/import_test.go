// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestProvider_plan_whenImportedConfigMatchesServer(t *testing.T) {
	// Given a KubeResourcePlan with a nested optional chart version and configured server metadata.
	e := newTestEnv(t)
	ref := e.ref(e.kind(t, "kube_resource_plan"), "ws", "example")
	ref.ObjectVersion = "1.2.3"
	e.fake.Seed(ref, map[string]any{
		"metadata": map[string]any{"labels": map[string]any{"server": "yes"}, "annotations": map[string]any{"controller": "yes"}},
		"spec":     map[string]any{"helm": map[string]any{"chartName": "podinfo", "chartVersion": "6.0.0"}},
	})
	cfg := e.providerBlock() + `
resource "wayfinder_kube_resource_plan" "example" {
 name = "example"
 workspace = "ws"
 version = "1.2.3"
 labels = {server = "yes"}
 annotations = {controller = "yes"}
 helm = {chart_name = "podinfo", chart_version = "6.0.0"}
}
`
	// When importing through configuration and immediately planning the matching config.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: cfg + "import {\n to = wayfinder_kube_resource_plan.example\n id = \"ws/example@1.2.3\"\n}\n"},
		{Config: cfg, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
	}})
	// Then the first config plan after the import step is empty (checked before apply).
}

func TestProvider_refresh_whenServerAddsUnconfiguredDefault(t *testing.T) {
	// Given a Terraform-created Integration with no configured enabled flag.
	e := newTestEnv(t)
	ref := e.ref(e.kind(t, "integration"), "ws", "example")
	cfg := e.providerBlock() + `
resource "wayfinder_integration" "example" {
 name = "example"
 workspace = "ws"
 description = "Chat notifications"
}
`
	// When an out-of-band edit adds a default before an ordinary refresh.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: cfg},
		{PreConfig: func() {
			e.fake.Mutate(ref, func(obj map[string]any) {
				obj["spec"].(map[string]any)["enabled"] = true
			})
		}, RefreshState: true, Check: resource.TestCheckNoResourceAttr("wayfinder_integration.example", "enabled")},
		{Config: cfg, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
	}})
	// Then Terraform did not adopt the default or send an update.
	if n := e.fake.CountRequests("PUT", ""); n != 0 {
		t.Fatalf("refresh PUT = %d", n)
	}
}

func TestProvider_plan_whenImportedServerStampedAnnotations(t *testing.T) {
	// Given a server object with server-stamped annotations and a user-set label, and config naming only the label.
	e := newTestEnv(t)
	ref := e.ref(e.kind(t, "kube_resource_plan"), "ws", "example")
	ref.ObjectVersion = "1.2.3"
	e.fake.Seed(ref, map[string]any{
		"metadata": map[string]any{
			"labels":      map[string]any{"team": "x"},
			"annotations": map[string]any{"appvia.io/createdBy": "u", "appvia.io/lastUpdated": "t", "appvia.io/lastUpdatedBy": "u"},
		},
		"spec": map[string]any{"helm": map[string]any{"chartName": "podinfo"}},
	})
	cfg := e.providerBlock() + `
resource "wayfinder_kube_resource_plan" "example" {
 name = "example"
 workspace = "ws"
 version = "1.2.3"
 labels = {team = "x"}
 helm = {chart_name = "podinfo"}
}
`
	// When classic-importing (persisting state) and planning.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: cfg, ResourceName: "wayfinder_kube_resource_plan.example", ImportState: true, ImportStateId: "ws/example@1.2.3", ImportStatePersist: true,
			ImportStateCheck: func(s []*terraform.InstanceState) error {
				if v := s[0].Attributes["labels.team"]; v != "x" {
					return fmt.Errorf("labels.team = %q", v)
				}
				return nil
			}},
		// Then user label was adopted and the plan is empty (no annotation removal).
		{Config: cfg, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
	}})
}
