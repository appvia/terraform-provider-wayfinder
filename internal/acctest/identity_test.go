// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package acctest

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
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

// kubeResourcePlanConfig follows doc/public/examples/KubeResourcePlan-redis.yaml. An empty
// publishState leaves the attribute out of the configuration.
func kubeResourcePlanConfig(name, description, publishState string) string {
	publish := ""
	if publishState != "" {
		publish = fmt.Sprintf("  publish_state      = %q\n", publishState)
	}

	return fmt.Sprintf(`
resource "wayfinder_kube_resource_plan" "test" {
  name               = %q
  version            = "1.0.0"
  description        = %q
%s  kube_resource_type = "AppComponent"
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
`, name, description, publish)
}

// TestAccKubeResourcePlan_versioned follows doc/public/examples/KubeResourcePlan-redis.yaml.
func TestAccKubeResourcePlan_versioned(t *testing.T) {
	name := RandomName()
	run(t, lifecycle{
		create:  kubeResourcePlanConfig(name, "Redis in-memory cache deployment", ""),
		update:  kubeResourcePlanConfig(name, "Redis cache, updated", ""),
		updated: "wayfinder_kube_resource_plan.test",
		// The server defaults helm.release_name to the plan name; import adopts it.
		imports: []importStep{{addr: "wayfinder_kube_resource_plan.test", id: name + "@1.0.0", ignore: []string{"helm.release_name"}}},
		checks:  []resource.TestCheckFunc{resource.TestCheckResourceAttr("wayfinder_kube_resource_plan.test", "id", name+"@1.0.0")},
	})
}

// TestAccKubeResourcePlan_publishState publishes a plan version, then moves it back to Draft,
// and checks the state, the object's metadata and the catalogue's published-versions list.
func TestAccKubeResourcePlan_publishState(t *testing.T) {
	name := RandomName()

	const addr = "wayfinder_kube_resource_plan.test"

	description := "Redis in-memory cache deployment"
	run(t, lifecycle{
		create:  kubeResourcePlanConfig(name, description, "Published"),
		update:  kubeResourcePlanConfig(name, description, "Draft"),
		updated: addr,
		imports: []importStep{{addr: addr, id: name + "@1.0.0", ignore: []string{"helm.release_name"}}},
		checks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(addr, "publish_state", "Published"),
			checkPlanPublishState(t, name, "1.0.0", "Published", true),
		},
		updateChecks: []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(addr, "publish_state", "Draft"),
			checkPlanPublishState(t, name, "1.0.0", "Draft", false),
		},
	})
}

// checkPlanPublishState confirms through the API that the KubeResourcePlan version has
// metadata.publishState == want and that the onlyPublished versions list includes it exactly
// when listed is true.
func checkPlanPublishState(t *testing.T, name, version, want string, listed bool) resource.TestCheckFunc {
	t.Helper()

	return func(*terraform.State) error {
		m, err := manifest.LoadEmbedded()
		if err != nil {
			return err
		}

		k, ok := m.FindKind("kuberesources", "KubeResourcePlan")
		if !ok {
			return fmt.Errorf("KubeResourcePlan is not in the embedded manifest")
		}

		c, err := NewClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		ref := client.Ref{Group: k.Group, Version: k.Version, APIPath: k.APIPath, Name: name, ObjectVersion: version, Versioned: true}

		obj, err := c.Get(ctx, ref)
		if err != nil {
			return err
		}

		meta, _ := obj["metadata"].(map[string]any)
		if got, _ := meta["publishState"].(string); got != want {
			return fmt.Errorf("metadata.publishState = %q, want %q", got, want)
		}

		col, err := ref.CollectionPath(c.Tenant)
		if err != nil {
			return err
		}

		var list struct {
			Items []struct {
				Metadata struct {
					Version string `json:"version"`
				} `json:"metadata"`
			} `json:"items"`
		}
		if err := c.DoJSON(ctx, http.MethodGet, col+"/"+url.PathEscape(name)+"/versions?onlyPublished=true", nil, &list); err != nil {
			return err
		}

		var versions []string
		for _, it := range list.Items {
			versions = append(versions, it.Metadata.Version)
		}

		t.Logf("%s onlyPublished versions: %v (publishState %s)", name, versions, want)

		if slices.Contains(versions, version) != listed {
			return fmt.Errorf("onlyPublished versions of %s = %v; want %s listed=%t", name, versions, version, listed)
		}

		return nil
	}
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
