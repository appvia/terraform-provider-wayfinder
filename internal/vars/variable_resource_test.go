// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package vars_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

func variableHCL(s fakeapi.VarScope, value string) string {
	return fmt.Sprintf("resource \"wayfinder_variable\" \"example\" {\n  key   = \"region\"\n%s  value = %s\n}\n", scopeHCL(s), value)
}

func expectVariable(e env, s fakeapi.VarScope, want string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		got, ok := e.fake.Variable(s, "region")
		if !ok {
			return fmt.Errorf("variable not stored in %+v", s)
		}

		if !jsonEqual(got, want) {
			return fmt.Errorf("stored value = %s, want %s", got, want)
		}

		return nil
	}
}

// TestVariable_lifecycle_at_every_scope covers every scope, including instance. IS-8: adding
// the Optional instance attribute without a schema version bump means Terraform core decodes
// prior state with instance = null, so the tenant, workspace and environment cases running
// unchanged with their empty-plan steps prove existing configurations plan with no diff; no
// state upgrader is added.
func TestVariable_lifecycle_at_every_scope(t *testing.T) {
	for name, scope := range testScopes {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: e.factories,
				CheckDestroy: func(*terraform.State) error {
					if _, ok := e.fake.Variable(scope, "region"); ok {
						return fmt.Errorf("variable still present after destroy")
					}

					return nil
				},
				Steps: []resource.TestStep{
					{
						Config: variableHCL(scope, `jsonencode({ region = "eu-west-2", zones = ["a", "b"], replicas = 3 })`),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("wayfinder_variable.example", "id", scopeID(scope, "region")),
							expectVariable(e, scope, `{"region":"eu-west-2","zones":["a","b"],"replicas":3}`),
						),
					},
					{
						Config: variableHCL(scope, `jsonencode("eu-west-1")`),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction("wayfinder_variable.example", plancheck.ResourceActionUpdate)},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: expectVariable(e, scope, `"eu-west-1"`),
					},
					{
						ResourceName:      "wayfinder_variable.example",
						ImportState:       true,
						ImportStateId:     scopeID(scope, "region"),
						ImportStateVerify: true,
					},
					{
						// Deleted outside Terraform: refresh drops it and the plan recreates it.
						PreConfig:          func() { deleteDirect(t, e.fake, varsBase(scope)+"/variables/region") },
						Config:             variableHCL(scope, `jsonencode("eu-west-1")`),
						PlanOnly:           true,
						ExpectNonEmptyPlan: true,
					},
				},
			})
		})
	}
}

func TestVariable_create_refuses_existing_key_without_writing(t *testing.T) {
	e := newEnv(t)
	scope := testScopes["workspace"]
	e.fake.SetVariable(scope, "region", []byte(`"set-in-ui"`))
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{{
			Config:      variableHCL(scope, `jsonencode("from-terraform")`),
			ExpectError: regexp.MustCompile(`(?s)Variable ws1/region already exists.*terraform import\s+wayfinder_variable\.<name>\s+ws1/region`),
		}},
	})

	if n := e.fake.CountRequests(http.MethodPut, ""); n != 0 {
		t.Errorf("PUT requests = %d, want 0", n)
	}

	got, _ := e.fake.Variable(scope, "region")
	if !jsonEqual(got, `"set-in-ui"`) {
		t.Errorf("stored value changed to %s", got)
	}
}

func TestVariable_environment_without_workspace_fails_at_plan(t *testing.T) {
	e := newEnv(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{{
			Config:      "resource \"wayfinder_variable\" \"example\" {\n  key = \"region\"\n  environment = \"dev\"\n  value = jsonencode(1)\n}\n",
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`(?s)workspace.*must be specified`),
		}},
	})

	if n := len(e.fake.Requests()); n != 0 {
		t.Errorf("API requests = %d, want 0", n)
	}
}

func TestVariable_rejects_invalid_key(t *testing.T) {
	e := newEnv(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{{
			Config:      "resource \"wayfinder_variable\" \"example\" {\n  key = \"1bad.key\"\n  value = jsonencode(1)\n}\n",
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`Invalid Attribute Value Match`),
		}},
	})
}

func TestVariableDataSource_reads_value_at_environment_scope(t *testing.T) {
	e := newEnv(t)
	scope := testScopes["environment"]
	e.fake.SetVariable(scope, "region", []byte(`{"name":"eu-west-2"}`))
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{
			{
				Config: "data \"wayfinder_variable\" \"r\" {\n  key = \"region\"\n" + scopeHCL(scope) + "}\n" +
					"output \"name\" { value = jsondecode(data.wayfinder_variable.r.value).name }\n",
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.wayfinder_variable.r", "id", "ws1/dev/region"),
					resource.TestCheckOutput("name", "eu-west-2"),
				),
			},
			{
				Config:      "data \"wayfinder_variable\" \"r\" {\n  key = \"missing\"\n}\n",
				ExpectError: regexp.MustCompile(`Variable missing not found`),
			},
		},
	})
}

func TestVariable_import_when_server_pretty_prints_value(t *testing.T) {
	// Given a variable the server returns indented, as Wayfinder does.
	e := newEnv(t)
	scope := testScopes["workspace"]
	e.fake.SetVariable(scope, "region", []byte("{\n  \"region\": \"eu-west-2\",\n  \"zones\": [\n    \"a\"\n  ]\n}"))
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{{
			// When importing it.
			Config:        variableHCL(scope, `jsonencode({ region = "eu-west-2", zones = ["a"] })`),
			ResourceName:  "wayfinder_variable.example",
			ImportState:   true,
			ImportStateId: scopeID(scope, "region"),
			// Then state holds the compact form jsonencode produces, so import matches create.
			ImportStateCheck: func(states []*terraform.InstanceState) error {
				if got := states[0].Attributes["value"]; got != `{"region":"eu-west-2","zones":["a"]}` {
					return fmt.Errorf("value = %q", got)
				}

				return nil
			},
		}},
	})
}
