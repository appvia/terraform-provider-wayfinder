// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestUpdate_whenParamRemoved(t *testing.T) {
	for _, tc := range []struct {
		name string
		keep int
		post int
	}{
		{name: "removal_only", keep: 1},
		{name: "changed_and_removed", keep: 2, post: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given two managed keys and an unrelated key owned outside Terraform.
			e := newEnv(t)
			e.fake.RecordBodies(true)
			e.fake.Seed(featureRef("azure"), featureObject("azure", true, map[string]any{"outside": "preserved"}))

			var generation float64

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: e.factories,
				Steps: []resource.TestStep{
					{Config: featureHCL("azure", `params = { keep = jsonencode(1), drop = jsonencode(false) }`)},
					{
						// When removing one key, optionally changing the retained value too.
						PreConfig: func() {
							obj, _ := e.fake.Object(featureRef("azure"))
							generation, _ = obj["metadata"].(map[string]any)["generation"].(json.Number).Float64()

							e.fake.ResetRequests()
						},
						Config: featureHCL("azure", fmt.Sprintf(`params = { keep = jsonencode(%d) }`, tc.keep)),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						// Then only the removed key disappears, using a fresh generation-bearing GET.
						Check: resource.ComposeAggregateTestCheckFunc(
							expectWrites(e, tc.post, 1),
							func(*terraform.State) error {
								obj, _ := e.fake.Object(featureRef("azure"))
								want := map[string]any{"outside": "preserved", "keep": json.Number(fmt.Sprint(tc.keep))}

								if got := obj["spec"].(map[string]any)["params"]; !reflect.DeepEqual(got, want) {
									return fmt.Errorf("stored params = %v, want %v", got, want)
								}

								requests := e.fake.Requests()
								for i, req := range requests {
									if req.Method != http.MethodPut {
										continue
									}

									if i == 0 || requests[i-1].Method != http.MethodGet || requests[i-1].Path != req.Path {
										return fmt.Errorf("PUT did not follow a fresh resource GET: %+v", requests)
									}

									var sent map[string]any
									if err := json.Unmarshal(req.Body, &sent); err != nil {
										return err
									}

									if got := sent["metadata"].(map[string]any)["generation"]; got != generation+float64(tc.post) {
										return fmt.Errorf("PUT generation = %v, want %v", got, generation+float64(tc.post))
									}
								}

								return nil
							},
						),
					},
				},
			})
		})
	}
}

func TestUpdate_whenParamRemovedAndOutsideIntegerExceedsFloat64Precision(t *testing.T) {
	// Given a managed key and an unrelated integer above 2^53 owned outside Terraform.
	e := newEnv(t)
	e.fake.RecordBodies(true)
	e.fake.Seed(featureRef("azure"), featureObject("azure", true, map[string]any{"outside": int64(9007199254740993)}))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{
			{Config: featureHCL("azure", `params = { keep = jsonencode(1), drop = jsonencode(false) }`)},
			{
				// When removing one managed key.
				PreConfig: e.fake.ResetRequests,
				Config:    featureHCL("azure", `params = { keep = jsonencode(1) }`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				// Then the single PUT re-sends the unmanaged integer with its exact digits.
				Check: resource.ComposeAggregateTestCheckFunc(
					expectWrites(e, 0, 1),
					func(*terraform.State) error {
						for _, req := range e.fake.Requests() {
							if req.Method != http.MethodPut {
								continue
							}

							if !strings.Contains(string(req.Body), "9007199254740993") || strings.Contains(string(req.Body), "9007199254740992") {
								return fmt.Errorf("PUT body lost integer precision: %s", req.Body)
							}
						}

						obj, _ := e.fake.Object(featureRef("azure"))
						if got := obj["spec"].(map[string]any)["params"].(map[string]any)["outside"]; got != json.Number("9007199254740993") {
							return fmt.Errorf("stored outside = %v, want 9007199254740993", got)
						}

						return nil
					},
				),
			},
		},
	})
}

func TestUpdate_whenAllParamsRemoved(t *testing.T) {
	for _, body := range []string{"", "params = {}"} {
		t.Run(fmt.Sprintf("config_%q", body), func(t *testing.T) {
			// Given a single managed parameter.
			e := newEnv(t)
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: e.factories,
				Steps: []resource.TestStep{
					{Config: featureHCL("azure", `params = { drop = jsonencode(false) }`)},
					{
						// When removing every parameter, with either null or an empty map.
						PreConfig: e.fake.ResetRequests,
						Config:    featureHCL("azure", body),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						// Then the server has no params property, rather than a stored JSON null.
						Check: resource.ComposeAggregateTestCheckFunc(
							expectWrites(e, 0, 1),
							func(*terraform.State) error {
								obj, _ := e.fake.Object(featureRef("azure"))
								if params, ok := obj["spec"].(map[string]any)["params"]; ok {
									return fmt.Errorf("spec.params still present: %v", params)
								}

								return nil
							},
						),
					},
				},
			})
		})
	}
}

func TestUpdate_whenParamRestoredAfterRefreshOnly(t *testing.T) {
	for _, body := range []string{"", "params = {}"} {
		t.Run(fmt.Sprintf("config_%q", body), func(t *testing.T) {
			// Given a managed key and an unrelated key owned outside Terraform.
			e := newEnv(t)
			e.fake.Seed(featureRef("azure"), featureObject("azure", true, map[string]any{"outside": "preserved"}))
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: e.factories,
				Steps: []resource.TestStep{
					{Config: featureHCL("azure", `params = { drop = jsonencode(false) }`)},
					{
						PreConfig: func() {
							e.changeFeature(t, "azure", func(spec map[string]any) {
								delete(spec["params"].(map[string]any), "drop")
							})
							e.fake.ResetRequests()
						},
						RefreshState:       true,
						ExpectNonEmptyPlan: true,
						RefreshPlanChecks: resource.RefreshPlanChecks{
							PostRefresh: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(address, "params.%", "0"),
							expectWrites(e, 0, 0),
						),
					},
					{
						// When the key is restored remotely, then removed from configuration.
						PreConfig: func() {
							e.changeFeature(t, "azure", func(spec map[string]any) {
								spec["params"].(map[string]any)["drop"] = false
							})
							e.fake.ResetRequests()
						},
						Config: featureHCL("azure", body),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						// Then Terraform deletes the owned key without touching the unrelated key.
						Check: resource.ComposeAggregateTestCheckFunc(
							expectWrites(e, 0, 1),
							func(*terraform.State) error {
								obj, _ := e.fake.Object(featureRef("azure"))
								want := map[string]any{"outside": "preserved"}

								if got := obj["spec"].(map[string]any)["params"]; !reflect.DeepEqual(got, want) {
									return fmt.Errorf("stored params = %v, want %v", got, want)
								}

								return nil
							},
						),
					},
				},
			})
		})
	}
}
