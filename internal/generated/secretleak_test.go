// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

//go:build secretleakcheck

package generated_test

import (
	"fmt"
	"regexp"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

// TestSecretLeak_whenServerEchoesWriteOnlyLiteral proves the secret-in-state checks of the
// generated suite are sensitive: the fake (test-only, via its status script) copies the
// fixture's write-only literal into status.message, an ordinary persisted attribute that also
// lands in status_json. Every check must then fail; afterwards the status is restored and the
// normal case, with its _wo-is-null checks, must pass. A _wo attribute cannot carry the leak:
// the framework nulls write-only values in responses (R17).
func TestSecretLeak_whenServerEchoesWriteOnlyLiteral(t *testing.T) {
	// Given the webhook fixture, whose security.secret_wo literal is unique.
	i := slices.IndexFunc(generatedCases, func(c generatedCase) bool { return c.Type == "wayfinder_webhook" })
	if i < 0 || len(generatedCases[i].Secrets) != 1 {
		t.Fatal("wayfinder_webhook fixture with one write-only literal not generated")
	}
	c := generatedCases[i]
	leaked := func(where string) *regexp.Regexp {
		return regexp.MustCompile(`write-only literal "` + regexp.QuoteMeta(c.Secrets[0]) + `" found in ` + where)
	}
	// When the server echoes the literal after the first apply.
	runCase(t, c, func(e caseEnv, cfg string) []resource.TestStep {
		return []resource.TestStep{
			{PreConfig: func() {
				e.fake.SetStatusScript(e.ref, []fakeapi.StatusStep{{Status: "Success", Message: c.Secrets[0]}})
				t.Logf("leak injected: fake status.message = %q", c.Secrets[0])
			}, RefreshState: true, Check: expectDetected(t, leaked("flatmap state"), secretsAbsentFlatmap(c.Secrets))},
			// Then the flatmap, state JSON and plan JSON checks each detect it.
			{Config: cfg, ConfigStateChecks: []statecheck.StateCheck{secretsAbsentState(c.Secrets)}, ExpectError: leaked("state JSON")},
			{Config: cfg, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{secretsAbsentPlan(c.Secrets)}}, ExpectError: leaked("plan JSON")},
			{PreConfig: func() {
				e.fake.SetStatusScript(e.ref, []fakeapi.StatusStep{{Status: "Success"}})
				t.Log("leak detected by flatmap, state JSON and plan JSON checks; status restored")
			}, RefreshState: true, Check: secretsAbsentFlatmap(c.Secrets)},
		}
	})
}

// expectDetected inverts check: it passes only when check reports the leak. (A RefreshState
// step fails the test directly on a Check error, so ExpectError cannot be used there.)
func expectDetected(t *testing.T, leaked *regexp.Regexp, check resource.TestCheckFunc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		err := check(s)
		if err == nil || !leaked.MatchString(err.Error()) {
			return fmt.Errorf("secret-in-state check did not detect the leak: %v", err)
		}
		t.Logf("detected as expected: %v", err)
		return nil
	}
}
