// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package generated_test

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi/fakekinds"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/appvia/terraform-provider-wayfinder/internal/provider"
)

// generatedCase is one typed kind's plan-stability fixture, emitted by tfgen into
// zz_resources_test.go.
type generatedCase struct {
	Type, Group, Kind, ImportID, Config string
	// Secrets are the unique write-only literals in Config; WriteOnly their state paths.
	Secrets, WriteOnly []string
	// Skip is set when tfgen could not synthesize a validator-compliant fixture.
	Skip string
}

const testTenant = "acme"

// embedded is the provider's manifest, parsed once for every case (read-only).
var embedded = sync.OnceValues(manifest.LoadEmbedded)

// TestMain clears every environment variable the provider's authentication reads, once, so the
// cases can run in parallel (t.Setenv forbids t.Parallel).
func TestMain(m *testing.M) {
	for _, k := range []string{
		"WAYFINDER_SERVER", "WAYFINDER_TOKEN", "WAYFINDER_TENANT", "WAYFINDER_CLIENT_ID", "WAYFINDER_CLIENT_SECRET",
		"WAYFINDER_SERVICE_ACCOUNT", "WAYFINDER_FEDERATED_TOKEN_FILE", "WAYFINDER_FEDERATED_TOKEN_REQUEST_URL",
		"WAYFINDER_FEDERATED_TOKEN_REQUEST_TOKEN", "WAYFINDER_FEDERATED_TOKEN_REQUEST_AUDIENCE",
		"ACTIONS_ID_TOKEN_REQUEST_URL", "ACTIONS_ID_TOKEN_REQUEST_TOKEN",
		"WAYFINDER_USE_CLI", "WAYFINDER_CLI_PROFILE", "WAYFINDER_CONFIG",
	} {
		if err := os.Unsetenv(k); err != nil {
			panic(err)
		}
	}
	// Never fall back to the developer's real wf CLI session.
	if err := os.Setenv("WAYFINDER_USE_CLI", "false"); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func TestGenerated(t *testing.T) {
	for _, c := range generatedCases {
		t.Run(c.Type, func(t *testing.T) {
			if c.Skip != "" {
				t.Skip(c.Skip)
			}

			t.Parallel()
			runCase(t, c, nil)
		})
	}
}

func TestGenerated_skipBudget(t *testing.T) {
	// Given the generated cases, When counting skips, Then at most 10% of kinds are skipped.
	var skipped []string

	for _, c := range generatedCases {
		if c.Skip != "" {
			skipped = append(skipped, c.Type+": "+c.Skip)
		}
	}

	t.Logf("generated plan-stability cases: %d kinds, %d skipped %v", len(generatedCases), len(skipped), skipped)

	if len(skipped)*10 > len(generatedCases) {
		t.Fatalf("%d of %d kinds skipped, budget is 10%%", len(skipped), len(generatedCases))
	}
}

// caseEnv is the real provider served in-process against a fake API that knows every kind of
// the embedded manifest, with server defaults for computed fields.
type caseEnv struct {
	fake      *fakeapi.Server
	ref       fakeapi.Ref
	addr      string
	provider  string
	factories map[string]func() (tfprotov6.ProviderServer, error)
}

func newCaseEnv(t *testing.T, c generatedCase) caseEnv {
	t.Helper()

	m, err := embedded()
	if err != nil {
		t.Fatal(err)
	}

	f := fakeapi.New(t)

	var kind *manifest.Kind

	for i, k := range m.Kinds {
		s := fakekinds.Spec(k)
		s.Defaults = fakekinds.ServerDefaults(k)
		f.RegisterKind(s)

		if k.Group == c.Group && k.Kind == c.Kind {
			kind = &m.Kinds[i]
		}
	}

	if kind == nil {
		t.Fatalf("embedded manifest has no kind %s/%s", c.Group, c.Kind)
	}

	return caseEnv{fake: f, ref: refFor(*kind, c.ImportID), addr: c.Type + ".example",
		provider: fmt.Sprintf("provider \"wayfinder\" {\n server = %q\n tenant = %q\n token = %q\n}\n", f.URL, testTenant, fakeapi.DefaultToken),
		factories: map[string]func() (tfprotov6.ProviderServer, error){
			"wayfinder": providerserver.NewProtocol6WithError(provider.New("test")()),
		}}
}

// refFor parses an import ID ([workspace/[environment/]]name[@version], D-ID) into a fake ref.
func refFor(k manifest.Kind, id string) fakeapi.Ref {
	r := fakeapi.Ref{Tenant: testTenant, Group: k.Group, Version: k.Version, APIPath: k.APIPath}
	id, r.ObjectVersion, _ = strings.Cut(id, "@")
	parts := strings.Split(id, "/")

	r.Name = parts[len(parts)-1]
	if len(parts) > 1 {
		r.Workspace = parts[0]
	}

	if len(parts) > 2 {
		r.Environment = parts[1]
	}

	return r
}

// leakHook, when set, adds steps after the first apply (the secretleakcheck self-test); cfg is
// the full configuration of the case.
type leakHook func(e caseEnv, cfg string) []resource.TestStep

// runCase applies the fixture, re-plans it empty, re-applies it as a no-op, imports and verifies
// it, then destroys it. Every plan and state is checked for the write-only literals, and every
// write-only attribute must be null in state. The fake adds an unmanaged server label to every
// object; the empty plans prove it never surfaces as drift (D-DRIFT).
func runCase(t *testing.T, c generatedCase, hook leakHook) {
	e := newCaseEnv(t, c)
	cfg := e.provider + c.Config
	noLeak := secretsAbsentPlan(c.Secrets)
	states := []statecheck.StateCheck{secretsAbsentState(c.Secrets)}
	// Import adopts every label/annotation key the server holds (the fake adds one), while state
	// written by create keeps only configured keys (D-DRIFT); the post-import plan step below
	// proves the adopted keys do not produce a diff.
	ignore := []string{"wait_for_ready", "timeouts", "status_json", "labels", "annotations"}

	for _, p := range c.WriteOnly {
		states = append(states, statecheck.ExpectKnownValue(e.addr, jsonPath(p), knownvalue.Null()))
		ignore = append(ignore, p, p+"_version")
	}

	steps := []resource.TestStep{{
		Config:            cfg,
		ConfigPlanChecks:  resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{noLeak}, PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan(), noLeak}},
		ConfigStateChecks: states,
		Check: resource.ComposeTestCheckFunc(
			secretsAbsentFlatmap(c.Secrets),
			resource.TestCheckResourceAttr(e.addr, "id", c.ImportID),
			resource.TestCheckNoResourceAttr(e.addr, "labels.%"),
			e.serverLabelPresent,
			e.secretsStored(c.Secrets),
		),
	}}
	if hook != nil {
		steps = append(steps, hook(e, cfg)...)
	}

	steps = append(steps,
		resource.TestStep{Config: cfg, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan(), noLeak}}, ConfigStateChecks: states},
		resource.TestStep{Config: cfg, ResourceName: e.addr, ImportState: true, ImportStateId: c.ImportID, ImportStateVerify: true, ImportStateVerifyIgnore: ignore},
		// Forget the object without destroying it, then import it through configuration: the
		// imported state must match the configuration except for the server-added label import
		// adopted and the _wo_version values state cannot know; applying that is the last change.
		resource.TestStep{Config: e.provider + fmt.Sprintf("removed {\n from = %s\n lifecycle {\n destroy = false\n }\n}\n", e.addr)},
		resource.TestStep{
			Config: cfg + fmt.Sprintf("import {\n to = %s\n id = %q\n}\n", e.addr, c.ImportID),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply:             []plancheck.PlanCheck{importOnlyAdoptsServerKeys(e.addr), noLeak},
				PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan(), noLeak},
			},
			ConfigStateChecks: states,
		},
	)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps:                    steps,
		CheckDestroy: func(*terraform.State) error {
			if _, ok := e.fake.Object(e.ref); ok {
				return fmt.Errorf("%s still exists after destroy", e.addr)
			}

			return nil
		},
	})
}

