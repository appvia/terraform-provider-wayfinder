package acctest

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVariable_tenantAndWorkspace(t *testing.T) {
	ws, tenantKey, wsKey := RandomName(), RandomName(), RandomName()
	cfg := func(value string) string {
		return workspaceHCL(ws, "Holds a test variable") + fmt.Sprintf(`
resource "wayfinder_variable" "tenant" {
  key   = %q
  value = jsonencode(%q)
}
resource "wayfinder_variable" "workspace" {
  workspace = wayfinder_workspace.test.name
  key       = %q
  value     = jsonencode({ region = "eu-west-2", zones = ["a", "b"] })
}
`, tenantKey, value, wsKey)
	}
	run(t, lifecycle{
		create: cfg("first"), update: cfg("second"), updated: "wayfinder_variable.tenant",
		imports: []importStep{
			{addr: "wayfinder_variable.tenant", id: tenantKey},
			{addr: "wayfinder_variable.workspace", id: ws + "/" + wsKey},
		},
		checks: []resource.TestCheckFunc{resource.TestCheckResourceAttr("wayfinder_variable.tenant", "value", `"first"`)},
	})
}

func TestAccSecret_tenantAndWorkspace(t *testing.T) {
	ws, tenantKey, wsKey := RandomName(), RandomName(), RandomName()
	first, second, wsSecret := "tfacc-first-"+RandomName(), "tfacc-second-"+RandomName(), "tfacc-ws-"+RandomName()
	cfg := func(value, version string) string {
		return workspaceHCL(ws, "Holds a test secret") + fmt.Sprintf(`
resource "wayfinder_secret" "tenant" {
  key              = %q
  value_wo         = jsonencode(%q)
  value_wo_version = %q
}
resource "wayfinder_secret" "workspace" {
  workspace        = wayfinder_workspace.test.name
  key              = %q
  value_wo         = jsonencode(%q)
  value_wo_version = "1"
}
`, tenantKey, value, version, wsKey, wsSecret)
	}
	ignore := []string{"value_wo", "value_wo_version"}
	run(t, lifecycle{
		// Rotation: a new value is only sent because value_wo_version changes.
		create: cfg(first, "1"), update: cfg(second, "2"), updated: "wayfinder_secret.tenant",
		imports: []importStep{
			{addr: "wayfinder_secret.tenant", id: tenantKey, ignore: ignore},
			{addr: "wayfinder_secret.workspace", id: ws + "/" + wsKey, ignore: ignore},
		},
		secrets: []string{first, second, wsSecret},
	})
}
