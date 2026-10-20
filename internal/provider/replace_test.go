// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestProvider_update_whenImmutableAttributeChanged(t *testing.T) {
	// Given: an applied workspace with a slug.
	e := newTestEnv(t)

	const addr = "wayfinder_workspace.example"

	ref := e.ref(e.kind(t, "workspace"), "", "example")
	cfg := func(slug, description string) string {
		return e.providerBlock() + fmt.Sprintf("resource \"wayfinder_workspace\" \"example\" {\n name = \"example\"\n slug = %q\n description = %q\n}\n", slug, description)
	}
	plan := func(action plancheck.ResourceActionType) resource.ConfigPlanChecks {
		return resource.ConfigPlanChecks{
			PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(addr, action)},
			PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
		}
	}
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: cfg("plat", "first"), ConfigPlanChecks: resource.ConfigPlanChecks{
			PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
		}},
		// When: the immutable slug changes.
		{
			PreConfig:        e.fake.ResetRequests,
			Config:           cfg("core", "first"),
			ConfigPlanChecks: plan(plancheck.ResourceActionReplace),
			Check: func(_ *terraform.State) error {
				// Then: the object is deleted and recreated, never updated in place.
				for method, want := range map[string]int{http.MethodDelete: 1, http.MethodPost: 1, http.MethodPut: 0} {
					if got := e.fake.CountRequests(method, ""); got != want {
						return fmt.Errorf("%s count = %d, want %d", method, got, want)
					}
				}

				obj, exists := e.fake.Object(ref)
				if !exists {
					return fmt.Errorf("replaced Workspace missing")
				}

				spec, err := nestedMap(obj, "spec")
				if err != nil {
					return err
				}

				if spec["slug"] != "core" {
					return fmt.Errorf("spec.slug = %v, want core", spec["slug"])
				}

				return nil
			},
		},
		// Control: a mutable attribute change is an in-place update.
		{PreConfig: e.fake.ResetRequests, Config: cfg("core", "second"), ConfigPlanChecks: plan(plancheck.ResourceActionUpdate)},
	}})
}
