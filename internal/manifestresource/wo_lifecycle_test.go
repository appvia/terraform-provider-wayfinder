package manifestresource

import (
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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

func TestPlan_whenOnlyVersionChangesAndStatusMoves(t *testing.T) {
	// Given an applied Integration whose apiKey comes from manifest_wo at version 1.
	e := newTestEnv(t)
	ref := e.ref(t, "Integration", "w", "", "i")
	text := "apiVersion: core/v3\nkind: Integration\nmetadata: {name: i, workspace: w}\nspec:\n  endpoint: https://e\n"
	wo := "spec:\n  apiKey: " + woPlanLiteral + "\n"
	// When only the manifest_wo_version changes to 2 and the manifest stays identical.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(text, wo, map[string]string{"spec.apiKey": "1"}),
			Check: resource.TestCheckResourceAttr("wayfinder_manifest.this", "status.message", "")},
		{PreConfig: func() {
			// Refresh and the update's fresh GET keep the old message. The PUT restarts
			// the script, and the waiter reaches a different reconciled status.
			e.fake.SetStatusScript(ref, []fakeapi.StatusStep{
				{Status: "Success"},
				{Status: "Success"},
				{Status: "Success", Message: "apiKey version 2 applied"},
			})
		}, Config: configWO(text, wo, map[string]string{"spec.apiKey": "2"}), ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction("wayfinder_manifest.this", plancheck.ResourceActionUpdate),
				plancheck.ExpectUnknownValue("wayfinder_manifest.this", tfjsonpath.New("status")),
			},
			PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
		}, Check: resource.TestCheckResourceAttr("wayfinder_manifest.this", "status.message", "apiKey version 2 applied")},
	}})
	// Then the version bump really writes and applies the changed status without inconsistency.
	countWrites(t, e.fake, map[string]int{http.MethodPost: 1, http.MethodPut: 1})
}

const (
	woAllFirst  = "spec:\n  apiKey: wo-lit-k1\n  headers: {X-A: wo-lit-a1, X-B: wo-lit-b1}\n"
	woAllSecond = "spec:\n  apiKey: wo-lit-k2\n  headers: {X-A: wo-lit-a2, X-B: wo-lit-b2}\n"
)

// checkStoredWO checks the actual stored secrets before Terraform destroys the test object,
// and checks that none of the test literals reached public state, including object_json.
func checkStoredWO(t *testing.T, e testEnv, ref fakeapi.Ref, want map[string]any) resource.TestCheckFunc {
	t.Helper()
	return func(state *terraform.State) error {
		obj, exists := e.fake.Object(ref)
		if !exists {
			return fmt.Errorf("stored object %s is missing", ref.Name)
		}
		for _, key := range slices.Sorted(maps.Keys(want)) {
			p, err := parseWOPath(key)
			if err != nil {
				return err
			}
			got, ok := lookup(obj, p)
			t.Logf("stored %s = %v", key, got)
			if !ok || !reflect.DeepEqual(got, want[key]) {
				return fmt.Errorf("stored %s = %#v, want %#v", key, got, want[key])
			}
		}
		for _, module := range state.Modules {
			for _, res := range module.Resources {
				for key, value := range res.Primary.Attributes {
					if strings.Contains(value, "wo-lit-") || key == "manifest_wo" && value != "" {
						return fmt.Errorf("write-only value found in state attribute %s", key)
					}
				}
			}
		}
		return nil
	}
}

func integrationWOValues(apiKey, a, b string) map[string]any {
	return map[string]any{
		"spec.apiKey":  apiKey,
		"spec.headers": map[string]any{"X-A": a, "X-B": b},
	}
}

