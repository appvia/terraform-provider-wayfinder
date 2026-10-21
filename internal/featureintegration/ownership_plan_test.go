// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestUpdate_whenMissingParamRemovedFromConfig(t *testing.T) {
	// Given two managed parameters.
	e := newEnv(t)
	retained := featureHCL("azure", `params = { keep = jsonencode(1) }`)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{
			{Config: featureHCL("azure", `params = { keep = jsonencode(1), drop = jsonencode(false) }`)},
			{
				// When a key disappears remotely and is then removed from configuration.
				PreConfig: func() {
					e.changeFeature(t, "azure", func(spec map[string]any) {
						delete(spec["params"].(map[string]any), "drop")
					})
					e.fake.ResetRequests()
				},
				Config: retained,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				// Then apply releases ownership without writing to the API.
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "params.%", "1"),
					expectWrites(e, 0, 0),
				),
			},
			{
				// Restoring the released key outside Terraform must not create drift.
				PreConfig: func() {
					e.changeFeature(t, "azure", func(spec map[string]any) {
						spec["params"].(map[string]any)["drop"] = true
					})
					e.fake.ResetRequests()
				},
				Config:   retained,
				PlanOnly: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					expectWrites(e, 0, 0),
					func(*terraform.State) error {
						obj, _ := e.fake.Object(featureRef("azure"))
						want := map[string]any{"keep": float64(1), "drop": true}

						if got := obj["spec"].(map[string]any)["params"]; !reflect.DeepEqual(got, want) {
							return fmt.Errorf("stored params = %v, want %v", got, want)
						}

						return nil
					},
				),
			},
		},
	})
}

func TestPlan_whenImportedWithoutParams(t *testing.T) {
	// Given an enabled feature with parameters managed outside Terraform.
	e := newEnv(t)
	e.fake.Seed(featureRef("azure"), featureObject("azure", true, map[string]any{"outside": "preserved"}))

	config := featureHCL("azure", "") + `
import {
  to = wayfinder_feature_integration.example
  id = "azure"
}
`
	// When importing without configured parameters, then both apply and later plans stabilize.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: expectWrites(e, 0, 0),
			},
			{
				PreConfig: e.fake.ResetRequests,
				Config:    config,
				PlanOnly:  true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: expectWrites(e, 0, 0),
			},
		},
	})
}
