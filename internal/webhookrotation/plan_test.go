// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package webhookrotation_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func expectRotations(t *testing.T, e env, want int) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if got := e.fake.CountRequests(http.MethodPost, e.rotatePath(t, "", "hook")); got != want {
			return fmt.Errorf("rotate requests = %d, want %d; log: %+v", got, want, e.fake.Requests())
		}

		return nil
	}
}

func TestPlan_whenTriggersChange(t *testing.T) {
	t.Parallel()
	// Given an existing webhook whose rotate response carries the canary URL.
	e := newEnv(t)
	e.seedWebhook("", "hook")
	e.allowRotate("", "hook")

	rotationIDs := statecheck.CompareValue(compare.ValuesDiffer())
	unknownOutputs := []plancheck.PlanCheck{
		plancheck.ExpectUnknownValue(address, tfjsonpath.New("name")),
		plancheck.ExpectUnknownValue(address, tfjsonpath.New("rotation_id")),
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{
			{
				// When the rotation is created, then name and rotation_id are unknown until apply,
				// one rotation happens and refresh plans nothing.
				Config: rotationHCL("1"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: append([]plancheck.PlanCheck{
						plancheck.ExpectResourceAction(address, plancheck.ResourceActionCreate),
					}, unknownOutputs...),
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(address, tfjsonpath.New("id"), knownvalue.StringExact("hook")),
					statecheck.ExpectKnownValue(address, tfjsonpath.New("name"), knownvalue.StringExact("hook")),
					statecheck.ExpectKnownValue(address, tfjsonpath.New("rotation_id"),
						knownvalue.StringRegexp(regexp.MustCompile(`^[0-9a-f]{32}$`))),
					rotationIDs.AddStateValue(address, tfjsonpath.New("rotation_id")),
				},
				Check: expectRotations(t, e, 1),
			},
			{
				// When a trigger changes, then the rotation is replaced, rotates once more and
				// gets a new rotation_id.
				Config: rotationHCL("2"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: append([]plancheck.PlanCheck{
						plancheck.ExpectResourceAction(address, plancheck.ResourceActionReplace),
					}, unknownOutputs...),
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{rotationIDs.AddStateValue(address, tfjsonpath.New("rotation_id"))},
				Check:             expectRotations(t, e, 2),
			},
			{
				// When the webhook is imported, then the identity matches and nothing rotates.
				ResourceName:            address,
				ImportState:             true,
				ImportStateId:           "hook",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"rotation_id", "rotated_at", "triggers"},
			},
		},
	})

	// Then import, refreshes and destroy added no rotations.
	if err := expectRotations(t, e, 2)(nil); err != nil {
		t.Fatal(err)
	}
}