func TestWO_whenCreatedSendsAll(t *testing.T) {
	// Given write-only string and map values, with only the string versioned.
	e := newTestEnv(t)
	ref := e.ref(t, "Integration", "w", "", "i")
	// When creating, then all values are stored and the post-apply plan is empty.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(woIntegration, woAllFirst, map[string]string{"spec.apiKey": "1"}),
			ConfigPlanChecks: emptyAfterApply(),
			Check:            checkStoredWO(t, e, ref, integrationWOValues("wo-lit-k1", "wo-lit-a1", "wo-lit-b1"))},
	}})
	countWrites(t, e.fake, map[string]int{http.MethodPost: 1, http.MethodPut: 0})
}

func TestWO_whenOnlyManifestWOChanges(t *testing.T) {
	// Given applied values at version 1.
	e := newTestEnv(t)
	ref := e.ref(t, "Integration", "w", "", "i")
	versions := map[string]string{"spec.apiKey": "1"}
	// When only the write-only literals change, then there is no plan or write.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(woIntegration, woAllFirst, versions)},
		{Config: configWO(woIntegration, woAllSecond, versions), ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
		}, Check: checkStoredWO(t, e, ref, integrationWOValues("wo-lit-k1", "wo-lit-a1", "wo-lit-b1"))},
	}})
	countWrites(t, e.fake, map[string]int{http.MethodPost: 1, http.MethodPut: 0})
}

func TestWO_whenOneKeyRotated(t *testing.T) {
	// Given a versioned apiKey and one versioned header.
	e := newTestEnv(t)
	ref := e.ref(t, "Integration", "w", "", "i")
	// When every literal changes but only X-A's version changes, then only X-A rotates.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(woIntegration, woAllFirst, map[string]string{"spec.apiKey": "1", `spec.headers["X-A"]`: "1"})},
		{Config: configWO(woIntegration, woAllSecond, map[string]string{"spec.apiKey": "1", `spec.headers["X-A"]`: "2"}),
			ConfigPlanChecks: emptyAfterApply(),
			Check:            checkStoredWO(t, e, ref, integrationWOValues("wo-lit-k1", "wo-lit-a2", "wo-lit-b1"))},
	}})
	countWrites(t, e.fake, map[string]int{http.MethodPost: 1, http.MethodPut: 1})
}

func TestWO_whenWholeMapRotated(t *testing.T) {
	// Given two headers versioned as a whole map.
	e := newTestEnv(t)
	ref := e.ref(t, "Integration", "w", "", "i")
	// When rotating the map to only X-A, then X-B is absent rather than overlaid back.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(woIntegration, woAllFirst, map[string]string{"spec.headers": "1"})},
		{Config: configWO(woIntegration, "spec:\n  apiKey: wo-lit-k1\n  headers: {X-A: wo-lit-a2}\n", map[string]string{"spec.headers": "2"}),
			ConfigPlanChecks: emptyAfterApply(),
			Check: checkStoredWO(t, e, ref, map[string]any{
				"spec.apiKey": "wo-lit-k1", "spec.headers": map[string]any{"X-A": "wo-lit-a2"},
			})},
	}})
	countWrites(t, e.fake, map[string]int{http.MethodPost: 1, http.MethodPut: 1})
}

func TestWO_whenVersionKeyRemoved(t *testing.T) {
	// Given a versioned apiKey.
	e := newTestEnv(t)
	ref := e.ref(t, "Integration", "w", "", "i")
	// When removing its version, then Terraform updates state without a PUT or secret deletion.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(woIntegration, woAllFirst, map[string]string{"spec.apiKey": "1"})},
		{Config: configWO(woIntegration, woAllSecond, map[string]string{}), ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction("wayfinder_manifest.this", plancheck.ResourceActionUpdate),
			},
			PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
		}, Check: checkStoredWO(t, e, ref, integrationWOValues("wo-lit-k1", "wo-lit-a1", "wo-lit-b1"))},
	}})
	countWrites(t, e.fake, map[string]int{http.MethodPost: 1, http.MethodPut: 0})
}

