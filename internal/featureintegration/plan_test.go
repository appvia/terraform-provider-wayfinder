// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestPlan_whenDisabledOutOfBand(t *testing.T) {
	// Given an enabled feature tracked in Terraform.
	e := newEnv(t)
	config := featureHCL("azure", "")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				// When an HTTP update disables it outside Terraform.
				PreConfig: func() {
					e.changeFeature(t, "azure", func(spec map[string]any) { spec["enabled"] = false })
					e.fake.ResetRequests()
				},
				Config: config,
				// Then an in-place update re-enables it and the next plan is empty.
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					expectEnabled(e, "azure", true),
					expectWrites(e, 1, 0),
				),
			},
		},
	})
}
