// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

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

func integrationConfig(endpoint, secret, version, headers string) string {
	return fmt.Sprintf(`resource "wayfinder_integration" "example" {
 name = "example"
 workspace = "ws"
 endpoint = %q
 api_key_wo = %q
 api_key_wo_version = %q
 headers_wo = %s
 headers_wo_version = %q
}`, endpoint, secret, version, headers, version)
}

func TestSecrets_whenVersionUnchangedOrRotated(t *testing.T) {
	// Given configured scalar and map secrets.
	e := newTestEnv(t, "Integration")
	check := func(wantKey string, wantHeaders map[string]any, blank bool) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			obj, ok := e.fake.Object(e.ref("ws", "", ""))
			if !ok {
				return fmt.Errorf("object missing")
			}

			spec := obj["spec"].(map[string]any)
			if spec["apiKey"] != wantKey || !reflect.DeepEqual(spec["headers"], wantHeaders) {
				return fmt.Errorf("stored secrets do not match rotation contract")
			}

			for _, req := range e.fake.Requests() {
				if req.Method != http.MethodPut {
					continue
				}

				var body struct {
					Spec struct {
						APIKey  string            `json:"apiKey"`
						Headers map[string]string `json:"headers"`
					} `json:"spec"`
				}
				if err := json.Unmarshal(req.Body, &body); err != nil {
					return err
				}

				if blank {
					if body.Spec.APIKey != "" || len(body.Spec.Headers) != 2 {
						return fmt.Errorf("non-secret update did not retain blank map keys")
					}

					for _, v := range body.Spec.Headers {
						if v != "" {
							return fmt.Errorf("unchanged header sent secret")
						}
					}
				}
			}

			return checkSecretAbsent(t)(s)
		}
	}
	// When a non-secret update keeps versions, then another update bumps them.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: integrationConfig("https://one", "secret-literal-one", "1", `{A="secret-literal-header-a", B="secret-literal-header-b"}`), Check: check("secret-literal-one", map[string]any{"A": "secret-literal-header-a", "B": "secret-literal-header-b"}, false)},
		{PreConfig: e.fake.ResetRequests, Config: integrationConfig("https://two", "secret-literal-ignored", "1", `{A="secret-literal-ignored"}`), Check: resource.ComposeAggregateTestCheckFunc(check("secret-literal-one", map[string]any{"A": "secret-literal-header-a", "B": "secret-literal-header-b"}, true), requestCount(t, e, http.MethodPut, 1)), ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
		{PreConfig: e.fake.ResetRequests, Config: integrationConfig("https://two", "secret-literal-two", "2", `{A="secret-literal-rotated"}`), Check: resource.ComposeAggregateTestCheckFunc(check("secret-literal-two", map[string]any{"A": "secret-literal-rotated"}, false), requestCount(t, e, http.MethodPut, 1)), ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
	}})
	// Then omitted headers are removed only on rotation and no secret enters state.
}

func requestCount(t *testing.T, e testEnv, method string, want int) resource.TestCheckFunc {
	return func(*terraform.State) error {
		got := e.fake.CountRequests(method, "")
		if got != want {
			return fmt.Errorf("%s requests = %d, want %d", method, got, want)
		}

		t.Logf("%s request count = %d", method, got)

		return nil
	}
}

func TestUpdate_whenExternalChangeBeforeApply(t *testing.T) {
	// Given state created by Terraform.
	e := newTestEnv(t, "Workspace")
	cfg := `resource "wayfinder_workspace" "example" {
 name="example"
 display_name="First"
 cloud_provider="aws"
}`
	// When an outside change precedes a non-secret Terraform update.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: cfg},
		{PreConfig: func() { e.fake.ResetRequests(); e.fake.Mutate(e.ref("", "", ""), func(map[string]any) {}) }, Config: strings.Replace(cfg, "First", "Second", 1), ExpectError: modifiedRegex()},
	}})
	// Then the engine sends no PUT after that outside change.
	if n := e.fake.CountRequests(http.MethodPut, ""); n != 0 {
		t.Fatalf("PUT requests = %d", n)
	}

	t.Log("Terraform stale-generation apply: provider PUT requests after external change = 0")
}