func TestWO_whenVersionBumpedButValueBlank(t *testing.T) {
	// Given an applied apiKey whose read value is blank.
	e := newTestEnv(t)
	ref := e.ref(t, "Integration", "w", "", "i")
	// When its version changes but the configured value is blank, then the stored key is kept.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(woIntegration, woAllFirst, map[string]string{"spec.apiKey": "1"})},
		{Config: configWO(woIntegration, "spec:\n  apiKey: \"\"\n", map[string]string{"spec.apiKey": "2"}),
			ConfigPlanChecks: emptyAfterApply(),
			Check:            checkStoredWO(t, e, ref, integrationWOValues("wo-lit-k1", "wo-lit-a1", "wo-lit-b1"))},
	}})
	countWrites(t, e.fake, map[string]int{http.MethodPost: 1, http.MethodPut: 0})
}

func TestWO_whenNoVersionMap(t *testing.T) {
	// Given values created without any version map.
	e := newTestEnv(t)
	ref := e.ref(t, "Integration", "w", "", "i")
	check := checkStoredWO(t, e, ref, integrationWOValues("wo-lit-k1", "wo-lit-a1", "wo-lit-b1"))
	// When only those literals change, then creation sent them but the update is an empty plan.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(woIntegration, woAllFirst, nil), ConfigPlanChecks: emptyAfterApply(), Check: check},
		{Config: configWO(woIntegration, woAllSecond, nil), ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
		}, Check: check},
	}})
	countWrites(t, e.fake, map[string]int{http.MethodPost: 1, http.MethodPut: 0})
}

func TestWO_whenReplaced(t *testing.T) {
	// Given an applied Integration with a fixed version.
	e := newTestEnv(t)
	versions := map[string]string{"spec.apiKey": "1"}
	// When the name changes and versions do not, then replacement sends every new literal.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(woIntegration, woAllFirst, versions)},
		{Config: configWO(strings.Replace(woIntegration, "name: i", "name: i2", 1), woAllSecond, versions),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("wayfinder_manifest.this", plancheck.ResourceActionDestroyBeforeCreate),
				},
				PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			},
			Check: checkStoredWO(t, e, e.ref(t, "Integration", "w", "", "i2"), integrationWOValues("wo-lit-k2", "wo-lit-a2", "wo-lit-b2"))},
	}})
	countWrites(t, e.fake, map[string]int{http.MethodPost: 2, http.MethodPut: 0})
}

func TestWO_whenImportedThenVersioned(t *testing.T) {
	// Given an existing Integration with an old stored key.
	e := newTestEnv(t)
	ref := e.ref(t, "Integration", "w", "", "i")
	e.fake.Seed(ref, map[string]any{"apiVersion": "core.appvia.io/v3", "kind": "Integration",
		"spec": map[string]any{"endpoint": "https://e", "apiKey": "wo-lit-old"}})
	// When importing with a new versioned key, then it is rotated in one PUT.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(woIntegration, "spec:\n  apiKey: wo-lit-k1\n", map[string]string{"spec.apiKey": "1"}) +
			"import {\n  to = wayfinder_manifest.this\n  id = \"core/Integration/w/i\"\n}\n",
			ConfigPlanChecks: emptyAfterApply(),
			Check:            checkStoredWO(t, e, ref, map[string]any{"spec.apiKey": "wo-lit-k1"})},
	}})
	countWrites(t, e.fake, map[string]int{http.MethodPost: 0, http.MethodPut: 1})
}

