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
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

func instanceScope(name string) fakeapi.VarScope {
	return fakeapi.VarScope{Tenant: tenant, Workspace: "ws1", Environment: "dev", Instance: name}
}

func TestVariable_instance_change_replaces(t *testing.T) {
	e := newEnv(t)
	app1, app2 := instanceScope("app1"), instanceScope("app2")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{
			{
				Config: variableHCL(app1, `jsonencode("eu-west-2")`),
				Check:  expectVariable(e, app1, `"eu-west-2"`),
			},
			{
				Config: variableHCL(app2, `jsonencode("eu-west-2")`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("wayfinder_variable.example", plancheck.ResourceActionDestroyBeforeCreate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("wayfinder_variable.example", "id", "ws1/dev/app2/region"),
					expectVariable(e, app2, `"eu-west-2"`),
					func(*terraform.State) error {
						if _, ok := e.fake.Variable(app1, "region"); ok {
							return fmt.Errorf("variable still present at the previous instance")
						}
						return nil
					},
				),
			},
		},
	})
}

func TestSecret_instance_change_replaces(t *testing.T) {
	e := newEnv(t)
	app1, app2 := instanceScope("app1"), instanceScope("app2")
	stateChecks, planChecks := secretAbsent()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		TerraformVersionChecks:   writeOnlyVersions,
		Steps: []resource.TestStep{
			{
				Config:            secretHCL(app1, secretV1, "1"),
				ConfigStateChecks: stateChecks,
				Check:             expectSecret(e, app1, secretV1),
			},
			{
				Config:            secretHCL(app2, secretV1, "1"),
				ConfigStateChecks: stateChecks,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: append([]plancheck.PlanCheck{
						plancheck.ExpectResourceAction("wayfinder_secret.example", plancheck.ResourceActionDestroyBeforeCreate),
					}, planChecks...),
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("wayfinder_secret.example", "id", "ws1/dev/app2/db_password"),
					resource.TestCheckNoResourceAttr("wayfinder_secret.example", "value_wo"),
					expectSecret(e, app2, secretV1),
					func(*terraform.State) error {
						if _, ok := e.fake.Secret(app1, "db_password"); ok {
							return fmt.Errorf("secret still present at the previous instance")
						}
						return nil
					},
				),
			},
		},
	})
}

func TestScopeValidation_rejects_invalid_instance_config(t *testing.T) {
	scopes := []struct {
		name  string
		scope string
		err   string
	}{
		{"instance_without_environment", "  workspace = \"ws1\"\n  instance = \"app1\"\n", `(?s)environment.*must be specified`},
		{"instance_without_workspace", "  environment = \"dev\"\n  instance = \"app1\"\n", `(?s)workspace.*must be specified`},
		{"empty_instance", "  workspace = \"ws1\"\n  environment = \"dev\"\n  instance = \"\"\n", `non-empty name without '/'`},
		{"instance_with_slash", "  workspace = \"ws1\"\n  environment = \"dev\"\n  instance = \"a/b\"\n", `non-empty name without '/'`},
	}
	schemas := []struct {
		name      string
		writeOnly bool
		hcl       func(scope string) string
	}{
		{"variable", false, func(s string) string {
			return "resource \"wayfinder_variable\" \"example\" {\n  key = \"region\"\n" + s + "  value = jsonencode(1)\n}\n"
		}},
		{"secret", true, func(s string) string {
			return "resource \"wayfinder_secret\" \"example\" {\n  key = \"db_password\"\n" + s +
				"  value_wo = jsonencode(\"x\")\n  value_wo_version = \"1\"\n}\n"
		}},
		{"data_source", false, func(s string) string {
			return "data \"wayfinder_variable\" \"r\" {\n  key = \"region\"\n" + s + "}\n"
		}},
	}
	for _, sc := range schemas {
		for _, c := range scopes {
			t.Run(sc.name+"/"+c.name, func(t *testing.T) {
				e := newEnv(t)
				tc := resource.TestCase{
					ProtoV6ProviderFactories: e.factories,
					Steps: []resource.TestStep{{
						Config:      sc.hcl(c.scope),
						PlanOnly:    true,
						ExpectError: regexp.MustCompile(c.err),
					}},
				}
				if sc.writeOnly {
					tc.TerraformVersionChecks = writeOnlyVersions
				}
				resource.UnitTest(t, tc)
				for _, r := range e.fake.Requests() {
					if strings.Contains(r.Path, "vars") {
						t.Errorf("unexpected vars request %s %s", r.Method, r.Path)
					}
				}
			})
		}
	}
}

