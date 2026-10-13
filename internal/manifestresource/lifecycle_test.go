package manifestresource

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func emptyAfterApply() resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}
}

func countWrites(t *testing.T, f *fakeapi.Server, want map[string]int) {
	t.Helper()
	for method, n := range want {
		got := f.CountRequests(method, "")
		t.Logf("%s request count = %d", method, got)
		if got != n {
			t.Fatalf("%s count = %d, want %d", method, got, n)
		}
	}
}

func TestLifecycle_whenTypedAndManifestOnlyKinds(t *testing.T) {
	for _, tc := range []struct{ name, kind, create, update, importID, status string }{
		{"typed Workspace", "Workspace",
			"apiVersion: core/v3\nkind: Workspace\nmetadata:\n  name: plat\nspec:\n  displayName: Platform\n  cloudProvider: aws\n",
			"apiVersion: core/v3\nkind: Workspace\nmetadata:\n  name: plat\n  labels:\n    team: platform\nspec:\n  displayName: Platform team\n  cloudProvider: aws\n",
			"core.appvia.io/Workspace/plat", "Success"},
		{"manifest-only Pipeline", "Pipeline",
			"apiVersion: core.appvia.io/v3\nkind: Pipeline\nmetadata:\n  name: build\n  workspace: ws\n  environment: dev\nspec:\n  trigger: manual\n  steps:\n    - name: test\n      command: make test\n",
			`{"apiVersion":"core.appvia.io/v3","kind":"Pipeline","metadata":{"name":"build","workspace":"ws","environment":"dev"},"spec":{"trigger":"push","steps":[{"name":"test","command":"make test"},{"name":"lint"}]}}`,
			// waitForReady:false: Terraform does not wait, so the create response status remains.
			"core.appvia.io/Pipeline/ws/dev/build", "Pending"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given a kind served by the HTTP fake (Pipeline is typed:false in the manifest).
			e := newTestEnv(t)
			if k := findKind(t, e.m, tc.kind); tc.kind == "Pipeline" && k.Typed {
				t.Fatal("Pipeline must be manifest-only")
			}
			// When Terraform creates, updates, imports and destroys it.
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
				{Config: config(tc.create), ConfigPlanChecks: emptyAfterApply(), Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("wayfinder_manifest.this", "id", tc.importID),
					resource.TestCheckResourceAttr("wayfinder_manifest.this", "status.status", tc.status),
				)},
				{Config: config(tc.update), ConfigPlanChecks: emptyAfterApply()},
				{ResourceName: "wayfinder_manifest.this", ImportState: true, ImportStateId: tc.importID, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"manifest", "object_json", "wait_for_ready", "timeouts"}},
			}})
			// Then exactly one POST, one PUT and one DELETE reached Wayfinder.
			countWrites(t, e.fake, map[string]int{http.MethodPost: 1, http.MethodPut: 1, http.MethodDelete: 1})
		})
	}
}

func TestLifecycle_whenImportedThenConfigured(t *testing.T) {
	// Given a Pipeline that already exists in Wayfinder.
	e := newTestEnv(t)
	ref := e.ref(t, "Pipeline", "ws", "dev", "build")
	e.fake.Seed(ref, map[string]any{"apiVersion": "core.appvia.io/v3", "kind": "Pipeline", "spec": map[string]any{"trigger": "manual"}})
	cfg := config("apiVersion: core/v3\nkind: Pipeline\nmetadata: {name: build, workspace: ws, environment: dev}\nspec: {trigger: manual}\n")
	// When importing it with an import block and applying the matching manifest.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: cfg + "import {\n  to = wayfinder_manifest.this\n  id = \"core/Pipeline/ws/dev/build\"\n}\n", ConfigPlanChecks: emptyAfterApply()},
	}})
	// Then the import adopted it without any write.
	countWrites(t, e.fake, map[string]int{http.MethodPost: 0, http.MethodPut: 0})
}

func TestGroupYAML_whenServerAddsLabelsAndDefaults(t *testing.T) {
	// Given the Group-developers.yaml example and a server that defaults spec.visibility.
	e := newTestEnv(t, func(s *fakeapi.KindSpec) {
		if s.Kind == "Group" {
			s.Defaults = map[string]any{"spec.visibility": "workspace"}
		}
	})
	group := "apiVersion: org/v3\nkind: Group\nmetadata:\n  name: developers\n  workspace: ws1\n  labels:\n    team: dev\nspec:\n  description: Development team members\n  managers:\n    - kind: User\n      name: charlie@example.com\n  members:\n    - kind: User\n      name: alice@example.com\n    - kind: User\n      name: bob@example.com\n"
	second := strings.Replace(group, "Development team members", "Developers", 1)
	third := strings.Replace(second, "    - kind: User\n      name: bob@example.com\n", "", 1)
	// When applying it and updating it twice.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: config(group), ConfigPlanChecks: emptyAfterApply()},
		{Config: config(second), ConfigPlanChecks: emptyAfterApply()},
		{Config: config(third), ConfigPlanChecks: emptyAfterApply()},
	}})
	// Then each change is one PUT and the server-added label and default never diff.
	countWrites(t, e.fake, map[string]int{http.MethodPost: 1, http.MethodPut: 2})
}

