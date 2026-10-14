package manifestresource

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// woAbsentPlan fails when a write-only literal appears in the plan JSON
// (`terraform show -json <plan>`), prior state included.
type woAbsentPlan []string

func (lits woAbsentPlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	plan := *req.Plan
	// The configuration section echoes the literal expressions of the test's own .tf files,
	// which hold the literals by construction; everything Terraform computed or stored
	// (planned values, resource changes, prior state, variables) is checked.
	plan.Config = nil
	resp.Error = findWOLiteral(plan, lits, "plan JSON")
}

// woAbsentState fails when a write-only literal appears in the state JSON (`terraform show -json`).
type woAbsentState []string

func (lits woAbsentState) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	resp.Error = findWOLiteral(req.State, lits, "state JSON")
}

// woAbsentFlatmap fails when a write-only literal appears in any attribute value of any
// resource in the testing framework's flatmap state.
func woAbsentFlatmap(lits []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, module := range s.Modules {
			for _, res := range module.Resources {
				if err := woAbsentAttributes(res.Primary.Attributes, lits); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

// woAbsentImported applies the flatmap check to the instance states an import step produced
// (import steps run ImportStateCheck, not Check).
func woAbsentImported(lits []string) resource.ImportStateCheckFunc {
	return func(states []*terraform.InstanceState) error {
		for _, s := range states {
			if err := woAbsentAttributes(s.Attributes, lits); err != nil {
				return err
			}
		}
		return nil
	}
}

func woAbsentAttributes(attrs map[string]string, lits []string) error {
	for _, key := range slices.Sorted(maps.Keys(attrs)) {
		for _, lit := range lits {
			if strings.Contains(attrs[key], lit) {
				return fmt.Errorf("write-only literal %q found in flatmap state", lit)
			}
		}
	}
	return nil
}

func findWOLiteral(doc any, lits []string, where string) error {
	b, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", where, err)
	}
	if i := slices.IndexFunc(lits, func(s string) bool { return strings.Contains(string(b), s) }); i >= 0 {
		return fmt.Errorf("write-only literal %q found in %s", lits[i], where)
	}
	return nil
}

// expectDetected inverts check: it passes only when check reports the leak. (A RefreshState
// step fails the test directly on a Check error, so ExpectError cannot be used there.)
func expectDetected(t *testing.T, leaked *regexp.Regexp, check resource.TestCheckFunc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		err := check(s)
		if err == nil || !leaked.MatchString(err.Error()) {
			return fmt.Errorf("secret-in-state check did not detect the leak: %v", err)
		}
		t.Logf("detected as expected: %v", err)
		return nil
	}
}

// loggedState and loggedPlan pass the wrapped check's error through unchanged and log it, so
// a detection asserted with ExpectError is visible in -v output.
type loggedState struct {
	t     *testing.T
	check statecheck.StateCheck
}

func (l loggedState) CheckState(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	l.check.CheckState(ctx, req, resp)
	if resp.Error != nil {
		l.t.Logf("detected as expected: %v", resp.Error)
	}
}

type loggedPlan struct {
	t     *testing.T
	check plancheck.PlanCheck
}

func (l loggedPlan) CheckPlan(ctx context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	l.check.CheckPlan(ctx, req, resp)
	if resp.Error != nil {
		l.t.Logf("detected as expected: %v", resp.Error)
	}
}

const (
	woLeakFirst   = "spec:\n  apiKey: wo-leak-k1\n  headers: {X-A: wo-leak-a1, X-B: wo-leak-b1}\n"
	woLeakRotated = "spec:\n  apiKey: wo-leak-k2\n  headers: {X-A: wo-leak-a1, X-B: wo-leak-b1}\n"
)

var woLeakLiterals = []string{"wo-leak-k1", "wo-leak-k2", "wo-leak-a1", "wo-leak-b1"}

func woObjectJSONClean(lits []string) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith("wayfinder_manifest.this", "object_json", func(v string) error {
		for _, lit := range lits {
			if strings.Contains(v, lit) {
				return fmt.Errorf("write-only literal %q found in object_json", lit)
			}
		}
		return nil
	})
}

