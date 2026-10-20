// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

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

// groupHCL renders a tenant group shaped like doc/public/examples/Group-developers.yaml.
// Wayfinder refuses subjects that do not exist, so members are other test groups (the cheapest
// subject a test can create) given as HCL name expressions, and managers is raw HCL or "".
func groupHCL(label, name, description, managers string, members ...string) string {
	list := ""
	if len(members) > 0 {
		list = "  members = [\n"
		for _, m := range members {
			list += fmt.Sprintf("    { kind = \"Group\", name = %s },\n", m)
		}
		list += "  ]\n"
	}
	if managers != "" {
		managers = "  managers = " + managers + "\n"
	}
	return fmt.Sprintf(`
resource "wayfinder_group" %q {
  name        = %q
  description = %q
%s%s}
`, label, name, description, managers, list)
}

func TestAccGroup_lifecycle(t *testing.T) {
	name, a, b, c := RandomName(), RandomName(), RandomName(), RandomName()
	cfg := func(description string, members ...string) string {
		return groupHCL("a", a, "Member group", "") + groupHCL("b", b, "Member group", "") +
			groupHCL("c", c, "Member group", "") + groupHCL("test", name, description, "", members...)
	}
	run(t, lifecycle{
		create:  cfg("Development team members", "wayfinder_group.a.name", "wayfinder_group.b.name"),
		update:  cfg("Development team members and c", "wayfinder_group.a.name", "wayfinder_group.b.name", "wayfinder_group.c.name"),
		updated: "wayfinder_group.test",
		imports: []importStep{{addr: "wayfinder_group.test", id: name}},
		checks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr("wayfinder_group.test", "members.#", "2"),
			resource.TestCheckResourceAttr("wayfinder_group.test", "members.0.kind", "Group"),
			resource.TestCheckResourceAttrPair("wayfinder_group.test", "members.1.name", "wayfinder_group.b", "name"),
		},
	})
}

func TestAccRoleBinding_lifecycle(t *testing.T) {
	ws, group, rb := RandomName(), RandomName(), RandomName()
	cfg := func(roles string) string {
		return workspaceHCL(ws, "Holds a test role binding") + groupHCL("test", group, "Bound group", "") + fmt.Sprintf(`
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
	// The server only accepts User managers and reports spec.managers[0]; mapped to that
	// attribute, Terraform quotes the manager's line rather than the resource block.
	resource.Test(t, resource.TestCase{
		PreCheck: func() { PreCheck(t) }, ProtoV6ProviderFactories: ProtoV6ProviderFactories, CheckDestroy: checkDestroyed,
		Steps: []resource.TestStep{{
			Config: groupHCL("test", RandomName(), "Invalid manager", `[
    { kind = "Group", name = "charlie@example.com" },
  ]`),
			ExpectError: regexp.MustCompile(`(?s)Wayfinder validation failed.*with wayfinder_group\.test,.*\d+:\s+\{ kind = "Group", name = "charlie@example\.com" \},.*Only 'User' subjects`),
		}},
	})
}
