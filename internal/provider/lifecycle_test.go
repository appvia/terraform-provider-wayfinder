package provider

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

const secretLiteral = "secret-literal-provider"

func TestProvider_lifecycle_whenTenantWorkspaceAndVersionedKinds(t *testing.T) {
	for _, tc := range []struct {
		name, typ, identity, create, update, importID string
	}{
		{name: "tenant", typ: "workspace", identity: "", importID: "example",
			create: `display_name = "First"
 cloud_provider = "aws"`,
			update: `display_name = "Second"
 cloud_provider = "aws"`},
		{name: "workspace", typ: "integration", identity: `workspace = "ws"`, importID: "ws/example",
			create: `endpoint = "https://example.com"
 api_key_wo = "` + secretLiteral + `"
 api_key_wo_version = "1"
 headers_wo = {"X-Token" = "` + secretLiteral + `-header"}
 headers_wo_version = "1"`,
			update: `endpoint = "https://example.com"
 timeout_seconds = 30
 api_key_wo = "` + secretLiteral + `"
 api_key_wo_version = "1"
 headers_wo = {"X-Token" = "` + secretLiteral + `-header"}
 headers_wo_version = "1"`},
		{name: "versioned", typ: "blueprint", identity: `workspace = "ws"
	 version = "1.2.3"`, importID: "ws/example@1.2.3",
			create: `source = {repository = "https://example.com/repo"}`,
			update: `source = {repository = "https://example.com/repo", ref = "main"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given the provider configured against the fake API.
			e := newTestEnv(t)
			typ := "wayfinder_" + tc.typ
			addr := typ + ".example"
			cfg := func(spec string) string {
				return e.providerBlock() + fmt.Sprintf("resource %q \"example\" {\n name = \"example\"\n labels = {\"wayfinder.appvia.io/managed-by-fake\" = \"true\"}\n %s\n %s\n}\n", typ, tc.identity, spec)
			}
			empty := resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}
			// When Terraform creates, re-plans, updates, imports and destroys it.
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: e.factories,
				Steps: []resource.TestStep{
					{Config: cfg(tc.create), ConfigPlanChecks: empty, Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(addr, "id", tc.importID),
						resource.TestCheckResourceAttr(addr, "status.status", "Success"),
						checkSecretAbsent(t),
					)},
					{Config: cfg(tc.create), PlanOnly: true},
					{Config: cfg(tc.update), ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate)},
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					}, Check: checkSecretAbsent(t)},
					{Config: cfg(tc.update), ResourceName: addr, ImportState: true, ImportStateId: tc.importID, ImportStateVerify: true,
						ImportStateVerifyIgnore: []string{
							"api_key_wo", "api_key_wo_version", "headers_wo", "headers_wo_version", "wait_for_ready", "timeouts",
							// The fake stamps status.lastReconcile.time on every GET.
							"status_json",
						}},
				},
			})
			// Then exactly one POST, one PUT and one DELETE reached the API.
			for method, want := range map[string]int{http.MethodPost: 1, http.MethodPut: 1, http.MethodDelete: 1} {
				if got := e.fake.CountRequests(method, ""); got != want {
					t.Fatalf("%s count = %d, want %d", method, got, want)
				}
			}
			t.Logf("%s: empty plan after apply, update, import verify and destroy PASS (POST=1 PUT=1 DELETE=1)", addr)
		})
	}
}

func TestProvider_dataSource_whenObjectExists(t *testing.T) {
	// Given a workspace created through the provider.
	e := newTestEnv(t)
	cfg := e.providerBlock() + `
resource "wayfinder_workspace" "example" {
 name = "example"
 display_name = "Platform"
 cloud_provider = "gcp"
}
data "wayfinder_workspace" "read" {
 name = wayfinder_workspace.example.name
}
`
	// When the data source reads it.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{{
		Config: cfg,
		// Then it exposes the stored attributes.
		Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.wayfinder_workspace.read", "id", "example"),
			resource.TestCheckResourceAttr("data.wayfinder_workspace.read", "display_name", "Platform"),
			resource.TestCheckResourceAttr("data.wayfinder_workspace.read", "cloud_provider", "gcp"),
			resource.TestCheckResourceAttr("data.wayfinder_workspace.read", "status.status", "Success"),
		),
	}}})
	t.Log("data.wayfinder_workspace read PASS")
}

func checkSecretAbsent(t *testing.T) resource.TestCheckFunc {
	t.Helper()
	return func(s *terraform.State) error {
		b, err := json.Marshal(s)
		if err != nil {
			return fmt.Errorf("marshal state: %w", err)
		}
		if strings.Contains(string(b), secretLiteral) {
			return fmt.Errorf("secret literal found in state")
		}
		t.Log("secret-not-in-state: PASS (marshalled Terraform state)")
		return nil
	}
}
