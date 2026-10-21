// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestUpdate_whenParamChanged(t *testing.T) {
	// Given an enabled feature with a configured boolean parameter.
	e := newEnv(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{
			{Config: featureHCL("ai", `params = { enable_wayfinder = jsonencode(false) }`)},
			{
				// When changing the configured value.
				PreConfig: e.fake.ResetRequests,
				Config:    featureHCL("ai", `params = { enable_wayfinder = jsonencode(true) }`),
				// Then an in-place update enables with the new parameter and stabilizes.
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "params.enable_wayfinder", "true"),
					expectWrites(e, 1, 0),
				),
			},
		},
	})

	if got := e.fake.CountRequests(http.MethodPost, "/api/v3/general/tenants/acme/integrations/ai/enable"); got != 1 {
		t.Errorf("enable requests = %d, want 1", got)
	}
}