func TestWO_whenImportedWithoutVersions(t *testing.T) {
	// Given an existing Integration with an old stored key.
	e := newTestEnv(t)
	ref := e.ref(t, "Integration", "w", "", "i")
	e.fake.Seed(ref, map[string]any{"apiVersion": "core.appvia.io/v3", "kind": "Integration",
		"spec": map[string]any{"endpoint": "https://e", "apiKey": "wo-lit-old"}})
	// When importing without versions, then the configured write-only value is not sent.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(woIntegration, "spec:\n  apiKey: wo-lit-k1\n", nil) +
			"import {\n  to = wayfinder_manifest.this\n  id = \"core/Integration/w/i\"\n}\n",
			ConfigPlanChecks: emptyAfterApply(),
			Check:            checkStoredWO(t, e, ref, map[string]any{"spec.apiKey": "wo-lit-old"})},
	}})
	countWrites(t, e.fake, map[string]int{http.MethodPost: 0, http.MethodPut: 0})
}

func TestWO_whenTypedPlaceholder(t *testing.T) {
	// Given a typed ExternalIdentity with a blank token placeholder.
	e := newTestEnv(t)
	text := "apiVersion: integrations/v3\nkind: ExternalIdentity\nmetadata: {name: mcp}\nspec:\n  type: BearerToken\n  bearerToken:\n    token: \"\"\n"
	// When the token is provided through manifest_wo, then the placeholder stays plan-stable.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(text, "spec:\n  bearerToken:\n    token: wo-lit-token\n", map[string]string{"spec.bearerToken.token": "1"}),
			ConfigPlanChecks: emptyAfterApply(),
			Check: checkStoredWO(t, e, e.ref(t, "ExternalIdentity", "", "", "mcp"),
				map[string]any{"spec.bearerToken.token": "wo-lit-token"})},
	}})
	countWrites(t, e.fake, map[string]int{http.MethodPost: 1, http.MethodPut: 0})
}

func TestWO_whenMovedFromManifestPlaceholder(t *testing.T) {
	// Given a secret map key represented by a blank manifest placeholder.
	e := newTestEnv(t)
	ref := e.ref(t, "Integration", "w", "", "i")
	wo := "spec:\n  headers: {X-A: wo-lit-a1}\n"
	versions := map[string]string{"spec.headers": "1"}
	check := checkStoredWO(t, e, ref, map[string]any{"spec.headers": map[string]any{"X-A": "wo-lit-a1"}})
	// When removing the manifest placeholder without rotating, then prior-key deletion keeps X-A.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(woIntegration+"  headers: {X-A: \"\"}\n", wo, versions), ConfigPlanChecks: emptyAfterApply(), Check: check},
		{Config: configWO(woIntegration, wo, versions), ConfigPlanChecks: emptyAfterApply(), Check: check},
	}})
	countWrites(t, e.fake, map[string]int{http.MethodPost: 1, http.MethodPut: 0})
}

func TestUpdate_whenServerRejectsManifestWOField(t *testing.T) {
	// Given a server that rejects apiKey on update, but accepts creation.
	e := newTestEnv(t, func(s *fakeapi.KindSpec) {
		if s.Kind == "Integration" {
			s.Validate = func(op fakeapi.Operation, _ map[string]any) []fakeapi.FieldError {
				if op == fakeapi.OpUpdate {
					return []fakeapi.FieldError{{Field: "spec.apiKey", Message: "bad key"}}
				}
				return nil
			}
		}
	})
	ref := e.ref(t, "Integration", "w", "", "i")
	// When rotating, then the error names manifest_wo and the server field, never a secret value.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(woIntegration, woAllFirst, map[string]string{"spec.apiKey": "1"})},
		{Config: configWO(woIntegration, woAllSecond, map[string]string{"spec.apiKey": "2"}),
			ExpectError: regexp.MustCompile(`(?s)manifest_wo\s*=.*spec\.apiKey: bad key`)},
		{Config: configWO(woIntegration, woAllFirst, map[string]string{"spec.apiKey": "1"}),
			ConfigPlanChecks: emptyAfterApply(),
			Check:            checkStoredWO(t, e, ref, integrationWOValues("wo-lit-k1", "wo-lit-a1", "wo-lit-b1"))},
	}})
	countWrites(t, e.fake, map[string]int{http.MethodPost: 1, http.MethodPut: 1})
}
