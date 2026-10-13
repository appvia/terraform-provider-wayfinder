package acctest

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccExternalIdentity_bearerToken follows doc/public/examples/ExternalIdentity-bearer-token-ci.yaml
// and rotates the token by bumping token_wo_version.
func TestAccExternalIdentity_bearerToken(t *testing.T) {
	name := RandomName()
	first, second := "tfacc-token-"+RandomName(), "tfacc-token-"+RandomName()
	cfg := func(description, token, version string) string {
		return fmt.Sprintf(`
resource "wayfinder_external_identity" "test" {
  name         = %q
  type         = "BearerToken"
  description  = %q
  service      = "mcp.example.com"
  usable_for   = ["mcp"]
  bearer_token = { token_wo = %q, token_wo_version = %q }
}
`, name, description, token, version)
	}
	run(t, lifecycle{
		create:  cfg("Token the CI triage agent presents to our MCP server", first, "1"),
		update:  cfg("Rotated token", second, "2"),
		updated: "wayfinder_external_identity.test",
		imports: []importStep{{addr: "wayfinder_external_identity.test", id: name,
			ignore: []string{"bearer_token.token_wo", "bearer_token.token_wo_version"}}},
		checks:  []resource.TestCheckFunc{resource.TestCheckResourceAttr("wayfinder_external_identity.test", "type", "BearerToken")},
		secrets: []string{first, second},
	})
}

// TestAccKubeResourcePlan_versioned follows doc/public/examples/KubeResourcePlan-redis.yaml.
func TestAccKubeResourcePlan_versioned(t *testing.T) {
	name := RandomName()
	cfg := func(description string) string {
		return fmt.Sprintf(`
resource "wayfinder_kube_resource_plan" "test" {
  name               = %q
  version            = "1.0.0"
  description        = %q
  kube_resource_type = "AppComponent"
  inputs = [
    { name = "maxmemory", type = "string", description = "Maximum memory limit for Redis", default_value_json = jsonencode("128mb") },
    { name = "replicas", type = "number", description = "Number of replicas", default_value_json = jsonencode(1) },
  ]
  helm = {
    chart_name      = "redis"
    repo_url        = "https://charts.bitnami.com/bitnami"
    chart_version   = "18.0.0"
    values_template = <<-EOT
      architecture: standalone
      auth:
        enabled: false
      master:
        resources:
          limits:
            memory: $${{ .Inputs.maxmemory }}
        persistence:
          enabled: false
    EOT
  }
}
`, name, description)
	}
	run(t, lifecycle{
		create: cfg("Redis in-memory cache deployment"), update: cfg("Redis cache, updated"),
		updated: "wayfinder_kube_resource_plan.test",
		// The server defaults helm.release_name to the plan name; import adopts it.
		imports: []importStep{{addr: "wayfinder_kube_resource_plan.test", id: name + "@1.0.0", ignore: []string{"helm.release_name"}}},
		checks:  []resource.TestCheckFunc{resource.TestCheckResourceAttr("wayfinder_kube_resource_plan.test", "id", name+"@1.0.0")},
	})
}

func TestAccManifest_group(t *testing.T) {
	name := RandomName()
	cfg := func(description string) string {
		return fmt.Sprintf(`
resource "wayfinder_manifest" "test" {
  manifest = <<-EOT
    apiVersion: org/v3
    kind: Group
    metadata:
      name: %s
    spec:
      description: %s
  EOT
}
`, name, description)
	}
	run(t, lifecycle{
		create: cfg("Created from a manifest"), update: cfg("Updated from a manifest"), updated: "wayfinder_manifest.test",
		imports: []importStep{{addr: "wayfinder_manifest.test", id: "org/Group/" + name, ignore: []string{"manifest", "object_json"}}},
		checks:  []resource.TestCheckFunc{resource.TestCheckResourceAttr("wayfinder_manifest.test", "id", "org/Group/"+name)},
	})
}
