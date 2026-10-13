package acctest

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func workspaceHCL(name, description string) string {
	return fmt.Sprintf(`
resource "wayfinder_workspace" "test" {
  name        = %q
  description = %q
}
`, name, description)
}

func TestAccWorkspace_lifecycle(t *testing.T) {
	name := RandomName()
	run(t, lifecycle{
		create: workspaceHCL(name, "Created by acceptance test"), update: workspaceHCL(name, "Updated by acceptance test"),
		updated: "wayfinder_workspace.test",
		imports: []importStep{{addr: "wayfinder_workspace.test", id: name}},
		checks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr("wayfinder_workspace.test", "id", name),
			resource.TestCheckResourceAttrSet("wayfinder_workspace.test", "slug"),
		},
	})
}

func TestAccWorkspaceDataSource_read(t *testing.T) {
	name := RandomName()
	cfg := workspaceHCL(name, "Read by data source") + `
data "wayfinder_workspace" "test" {
  name = wayfinder_workspace.test.name
}
`
	resource.Test(t, resource.TestCase{
		PreCheck: func() { PreCheck(t) }, ProtoV6ProviderFactories: ProtoV6ProviderFactories, CheckDestroy: checkDestroyed,
		Steps: []resource.TestStep{{Config: cfg, Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.wayfinder_workspace.test", "id", name),
			resource.TestCheckResourceAttr("data.wayfinder_workspace.test", "description", "Read by data source"),
			resource.TestCheckResourceAttrPair("data.wayfinder_workspace.test", "slug", "wayfinder_workspace.test", "slug"),
		)}},
	})
}

func TestAccEnvironment_lifecycle(t *testing.T) {
	ws, env := RandomName(), RandomName()
	cfg := func(description string) string {
		return workspaceHCL(ws, "Holds a test environment") + fmt.Sprintf(`
resource "wayfinder_environment" "test" {
  workspace   = wayfinder_workspace.test.name
  name        = %q
  description = %q
}
`, env, description)
	}
	run(t, lifecycle{
		create: cfg("Created"), update: cfg("Updated"), updated: "wayfinder_environment.test",
		imports: []importStep{{addr: "wayfinder_environment.test", id: ws + "/" + env}},
		checks:  []resource.TestCheckFunc{resource.TestCheckResourceAttr("wayfinder_environment.test", "id", ws+"/"+env)},
	})
}

// groupHCL follows doc/public/examples/Group-developers.yaml at tenant scope.
func groupHCL(name, description, managerKind string, members ...string) string {
	list := ""
	for _, m := range members {
		list += fmt.Sprintf("    { kind = \"User\", name = %q },\n", m)
	}
	return fmt.Sprintf(`
resource "wayfinder_group" "test" {
  name        = %q
  description = %q
  managers = [
    { kind = %q, name = "charlie@example.com" },
  ]
  members = [
%s  ]
}
`, name, description, managerKind, list)
}

func TestAccGroup_lifecycle(t *testing.T) {
	name := RandomName()
	run(t, lifecycle{
		create:  groupHCL(name, "Development team members", "User", "alice@example.com", "bob@example.com"),
		update:  groupHCL(name, "Development team members and dana", "User", "alice@example.com", "bob@example.com", "dana@example.com"),
		updated: "wayfinder_group.test",
		imports: []importStep{{addr: "wayfinder_group.test", id: name}},
		checks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr("wayfinder_group.test", "members.#", "2"),
			resource.TestCheckResourceAttr("wayfinder_group.test", "managers.0.name", "charlie@example.com"),
		},
	})
}

func TestAccRoleBinding_lifecycle(t *testing.T) {
	ws, group, rb := RandomName(), RandomName(), RandomName()
	cfg := func(roles string) string {
		return workspaceHCL(ws, "Holds a test role binding") + groupHCL(group, "Bound group", "User", "alice@example.com") + fmt.Sprintf(`
resource "wayfinder_role_binding" "test" {
  workspace = wayfinder_workspace.test.name
  name      = %q
  roles     = %s
  subject   = { kind = "Group", name = wayfinder_group.test.name }
}
`, rb, roles)
	}
	run(t, lifecycle{
		create: cfg(`["workspace.member"]`), update: cfg(`["workspace.member", "workspace.manager"]`),
		updated: "wayfinder_role_binding.test",
		imports: []importStep{{addr: "wayfinder_role_binding.test", id: ws + "/" + rb}},
		checks:  []resource.TestCheckFunc{resource.TestCheckResourceAttr("wayfinder_role_binding.test", "roles.#", "1")},
	})
}

func TestAccServiceAccount_lifecycle(t *testing.T) {
	ws, tenantSA, wsSA := RandomName(), RandomName(), RandomName()
	cfg := func(description string) string {
		return workspaceHCL(ws, "Holds a test service account") + fmt.Sprintf(`
resource "wayfinder_service_account" "tenant" {
  name        = %q
  description = %q
}
resource "wayfinder_service_account" "workspace" {
  workspace   = wayfinder_workspace.test.name
  name        = %q
  description = "Workspace service account"
}
`, tenantSA, description, wsSA)
	}
	run(t, lifecycle{
		create: cfg("CI deployer"), update: cfg("CI deployer, updated"), updated: "wayfinder_service_account.tenant",
		imports: []importStep{
			{addr: "wayfinder_service_account.tenant", id: tenantSA},
			{addr: "wayfinder_service_account.workspace", id: ws + "/" + wsSA},
		},
	})
}

func TestAccWorkspace_invalidEnumFailsAtPlan(t *testing.T) {
	cfg := fmt.Sprintf(`
resource "wayfinder_workspace" "test" {
  name = %q
  type = "not-a-workspace-type"
}
`, RandomName())
	resource.Test(t, resource.TestCase{
		PreCheck: func() { PreCheck(t) }, ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: cfg, PlanOnly: true,
			ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Value Match.*with wayfinder_workspace\.test.*value must be one of`),
		}},
	})
}

func TestAccGroup_serverValidationMapsToAttribute(t *testing.T) {
	// The server only accepts User managers and reports spec.managers[0].kind.
	resource.Test(t, resource.TestCase{
		PreCheck: func() { PreCheck(t) }, ProtoV6ProviderFactories: ProtoV6ProviderFactories, CheckDestroy: checkDestroyed,
		Steps: []resource.TestStep{{
			Config:      groupHCL(RandomName(), "Invalid manager", "Group", "alice@example.com"),
			ExpectError: regexp.MustCompile(`(?s)Wayfinder validation failed.*with wayfinder_group\.test,.*managers.*Only 'User' subjects`),
		}},
	})
}
