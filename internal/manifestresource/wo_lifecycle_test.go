package manifestresource

import (
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// configWO renders wayfinder_manifest.this with manifest, an optional manifest_wo heredoc and
// an optional manifest_wo_version map.
func configWO(manifest, wo string, versions map[string]string) string {
	var b strings.Builder
	b.WriteString("resource \"wayfinder_manifest\" \"this\" {\n  manifest = <<-EOT\n" + indent(manifest) + "  EOT\n")
	if wo != "" {
		b.WriteString("  manifest_wo = <<-EOT\n" + indent(wo) + "  EOT\n")
	}
	if versions != nil {
		b.WriteString("  manifest_wo_version = {\n")
		for _, k := range slices.Sorted(maps.Keys(versions)) {
			b.WriteString("    " + strconv.Quote(k) + " = " + strconv.Quote(versions[k]) + "\n")
		}
		b.WriteString("  }\n")
	}
	b.WriteString("}\n")
	return b.String()
}

// wrapTolerant matches phrase literally, allowing Terraform to wrap it at any space.
func wrapTolerant(phrase string) *regexp.Regexp {
	return regexp.MustCompile(strings.ReplaceAll(regexp.QuoteMeta(phrase), " ", `\s+`))
}

// sourceLines matches the configuration lines Terraform quotes under a diagnostic; they echo the
// test's own literal configuration, not text produced by the provider.
var sourceLines = regexp.MustCompile(`(?m)^\s*\d+: .*$`)

const (
	woIntegration = "apiVersion: core/v3\nkind: Integration\nmetadata: {name: i, workspace: w}\nspec:\n  endpoint: https://e\n"
	woPlanLiteral = "wo-lit-k1"
)

func TestPlan_whenManifestWORejected(t *testing.T) {
	pipeline := "apiVersion: core.appvia.io/v3\nkind: Pipeline\nmetadata: {name: build, workspace: ws, environment: dev}\nspec:\n  trigger: manual\n  steps:\n    - {name: test}\n    - {name: lint}\n"
	apiKey := "spec:\n  apiKey: " + woPlanLiteral + "\n"
	for _, tc := range []struct {
		name, manifest, wo string
		versions           map[string]string
		want, literal      string
	}{
		{"metadata in wo", woIntegration, "metadata: {name: " + woPlanLiteral + "}\n", nil, "manifest_wo must not set metadata; set it in manifest.", woPlanLiteral},
		{"status in wo", woIntegration, "status: {message: " + woPlanLiteral + "}\n", nil, "manifest_wo must not set status; set it in manifest.", woPlanLiteral},
		{"versions without wo", woIntegration, "", map[string]string{"spec.apiKey": "1"}, "manifest_wo_version is set but manifest_wo is not.", ""},
		{"unknown key", woIntegration, apiKey, map[string]string{"spec.nope": "1"}, "spec.nope does not match a value in manifest_wo.", woPlanLiteral},
		{"invalid key", woIntegration, apiKey, map[string]string{"spec[": "1"}, "Invalid manifest_wo_version key", woPlanLiteral},
		{"list length mismatch", pipeline, "spec:\n  steps:\n    - {command: " + woPlanLiteral + "}\n", nil, "spec.steps has 1 items in manifest_wo but 2 in manifest", woPlanLiteral},
		{"set in both", woIntegration, "spec:\n  endpoint: https://" + woPlanLiteral + "\n", nil, "spec.endpoint is set in both manifest and manifest_wo.", woPlanLiteral},
		{"malformed yaml", woIntegration, "spec: {token: s3cr3t-literal\n", nil, "manifest_wo is not valid YAML or JSON", "s3cr3t-literal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given a manifest_wo configuration the provider must refuse.
			e := newTestEnv(t)
			var planErr error
			// When planning it.
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: e.factories,
				// ExpectError skips ErrorCheck, so the error is captured here to also assert absence.
				ErrorCheck: func(err error) error { planErr = err; return nil },
				Steps:      []resource.TestStep{{Config: configWO(tc.manifest, tc.wo, tc.versions), PlanOnly: true}},
			})
			// Then planning fails with the problem, without the manifest_wo value, before any API call.
			if planErr == nil {
				t.Fatal("expected a plan error, got none")
			}
			t.Logf("plan error: %s", planErr)
			if !wrapTolerant(tc.want).MatchString(planErr.Error()) {
				t.Fatalf("error does not match %q: %s", tc.want, planErr)
			}
			if tc.literal != "" && strings.Contains(sourceLines.ReplaceAllString(planErr.Error(), ""), tc.literal) {
				t.Fatalf("diagnostic contains manifest_wo literal %q: %s", tc.literal, planErr)
			}
			if n := len(e.fake.Requests()); n != 0 {
				t.Fatalf("requests = %d", n)
			}
		})
	}
}

func TestPlan_whenManifestWOValid(t *testing.T) {
	// Given an ExternalIdentity with a blank token placeholder and the token in manifest_wo.
	e := newTestEnv(t)
	text := "apiVersion: integrations/v3\nkind: ExternalIdentity\nmetadata: {name: mcp}\nspec:\n  type: BearerToken\n  bearerToken:\n    token: \"\"\n"
	wo := "spec:\n  bearerToken:\n    token: " + woPlanLiteral + "\n"
	// When planning it.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(text, wo, map[string]string{"spec.bearerToken.token": "1"}), PlanOnly: true, ExpectNonEmptyPlan: true},
	}})
	// Then the plan succeeds without any API call.
	if n := len(e.fake.Requests()); n != 0 {
		t.Fatalf("requests = %d", n)
	}
}

func TestPlan_whenOnlyVersionChanges(t *testing.T) {
	// Given an applied Integration whose apiKey comes from manifest_wo at version 1.
	e := newTestEnv(t)
	text := "apiVersion: core/v3\nkind: Integration\nmetadata: {name: i, workspace: w}\nspec:\n  endpoint: https://e\n"
	wo := "spec:\n  apiKey: " + woPlanLiteral + "\n"
	// When only the manifest_wo_version changes to 2 and the manifest stays identical.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(text, wo, map[string]string{"spec.apiKey": "1"})},
		{Config: configWO(text, wo, map[string]string{"spec.apiKey": "2"}), ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction("wayfinder_manifest.this", plancheck.ResourceActionUpdate),
				plancheck.ExpectUnknownValue("wayfinder_manifest.this", tfjsonpath.New("status")),
			},
			PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
		}},
	}})
}