// serverLabelPresent proves the drift scenario is exercised: the stored object carries the
// fake's server-added label while the configuration manages no labels.
func (e caseEnv) serverLabelPresent(*terraform.State) error {
	obj, ok := e.fake.Object(e.ref)
	if !ok {
		return fmt.Errorf("%s not stored at %+v", e.addr, e.ref)
	}

	labels, _ := obj["metadata"].(map[string]any)["labels"].(map[string]any)
	if labels[fakeapi.ManagedByLabel] != "true" {
		return fmt.Errorf("server label %s missing: %v", fakeapi.ManagedByLabel, labels)
	}

	return nil
}

// secretsStored proves every write-only literal was sent to the API on create (the fake keeps
// secrets in its stored form and only blanks them in responses).
func (e caseEnv) secretsStored(secrets []string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		obj, _ := e.fake.Object(e.ref)

		b, err := json.Marshal(obj)
		if err != nil {
			return err
		}

		for _, s := range secrets {
			if !strings.Contains(string(b), s) {
				return fmt.Errorf("write-only literal %q was not sent to the API", s)
			}
		}

		return nil
	}
}

// jsonPath turns a flatmap path (a.0.b) into a state JSON path; numeric segments index lists.
func jsonPath(p string) tfjsonpath.Path {
	segs := strings.Split(p, ".")

	path := tfjsonpath.New(segs[0])
	for _, s := range segs[1:] {
		if i, err := strconv.Atoi(s); err == nil {
			path = path.AtSliceIndex(i)
		} else {
			path = path.AtMapKey(s)
		}
	}

	return path
}

// findSecret reports the first write-only literal present in the JSON form of doc.
func findSecret(doc any, secrets []string, where string) error {
	b, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", where, err)
	}

	if i := slices.IndexFunc(secrets, func(s string) bool { return strings.Contains(string(b), s) }); i >= 0 {
		return fmt.Errorf("write-only literal %q found in %s", secrets[i], where)
	}

	return nil
}