func TestWOLeak_whenAppliedRotatedImported(t *testing.T) {
	// Given an Integration whose secrets come only from manifest_wo, with unique literals.
	e := newTestEnv(t)
	lits := woLeakLiterals
	checks := func() (resource.ConfigPlanChecks, []statecheck.StateCheck, resource.TestCheckFunc) {
		return resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{woAbsentPlan(lits)}},
			[]statecheck.StateCheck{woAbsentState(lits)},
			resource.ComposeAggregateTestCheckFunc(woAbsentFlatmap(lits), woObjectJSONClean(lits))
	}
	createPlan, createState, createCheck := checks()
	rotatePlan, rotateState, rotateCheck := checks()
	// When creating, rotating apiKey, refreshing and importing it.
	// Then no literal appears in plan JSON, state JSON, flatmap state or object_json.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: configWO(woIntegration, woLeakFirst, map[string]string{"spec.apiKey": "1", "spec.headers": "1"}),
			ConfigPlanChecks: createPlan, ConfigStateChecks: createState, Check: createCheck},
		{Config: configWO(woIntegration, woLeakRotated, map[string]string{"spec.apiKey": "2", "spec.headers": "1"}),
			ConfigPlanChecks: rotatePlan, ConfigStateChecks: rotateState, Check: rotateCheck},
		{RefreshState: true, Check: woAbsentFlatmap(lits)},
		{ResourceName: "wayfinder_manifest.this", ImportState: true, ImportStateId: "core/Integration/w/i", ImportStateVerify: true,
			ImportStateVerifyIgnore: []string{"manifest", "manifest_wo_version", "object_json", "wait_for_ready", "timeouts"},
			ImportStateCheck:        woAbsentImported(lits)},
	}})
}

func TestWOLeak_whenServerEchoesLiteral(t *testing.T) {
	// Given an applied Integration whose apiKey literal is unique.
	e := newTestEnv(t)
	ref := e.ref(t, "Integration", "w", "", "i")
	lit := "wo-leak-k1"
	lits := woLeakLiterals
	cfg := configWO(woIntegration, woLeakFirst, map[string]string{"spec.apiKey": "1"})
	leaked := func(where string) *regexp.Regexp {
		return regexp.MustCompile(`write-only literal "` + regexp.QuoteMeta(lit) + `" found in ` + where)
	}
	// When the server echoes the literal in status.message after the first apply.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: cfg, Check: woAbsentFlatmap(lits)},
		{PreConfig: func() {
			e.fake.SetStatusScript(ref, []fakeapi.StatusStep{{Status: "Success", Message: lit}})
			t.Logf("leak injected: fake status.message = %q", lit)
		}, RefreshState: true, Check: expectDetected(t, leaked("flatmap state"), woAbsentFlatmap(lits))},
		// Then the flatmap, state JSON and plan JSON checks each detect it.
		{Config: cfg, ConfigStateChecks: []statecheck.StateCheck{loggedState{t, woAbsentState(lits)}}, ExpectError: leaked("state JSON")},
		{Config: cfg, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{loggedPlan{t, woAbsentPlan(lits)}}}, ExpectError: leaked("plan JSON")},
		// And once the status is restored, the same checks pass again.
		{PreConfig: func() {
			e.fake.SetStatusScript(ref, []fakeapi.StatusStep{{Status: "Success"}})
			t.Log("leak detected by flatmap, state JSON and plan JSON checks; status restored")
		}, RefreshState: true, Check: woAbsentFlatmap(lits)},
		{Config: cfg, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{woAbsentPlan(lits)}},
			ConfigStateChecks: []statecheck.StateCheck{woAbsentState(lits)}, Check: woAbsentFlatmap(lits)},
	}})
}

func TestWO_whenOnlyGenerationInPrivateState(t *testing.T) {
	// Given every non-test Go file of this package.
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var withSetKey []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "SetKey(") {
			withSetKey = append(withSetKey, name)
		}
	}
	// Then private state is written only by common.go (the generation baseline).
	if !slices.Equal(withSetKey, []string{"common.go"}) {
		t.Fatalf("SetKey( found in %v, want only [common.go]", withSetKey)
	}
}
