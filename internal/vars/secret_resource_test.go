// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package vars_test

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

const (
	secretV1 = "s3cr3t-literal-v1"
	secretV2 = "s3cr3t-literal-v2"
	secretV3 = "s3cr3t-literal-v3"
)

func secretHCL(s fakeapi.VarScope, value, version string) string {
	return fmt.Sprintf("resource \"wayfinder_secret\" \"example\" {\n  key              = \"db_password\"\n%s  value_wo         = jsonencode(%q)\n  value_wo_version = %q\n}\n",
		scopeHCL(s), value, version)
}

func expectSecret(e env, s fakeapi.VarScope, want string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		got, ok := e.fake.Secret(s, "db_password")
		if !ok {
			return fmt.Errorf("secret not stored in %+v", s)
		}

		if !jsonEqual(got, fmt.Sprintf("%q", want)) {
			return fmt.Errorf("stored secret does not match the expected value")
		}

		return nil
	}
}

func expectPuts(e env, want int) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if n := e.fake.CountRequests(http.MethodPut, ""); n != want {
			return fmt.Errorf("PUT requests = %d, want %d", n, want)
		}

		return nil
	}
}

// secretAbsent asserts every secret literal is missing from state and plan.
func secretAbsent() ([]statecheck.StateCheck, []plancheck.PlanCheck) {
	lits := []string{secretV1, secretV2, secretV3}

	var (
		sc []statecheck.StateCheck
		pc []plancheck.PlanCheck
	)

	for _, l := range lits {
		sc = append(sc, absentFromState(l))
		pc = append(pc, absentFromPlan(l))
	}

	return sc, pc
}

func TestSecret_lifecycle_at_every_scope(t *testing.T) {
	for name, scope := range testScopes {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)
			e.fake.RecordBodies(true)

			stateChecks, planChecks := secretAbsent()
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: e.factories,
				TerraformVersionChecks:   writeOnlyVersions,
				CheckDestroy: func(*terraform.State) error {
					if _, ok := e.fake.Secret(scope, "db_password"); ok {
						return fmt.Errorf("secret still present after destroy")
					}

					return nil
				},
				Steps: []resource.TestStep{
					{
						Config:            secretHCL(scope, secretV1, "1"),
						ConfigStateChecks: stateChecks,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             planChecks,
							PostApplyPostRefresh: append([]plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}, planChecks...),
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("wayfinder_secret.example", "id", scopeID(scope, "db_password")),
							resource.TestCheckNoResourceAttr("wayfinder_secret.example", "value_wo"),
							expectSecret(e, scope, secretV1),
						),
					},
					{
						// Changing value_wo alone is not a change: no API write.
						PreConfig:         e.fake.ResetRequests,
						Config:            secretHCL(scope, secretV2, "1"),
						ConfigStateChecks: stateChecks,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: append([]plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}, planChecks...),
						},
						Check: resource.ComposeAggregateTestCheckFunc(expectPuts(e, 0), expectSecret(e, scope, secretV1)),
					},
					{
						// Bumping value_wo_version sends value_wo exactly once.
						PreConfig:         e.fake.ResetRequests,
						Config:            secretHCL(scope, secretV3, "2"),
						ConfigStateChecks: stateChecks,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             append([]plancheck.PlanCheck{plancheck.ExpectResourceAction("wayfinder_secret.example", plancheck.ResourceActionUpdate)}, planChecks...),
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							expectPuts(e, 1), expectSecret(e, scope, secretV3), expectPutBody(t, e, secretV3),
						),
					},
					{
						ResourceName:                         "wayfinder_secret.example",
						ImportState:                          true,
						ImportStateId:                        scopeID(scope, "db_password"),
						ImportStateVerify:                    true,
						ImportStateVerifyIdentifierAttribute: "id",
						ImportStateVerifyIgnore:              []string{"value_wo", "value_wo_version"},
					},
					{
						PreConfig:          func() { deleteDirect(t, e.fake, varsBase(scope)+"/secrets/db_password") },
						Config:             secretHCL(scope, secretV3, "2"),
						PlanOnly:           true,
						ExpectNonEmptyPlan: true,
					},
				},
			})
		})
	}
}

// expectPutBody checks the single logged PUT carried the rotated value, and logs the request
// log (paths and statuses only) as evidence.
func expectPutBody(t *testing.T, e env, want string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		for _, r := range e.fake.Requests() {
			t.Logf("request log: %s %s -> %d", r.Method, r.Path, r.Status)

			if r.Method == http.MethodPut && !strings.Contains(string(r.Body), want) {
				return fmt.Errorf("PUT body does not carry the rotated value")
			}
		}

		return nil
	}
}

func TestSecret_create_refuses_existing_key_without_writing(t *testing.T) {
	e := newEnv(t)
	scope := testScopes["tenant"]
	e.fake.SetSecret(scope, "db_password", []byte(`"set-in-ui"`))
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		TerraformVersionChecks:   writeOnlyVersions,
		Steps: []resource.TestStep{{
			Config:      secretHCL(scope, secretV1, "1"),
			ExpectError: regexp.MustCompile(`(?s)Secret db_password already exists.*terraform import\s+wayfinder_secret\.<name>\s+db_password`),
		}},
	})

	for _, r := range e.fake.Requests() {
		t.Logf("request log: %s %s -> %d", r.Method, r.Path, r.Status)
	}

	if n := e.fake.CountRequests(http.MethodPut, ""); n != 0 {
		t.Errorf("PUT requests = %d, want 0", n)
	}

	got, _ := e.fake.Secret(scope, "db_password")
	if !jsonEqual(got, `"set-in-ui"`) {
		t.Error("stored secret was overwritten")
	}
}

func TestSecret_environment_without_workspace_fails_at_plan(t *testing.T) {
	e := newEnv(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		TerraformVersionChecks:   writeOnlyVersions,
		Steps: []resource.TestStep{{
			Config:      secretHCL(fakeapi.VarScope{Tenant: tenant, Environment: "dev"}, secretV1, "1"),
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`(?s)workspace.*must be specified`),
		}},
	})

	if n := len(e.fake.Requests()); n != 0 {
		t.Errorf("API requests = %d, want 0", n)
	}
}
