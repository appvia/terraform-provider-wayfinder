// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestProvider_applies_workspace_when_actions_audience_is_discovered(t *testing.T) {
	// Given the documented GitHub Actions environment, with no explicit audience.
	e := newTestEnv(t)
	e.fake.AddFederatedIdentity("acme:ci:robot", "subject")

	oidc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("audience") != "https://portal.example/t/acme" || req.Header.Get("Authorization") != "Bearer oidc-request-token" {
			t.Error("unexpected GitHub Actions identity request")
		}

		if _, err := w.Write([]byte(`{"value":"subject"}`)); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(oidc.Close)
	t.Setenv("WAYFINDER_SERVER", e.fake.URL)
	t.Setenv("WAYFINDER_SERVICE_ACCOUNT", "acme:ci:robot")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", oidc.URL)
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "oidc-request-token")

	// When Terraform applies a workspace through the real provider.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{{
			Config: `provider "wayfinder" {}
resource "wayfinder_workspace" "example" {
  name = "example"
  description = "GitHub Actions"
}`,
			// Then the workspace is persisted and its post-apply plan is empty.
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("wayfinder_workspace.example", "id", "example"),
				resource.TestCheckResourceAttr("wayfinder_workspace.example", "description", "GitHub Actions"),
			),
			ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
		}},
	})

	if e.fake.Mints(fakeapi.GrantTokenExchange) == 0 || e.fake.CountRequests("", "/api/v3/general/serverinfo") != 0 {
		t.Fatal("expected token exchange without serverinfo discovery")
	}
}
