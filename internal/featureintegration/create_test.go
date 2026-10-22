// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestCreate_whenFeatureNotEnabled(t *testing.T) {
	// Given a feature that has not been enabled.
	e := newEnv(t)
	// When applying the feature resource.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		CheckDestroy:             expectEnabled(e, "azure", false),
		Steps: []resource.TestStep{{
			Config: featureHCL("azure", ""),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			},
			// Then enable returns its identity and leaves an empty plan.
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(address, "id", "wayfinder.azure"),
				resource.TestCheckResourceAttr(address, "enabled", "true"),
				resource.TestCheckResourceAttr(address, "type", "feature"),
				resource.TestCheckResourceAttr(address, "keep_enabled_on_destroy", "false"),
				expectEnabled(e, "azure", true),
			),
		}},
	})

	if got := e.fake.CountRequests(http.MethodPost, "/api/v3/general/tenants/acme/integrations/azure/enable"); got != 1 {
		t.Errorf("enable requests = %d, want 1", got)
	}

	if got := e.fake.CountRequests(http.MethodDelete, ""); got != 0 {
		t.Errorf("DELETE requests = %d, want 0", got)
	}
}

func TestCreate_whenParamsSet(t *testing.T) {
	// Given JSON values with significant types and insignificant formatting.
	e := newEnv(t)
	config := featureHCL("ai", `params = {
 enable_wayfinder = jsonencode(false)
 nullable = jsonencode(null)
 options = jsonencode({
   z = [1, true, null]
   a = "value"
 })
}`)
	// When applying and then configuring semantically identical JSON.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "params.enable_wayfinder", "false"),
					func(*terraform.State) error {
						obj, _ := e.fake.Object(featureRef("ai"))
						got := obj["spec"].(map[string]any)["params"]
						want := map[string]any{"enable_wayfinder": false, "nullable": nil, "options": map[string]any{
							"z": []any{json.Number("1"), true, nil}, "a": "value",
						}}

						if !reflect.DeepEqual(got, want) {
							return fmt.Errorf("stored params = %v, want %v", got, want)
						}

						return nil
					},
				),
			},
			{
				Config: featureHCL("ai", `params = {
 enable_wayfinder = jsonencode(false)
 nullable = jsonencode(null)
 options = jsonencode({ a = "value", z = [1, true, null] })
}`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestCreate_whenAlreadyEnabledOutOfBand(t *testing.T) {
	// Given an enabled feature with a parameter Terraform does not manage.
	e := newEnv(t)
	e.fake.Seed(featureRef("azure"), featureObject("azure", true, map[string]any{"outside": "preserved"}))
	// When enabling it through Terraform.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{{
			Config: featureHCL("azure", `params = { owned = jsonencode("configured") }`),
			// Then enable merges parameters without adopting the outside key.
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(address, "params.%", "1"),
				resource.TestCheckResourceAttr(address, "params.owned", `"configured"`),
				func(*terraform.State) error {
					obj, _ := e.fake.Object(featureRef("azure"))
					got := obj["spec"].(map[string]any)["params"]

					if want := map[string]any{"outside": "preserved", "owned": "configured"}; !reflect.DeepEqual(got, want) {
						return fmt.Errorf("merged params = %v, want %v", got, want)
					}

					return nil
				},
			),
		}},
	})
}
