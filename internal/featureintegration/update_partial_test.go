// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

func TestUpdate_whenRemovalFailsAfterEnableAndConfigRollsBack(t *testing.T) {
	// Given a managed required key and an unrelated parameter.
	e := newEnv(t)
	e.fake.RegisterFeature(fakeapi.Feature{Name: "azure", RequiredParams: []string{"old"}})
	e.fake.Seed(featureRef("azure"), featureObject("azure", true, map[string]any{"outside": "preserved"}))

	original := featureHCL("azure", `params = { old = jsonencode("required") }`)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{
			{Config: original},
			{
				// When enable adds a key but removing the required key fails.
				PreConfig:   e.fake.ResetRequests,
				Config:      featureHCL("azure", `params = { new = jsonencode(false) }`),
				ExpectError: regexp.MustCompile(`(?s).+`),
			},
			{
				PreConfig: func() {
					if err := expectWrites(e, 1, 1)(nil); err != nil {
						t.Fatal(err)
					}

					obj, _ := e.fake.Object(featureRef("azure"))
					want := map[string]any{"old": "required", "new": false, "outside": "preserved"}

					if got := obj["spec"].(map[string]any)["params"]; !reflect.DeepEqual(got, want) {
						t.Fatalf("partially stored params = %v, want %v", got, want)
					}

					e.fake.ResetRequests()
				},
				Config:             original,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					// Then even without Read, Terraform saved the errored Update's state.
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				// And a rollback apply removes the added key and stabilizes after Read.
				PreConfig: e.fake.ResetRequests,
				Config:    original,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "params.%", "1"),
					resource.TestCheckResourceAttr(address, "params.old", `"required"`),
					expectWrites(e, 0, 1),
					func(*terraform.State) error {
						obj, _ := e.fake.Object(featureRef("azure"))
						want := map[string]any{"old": "required", "outside": "preserved"}

						if got := obj["spec"].(map[string]any)["params"]; !reflect.DeepEqual(got, want) {
							return fmt.Errorf("stored params after rollback = %v, want %v", got, want)
						}

						return nil
					},
				),
			},
		},
	})
}
