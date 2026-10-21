// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/appvia/terraform-provider-wayfinder/internal/featureintegration"
)

func TestRead_whenIntegrationDeleted(t *testing.T) {
	// Given a feature tracked in Terraform.
	e := newEnv(t)
	config := featureHCL("azure", "")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				// When the object is deleted outside Terraform and refreshed.
				PreConfig: func() {
					if err := e.client.Delete(t.Context(), featureintegration.IntegrationRef("azure")); err != nil {
						t.Fatal(err)
					}
				},
				Config: config,
				// Then refresh removes it from state so Terraform plans to create it.
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionCreate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestRead_whenJSONFormattingDiffers(t *testing.T) {
	// Given configured JSON whose whitespace and key order differ from the API response.
	e := newEnv(t)
	// When enabling and refreshing it.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{{
			Config: featureHCL("azure", `params = { options = "{ \"z\": true, \"a\": 1 }" }`),
			// Then semantic equality retains the configured value without a perpetual diff.
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			},
		}},
	})
}

func TestRead_whenParamChangedOutOfBand(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprintf("missing_%t", missing), func(t *testing.T) {
			// Given a managed parameter on an enabled feature.
			e := newEnv(t)
			config := featureHCL("ai", `params = { enable_wayfinder = jsonencode(false) }`)
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: e.factories,
				Steps: []resource.TestStep{
					{Config: config},
					{
						// When an out-of-band HTTP update changes or removes it.
						PreConfig: func() {
							e.changeFeature(t, "ai", func(spec map[string]any) {
								params := spec["params"].(map[string]any)
								if missing {
									delete(params, "enable_wayfinder")
								} else {
									params["enable_wayfinder"] = true
								}
							})
							e.fake.ResetRequests()
						},
						Config: config,
						// Then apply restores the configured value and leaves an empty plan.
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(address, "params.enable_wayfinder", "false"),
							expectWrites(e, 1, 0),
						),
					},
				},
			})
		})
	}
}
