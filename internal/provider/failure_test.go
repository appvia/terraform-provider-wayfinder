// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

const workspaceConfig = `
resource "wayfinder_workspace" "example" {
 name = "example"
 description = %q
}
`

func TestProvider_configure_whenTenantMissing(t *testing.T) {
	// Given a provider block with a token but no tenant, and no WAYFINDER_TENANT.
	e := newTestEnv(t)
	cfg := `provider "wayfinder" {
 server = "` + e.fake.URL + `"
 token = "` + fakeapi.DefaultToken + `"
}
` + workspaceConfig
	// When Terraform plans, then configuration fails with the documented message.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{{
		Config:      fmt.Sprintf(cfg, "First"),
		ExpectError: regexp.MustCompile(regexp.QuoteMeta("Set tenant in the provider block or WAYFINDER_TENANT.")),
	}}})

	if n := len(e.fake.Requests()); n != 0 {
		t.Fatalf("requests = %d, want 0", n)
	}

	t.Log("missing tenant: error \"Set tenant in the provider block or WAYFINDER_TENANT.\" PASS")
}

func TestProvider_configure_whenClientCredentialsFromEnvironment(t *testing.T) {
	// Given only WAYFINDER_* environment variables: server and client credentials whose
	// client ID prefix names the tenant.
	e := newTestEnv(t)
	e.fake.AddClientCredential(testTenant+":ci", "client-secret-value")
	t.Setenv("WAYFINDER_SERVER", e.fake.URL)
	t.Setenv("WAYFINDER_CLIENT_ID", testTenant+":ci")
	t.Setenv("WAYFINDER_CLIENT_SECRET", "client-secret-value")
	// When Terraform applies a workspace with an empty provider block.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{{
		Config: `provider "wayfinder" {}` + fmt.Sprintf(workspaceConfig, "First"),
		Check:  resource.TestCheckResourceAttr("wayfinder_workspace.example", "id", "example"),
	}}})
	// Then a token was minted and the object was created in the derived tenant.
	if n := e.fake.Mints(fakeapi.GrantClientCredentials); n < 1 {
		t.Fatalf("client-credential mints = %d", n)
	}

	if n := e.fake.CountRequests(http.MethodPost, "/api/v3/resources/tenants/"+testTenant+"/"); n != 1 {
		t.Fatalf("POST to tenant %s = %d, want 1", testTenant, n)
	}
}

func TestProvider_update_whenChangedOutsideTerraform(t *testing.T) {
	// Given a workspace applied by Terraform and then edited directly in Wayfinder.
	e := newTestEnv(t)
	ref := e.ref(e.kind(t, "workspace"), "", "example")
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: e.providerBlock() + fmt.Sprintf(workspaceConfig, "First")},
		{
			PreConfig: func() {
				e.fake.Mutate(ref, func(obj map[string]any) {
					obj["spec"].(map[string]any)["description"] = "Edited in Wayfinder"
				})
				e.fake.ResetRequests()
			},
			// When Terraform changes a managed attribute.
			Config: e.providerBlock() + fmt.Sprintf(workspaceConfig, "Second"),
			// Then the apply fails with the outside-change error.
			ExpectError: regexp.MustCompile(`was changed outside Terraform`),
		},
	}})
	// And no PUT reached the API after the outside edit (step 2 and destroy).
	for _, r := range e.fake.Requests() {
		t.Logf("request log after outside edit: %s %s -> %d", r.Method, r.Path, r.Status)
	}

	if n := e.fake.CountRequests(http.MethodPut, ""); n != 0 {
		t.Fatalf("PUT after outside edit = %d, want 0", n)
	}

	t.Log("external change: \"was changed outside Terraform\" with PUT=0 PASS")
}
