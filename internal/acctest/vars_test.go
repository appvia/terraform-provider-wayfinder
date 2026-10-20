package acctest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
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

// instanceEnvHCL is a workspace plus an environment, the parents of an instance-scoped key.
// No StackInstance is created: the server creates the instance's store on the first write.
func instanceEnvHCL(ws, env string) string {
	return workspaceHCL(ws, "Holds an instance-scoped key") + fmt.Sprintf(`
resource "wayfinder_environment" "test" {
  workspace = wayfinder_workspace.test.name
  name      = %q
}
`, env)
}

// checkVarsDestroyed confirms through the API that every instance-scoped variable and secret
// in the final state is gone. checkDestroyed only covers typed kinds, and instance stores
// cannot be listed, so each key's item path is read directly and must be 404.
func checkVarsDestroyed(s *terraform.State) error {
	c, err := NewClient()
	if err != nil {
		return err
	}
	kinds := map[string]string{"wayfinder_variable": "variables", "wayfinder_secret": "secrets"}
	checked := 0
	for _, rs := range s.RootModule().Resources {
		kind, ok := kinds[rs.Type]
		if !ok || rs.Primary == nil || rs.Primary.Attributes["instance"] == "" {
			continue
		}
		a := rs.Primary.Attributes
		p := "/api/v3/general/tenants/" + url.PathEscape(c.Tenant) + "/workspaces/" + url.PathEscape(a["workspace"]) +
			"/envs/" + url.PathEscape(a["environment"]) + "/instancevars/" + url.PathEscape(a["instance"]) +
			"/" + kind + "/" + url.PathEscape(a["key"])
		if err := c.DoJSON(context.Background(), http.MethodGet, p, nil, nil); !errors.Is(err, client.ErrNotFound) {
			return fmt.Errorf("%s %s still exists after destroy (err=%v)", rs.Type, rs.Primary.ID, err)
		}
		checked++
	}
	if checked == 0 {
		return fmt.Errorf("checkVarsDestroyed: no instance-scoped variable or secret found in state")
	}
	return nil
}

func checkDestroyedWithVars(s *terraform.State) error {
	if err := checkDestroyed(s); err != nil {
		return err
	}
	return checkVarsDestroyed(s)
}

func TestAccVariable_instance(t *testing.T) {
	ws, env, instance, key := RandomName(), RandomName(), RandomName(), RandomName()
	cfg := func(value string) string {
		return instanceEnvHCL(ws, env) + fmt.Sprintf(`
resource "wayfinder_variable" "instance" {
  workspace   = wayfinder_workspace.test.name
  environment = wayfinder_environment.test.name
  instance    = %q
  key         = %q
  value       = jsonencode(%q)
}
`, instance, key, value)
	}
	run(t, lifecycle{
		create: cfg("first"), update: cfg("second"), updated: "wayfinder_variable.instance",
		imports:      []importStep{{addr: "wayfinder_variable.instance", id: ws + "/" + env + "/" + instance + "/" + key}},
		checks:       []resource.TestCheckFunc{resource.TestCheckResourceAttr("wayfinder_variable.instance", "value", `"first"`)},
		checkDestroy: checkDestroyedWithVars,
	})
}

func TestAccSecret_instance(t *testing.T) {
	ws, env, instance, key := RandomName(), RandomName(), RandomName(), RandomName()
	first, second := "tfacc-first-"+RandomName(), "tfacc-second-"+RandomName()
	cfg := func(value, version string) string {
		return instanceEnvHCL(ws, env) + fmt.Sprintf(`
resource "wayfinder_secret" "instance" {
  workspace        = wayfinder_workspace.test.name
  environment      = wayfinder_environment.test.name
  instance         = %q
  key              = %q
  value_wo         = jsonencode(%q)
  value_wo_version = %q
}
`, instance, key, value, version)
	}
	run(t, lifecycle{
		// Rotation: a new value is only sent because value_wo_version changes.
		create: cfg(first, "1"), update: cfg(second, "2"), updated: "wayfinder_secret.instance",
		imports: []importStep{{
			addr: "wayfinder_secret.instance", id: ws + "/" + env + "/" + instance + "/" + key,
			ignore: []string{"value_wo", "value_wo_version"},
		}},
		secrets:      []string{first, second},
		checkDestroy: checkDestroyedWithVars,
	})
}
