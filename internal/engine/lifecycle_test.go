// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestLifecycle_whenGoldenKindsAtEveryScope(t *testing.T) {
	for _, tc := range []struct{ kind, scope, version, spec string }{
		{"Workspace", "", "", `display_name = "First"
 cloud_provider = "aws"`},
		{"Integration", `workspace = "ws"`, "", `endpoint = "https://example.com"
 api_key_wo = "secret-literal-one"
 api_key_wo_version = "1"`},
		{"Integration", `workspace = "ws"
 environment = "dev"`, "", `endpoint = "https://example.com"
 api_key_wo = "secret-literal-one"
 api_key_wo_version = "1"`},
		{"Blueprint", `workspace = "ws"`, `version = "1.2.3"`, `source = {repository = "https://example.com/repo"}`},
	} {
		t.Run(tc.kind+tc.scope, func(t *testing.T) {
			// Given a golden-kind resource served by the real HTTP fake.
			e := newTestEnv(t, tc.kind)
			typ := "wayfinder_" + e.k.TerraformName
			cfg := fmt.Sprintf("resource %q \"example\" {\n name = \"example\"\n %s\n %s\n %s\n}\n", typ, tc.scope, tc.version, tc.spec)
			// When Terraform applies, updates, imports and destroys it.
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
				{Config: cfg, ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}, Check: checkSecretAbsent(t)},
				{Config: strings.Replace(cfg, tc.spec, tc.spec+"\n labels = {managed = \"yes\"}", 1), ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
				{ResourceName: typ + ".example", ImportState: true},
			}})
			// Then there is one POST, one PUT, one DELETE; secrets never reach state.
			for method, want := range map[string]int{http.MethodPost: 1, http.MethodPut: 1, http.MethodDelete: 1} {
				if got := e.fake.CountRequests(method, ""); got != want {
					t.Fatalf("%s count = %d, want %d", method, got, want)
				}

				t.Logf("%s request count = %d", method, want)
			}
		})
	}
}

func checkSecretAbsent(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		b, err := json.Marshal(s)
		if err != nil {
			return fmt.Errorf("marshal state: %w", err)
		}

		if strings.Contains(string(b), "secret-literal-") {
			return fmt.Errorf("secret literal in state")
		}

		t.Log("secret-not-in-state: PASS (marshalled Terraform state)")

		return nil
	}
}