func TestVariableDataSource_reads_value_at_instance_scope(t *testing.T) {
	e := newEnv(t)
	scope := testScopes["instance"]
	e.fake.SetVariable(scope, "region", []byte(`{"name":"eu-west-2"}`))
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{{
			Config: "data \"wayfinder_variable\" \"r\" {\n  key = \"region\"\n" + scopeHCL(scope) + "}\n" +
				"output \"name\" { value = jsondecode(data.wayfinder_variable.r.value).name }\n",
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.wayfinder_variable.r", "id", "ws1/dev/app1/region"),
				resource.TestCheckResourceAttr("data.wayfinder_variable.r", "instance", "app1"),
				resource.TestCheckOutput("name", "eu-west-2"),
			),
		}},
	})
}

// TestVariable_instance_scope_isolation manages one instance key and proves the environment
// key and a sibling instance key with the same name are never touched, and that reads at
// instance scope do not fall back to the environment value.
func TestVariable_instance_scope_isolation(t *testing.T) {
	e := newEnv(t)
	envScope := testScopes["environment"]
	app1, app2 := instanceScope("app1"), instanceScope("app2")
	e.fake.SetVariable(envScope, "region", []byte(`"env-value"`))
	e.fake.SetVariable(app1, "region", []byte(`"app1-value"`))
	e.fake.SetVariable(app2, "region", []byte(`"app2-value"`))
	e.fake.SetVariable(envScope, "only_env", []byte(`"env-only"`))

	untouched := func(*terraform.State) error {
		if got, _ := e.fake.Variable(envScope, "region"); !jsonEqual(got, `"env-value"`) {
			return fmt.Errorf("environment value changed to %s", got)
		}
		if got, _ := e.fake.Variable(app2, "region"); !jsonEqual(got, `"app2-value"`) {
			return fmt.Errorf("sibling instance value changed to %s", got)
		}
		return nil
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		CheckDestroy: func(s *terraform.State) error {
			if _, ok := e.fake.Variable(app1, "region"); ok {
				return fmt.Errorf("instance variable still present after destroy")
			}
			return untouched(s)
		},
		Steps: []resource.TestStep{
			{
				Config:             variableHCL(app1, `jsonencode("app1-value")`),
				ResourceName:       "wayfinder_variable.example",
				ImportState:        true,
				ImportStateId:      "ws1/dev/app1/region",
				ImportStatePersist: true,
			},
			{
				Config: variableHCL(app1, `jsonencode("app1-updated")`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("wayfinder_variable.example", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(expectVariable(e, app1, `"app1-updated"`), untouched),
			},
			{
				Config:      "data \"wayfinder_variable\" \"r\" {\n  key = \"only_env\"\n" + scopeHCL(app1) + "}\n",
				ExpectError: regexp.MustCompile(`Variable ws1/dev/app1/only_env not found`),
			},
		},
	})

	var sawInstance bool
	for _, r := range e.fake.Requests() {
		t.Logf("request log: %s %s -> %d", r.Method, r.Path, r.Status)
		if strings.Contains(r.Path, "/instancevars/app1/variables/") {
			sawInstance = true
		}
		if strings.Contains(r.Path, "/instancevars/app2/") || strings.Contains(r.Path, "/envs/dev/vars/") {
			t.Errorf("request reached another scope: %s %s", r.Method, r.Path)
		}
	}
	if !sawInstance {
		t.Error("no request reached /instancevars/app1/variables/")
	}
	if n := e.fake.CountRequests(http.MethodPut, ""); n != 1 {
		t.Errorf("PUT requests = %d, want 1", n)
	}
}