func TestUpdate_whenWhitespaceOnlyChangeSendsNoPUT(t *testing.T) {
	// Given an applied Workspace manifest.
	e := newTestEnv(t)
	text := "apiVersion: core/v3\nkind: Workspace\nmetadata: {name: plat}\nspec: {displayName: P, cloudProvider: aws}\n"
	// When the manifest is reformatted as JSON with the same content.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: config(text)},
		{Config: config(`{"apiVersion":"core/v3","kind":"Workspace","metadata":{"name":"plat"},"spec":{"displayName":"P","cloudProvider":"aws"}}`), ConfigPlanChecks: emptyAfterApply()},
	}})
	// Then nothing is written.
	countWrites(t, e.fake, map[string]int{http.MethodPut: 0})
}

func TestPlan_whenNameChangesReplaces(t *testing.T) {
	// Given an applied Workspace.
	e := newTestEnv(t)
	text := "apiVersion: core/v3\nkind: Workspace\nmetadata: {name: plat}\nspec: {displayName: P, cloudProvider: aws}\n"
	// When metadata.name changes.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: config(text)},
		{Config: config(strings.Replace(text, "name: plat", "name: plat2", 1)), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
			plancheck.ExpectResourceAction("wayfinder_manifest.this", plancheck.ResourceActionDestroyBeforeCreate),
		}}},
	}})
	// Then the old object is deleted and the new one created.
	countWrites(t, e.fake, map[string]int{http.MethodPost: 2, http.MethodDelete: 2, http.MethodPut: 0})
}

func TestPlan_whenManifestRejected(t *testing.T) {
	for _, tc := range []struct{ name, text, want string }{
		{"excluded kind", "apiVersion: core/v3\nkind: Run\nmetadata: {name: r}\n", `Kind Run cannot be managed with Terraform\.`},
		{"unknown kind", "apiVersion: core/v3\nkind: Gadget\nmetadata: {name: g}\n", `Kind Gadget is not supported by this provider version; upgrade the\s+provider\.`},
		{"bearer token", "apiVersion: integrations/v3\nkind: ExternalIdentity\nmetadata: {name: mcp}\nspec:\n  type: BearerToken\n  bearerToken:\n    token: x\n",
			`spec\.bearerToken\.token is a secret\. Set it with the\s+wayfinder_external_identity\.bearer_token\.token_wo attribute so it is never\s+stored in state\.`},
		{"secret map", "apiVersion: core/v3\nkind: Integration\nmetadata: {name: i, workspace: w}\nspec:\n  endpoint: https://e\n  headers: {Authorization: secret-literal}\n",
			`spec\.headers is a secret\. Set it with the wayfinder_integration\.headers_wo\s+attribute`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given a manifest the provider must refuse.
			e := newTestEnv(t)
			// When planning it.
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
				{Config: config(tc.text), PlanOnly: true, ExpectError: regexp.MustCompile(tc.want)},
			}})
			// Then planning fails before any API call.
			if n := len(e.fake.Requests()); n != 0 {
				t.Fatalf("requests = %d", n)
			}
		})
	}
}

func TestUpdate_whenChangedOutsideTerraform(t *testing.T) {
	// Given an applied Pipeline that is then edited outside Terraform.
	e := newTestEnv(t)
	ref := e.ref(t, "Pipeline", "ws", "dev", "build")
	text := "apiVersion: core/v3\nkind: Pipeline\nmetadata: {name: build, workspace: ws, environment: dev}\nspec: {trigger: manual}\n"
	// When a managed field changes and Terraform applies.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: config(text)},
		{
			PreConfig: func() {
				e.fake.Mutate(ref, func(obj map[string]any) { obj["spec"].(map[string]any)["steps"] = []any{map[string]any{"name": "ui"}} })
				e.fake.ResetRequests()
			},
			Config:      config(strings.Replace(text, "manual", "push", 1)),
			ExpectError: regexp.MustCompile(`(?s)changed outside Terraform.*terraform import\s+wayfinder_manifest.<name>\s+core.appvia.io/Pipeline/ws/dev/build`),
		},
	}})
	// Then no PUT was sent.
	if n := e.fake.CountRequests(http.MethodPut, ""); n != 0 {
		t.Fatalf("PUT = %d", n)
	}
}

func TestCreate_whenServerValidationFails(t *testing.T) {
	// Given a server that rejects spec.trigger.
	e := newTestEnv(t, func(s *fakeapi.KindSpec) {
		if s.Kind == "Pipeline" {
			s.Validate = func(fakeapi.Operation, map[string]any) []fakeapi.FieldError {
				return []fakeapi.FieldError{{Field: "spec.trigger", Message: "trigger not allowed"}}
			}
		}
	})
	// When creating. Then the field and message are reported on manifest.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: config("apiVersion: core/v3\nkind: Pipeline\nmetadata: {name: b, workspace: ws, environment: dev}\nspec: {trigger: push}\n"), ExpectError: regexp.MustCompile(`spec\.trigger: trigger not allowed`)},
	}})
}
