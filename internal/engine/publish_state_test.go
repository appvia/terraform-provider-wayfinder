// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	publishStateAddr = "wayfinder_blueprint.example"
	publishStateID   = "ws/example@1.2.3"
)

// publishStateConfig is a Blueprint resource with an optional publish_state
// line and any extra attribute lines.
func publishStateConfig(state string, extra ...string) string {
	lines := []string{`name = "example"`, `workspace = "ws"`, `version = "1.2.3"`, `source = {repository = "https://example.com/repo"}`}
	if state != "" {
		lines = append(lines, fmt.Sprintf("publish_state = %q", state))
	}

	return "resource \"wayfinder_blueprint\" \"example\" {\n " + strings.Join(append(lines, extra...), "\n ") + "\n}\n"
}

// sentPublishStates decodes metadata.publishState from every logged request
// with method; present is false when the key is absent from a body.
func sentPublishStates(t *testing.T, e testEnv, method string) []publishStateSent {
	t.Helper()

	var out []publishStateSent

	for _, q := range e.fake.Requests() {
		if q.Method != method {
			continue
		}

		var body struct {
			Metadata map[string]any `json:"metadata"`
		}
		if err := json.Unmarshal(q.Body, &body); err != nil {
			t.Fatalf("%s body %s: %v", method, q.Body, err)
		}

		v, ok := body.Metadata["publishState"]
		out = append(out, publishStateSent{present: ok, value: v})
	}

	return out
}

type publishStateSent struct {
	present bool
	value   any
}

func expectSent(t *testing.T, e testEnv, method string, want publishStateSent) {
	t.Helper()

	got := sentPublishStates(t, e, method)
	if len(got) != 1 || got[0] != want {
		t.Fatalf("%s metadata.publishState = %+v, want one %+v", method, got, want)
	}

	t.Logf("%s metadata.publishState = %+v", method, got[0])
}

// storedPublishState checks the fake's object, or its absence when want is nil.
func storedPublishState(e testEnv, want any) resource.TestCheckFunc {
	return func(*terraform.State) error {
		o, ok := e.fake.Object(e.ref("ws", "", "1.2.3"))
		if !ok {
			return fmt.Errorf("object missing")
		}

		md, _ := o["metadata"].(map[string]any)
		if got := md["publishState"]; got != want {
			return fmt.Errorf("stored metadata.publishState = %v, want %v", got, want)
		}

		return nil
	}
}

func emptyAfterApply() resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}
}

func TestPublishState_whenConfiguredOnCreate(t *testing.T) {
	for _, state := range []string{"Published", "Deprecated"} {
		t.Run(state, func(t *testing.T) {
			// Given a Blueprint configured with a publish state.
			e := newTestEnv(t, "Blueprint")
			// When Terraform creates it.
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{{
				Config:           publishStateConfig(state),
				ConfigPlanChecks: emptyAfterApply(),
				Check:            resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttr(publishStateAddr, "publish_state", state), storedPublishState(e, state)),
			}}})
			// Then the POST carried the value.
			expectSent(t, e, http.MethodPost, publishStateSent{present: true, value: state})
		})
	}
}

func TestPublishState_whenUnsetOnCreate(t *testing.T) {
	// Given a Blueprint with no publish state.
	e := newTestEnv(t, "Blueprint")
	// When Terraform creates it.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{{
		Config:           publishStateConfig(""),
		ConfigPlanChecks: emptyAfterApply(),
		Check:            resource.ComposeAggregateTestCheckFunc(resource.TestCheckNoResourceAttr(publishStateAddr, "publish_state"), storedPublishState(e, nil)),
	}}})
	// Then the provider chose no state.
	expectSent(t, e, http.MethodPost, publishStateSent{})
}

func TestPublishState_whenUnsetThenUnrelatedUpdate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		seed   bool
		stored any
	}{
		{name: "absent"},
		{name: "blank", seed: true, stored: ""},
		{name: "null", seed: true, stored: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given an unset publish_state over an absent, blank or JSON-null server value.
			e := newTestEnv(t, "Blueprint")
			first := resource.TestStep{Config: publishStateConfig(""), ConfigPlanChecks: emptyAfterApply()}

			if tc.seed {
				e.fake.Seed(e.ref("ws", "", "1.2.3"), map[string]any{
					"metadata": map[string]any{"publishState": tc.stored},
					"spec":     map[string]any{"source": map[string]any{"repository": "https://example.com/repo"}},
				})

				first = resource.TestStep{Config: publishStateConfig(""), ResourceName: publishStateAddr, ImportState: true, ImportStateId: publishStateID, ImportStatePersist: true}
			}
			// When only labels change.
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
				first,
				{
					PreConfig:        e.fake.ResetRequests,
					Config:           publishStateConfig("", `labels = {managed = "yes"}`),
					ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(publishStateAddr, plancheck.ResourceActionUpdate)}, PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
					Check:            resource.TestCheckNoResourceAttr(publishStateAddr, "publish_state"),
				},
			}})
			// Then the PUT wrote back what the server held, and nothing when it held nothing.
			expectSent(t, e, http.MethodPut, publishStateSent{present: tc.seed, value: tc.stored})
		})
	}
}

func TestPublishState_whenChangedInPlace(t *testing.T) {
	// Given a published Blueprint.
	e := newTestEnv(t, "Blueprint")
	// When the config changes it to Draft.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: publishStateConfig("Published"), ConfigPlanChecks: emptyAfterApply()},
		{
			Config:           publishStateConfig("Draft"),
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(publishStateAddr, plancheck.ResourceActionUpdate)}, PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			Check:            resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttr(publishStateAddr, "publish_state", "Draft"), storedPublishState(e, "Draft")),
		},
	}})
	// Then it is updated in place, never recreated.
	expectSent(t, e, http.MethodPut, publishStateSent{present: true, value: "Draft"})

	if n := e.fake.CountRequests(http.MethodPost, "") - 1; n != 0 {
		t.Fatalf("extra POST count = %d", n)
	}
}

func TestPublishState_whenRemovedFromConfig(t *testing.T) {
	// Given a published Blueprint.
	e := newTestEnv(t, "Blueprint")
	// When publish_state is dropped from config, then labels change.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: publishStateConfig("Published"), ConfigPlanChecks: emptyAfterApply()},
		{Config: publishStateConfig(""), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
		{
			Config:           publishStateConfig("", `labels = {managed = "yes"}`),
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(publishStateAddr, plancheck.ResourceActionUpdate)}, PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			Check:            resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttr(publishStateAddr, "publish_state", "Published"), storedPublishState(e, "Published")),
		},
	}})
	// Then the server value was kept on the update.
	expectSent(t, e, http.MethodPut, publishStateSent{present: true, value: "Published"})
}

func TestPublishState_whenUnsetAndPublishedOutOfBand(t *testing.T) {
	// Given an unset publish_state.
	e := newTestEnv(t, "Blueprint")
	// When the version is published outside Terraform (as wf publish does).
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: publishStateConfig(""), ConfigPlanChecks: emptyAfterApply()},
		{
			PreConfig: func() {
				e.fake.ResetRequests()
				e.fake.Mutate(e.ref("ws", "", "1.2.3"), func(o map[string]any) {
					o["metadata"].(map[string]any)["publishState"] = "Published"
				})
			},
			Config:           publishStateConfig(""),
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			Check:            resource.TestCheckResourceAttr(publishStateAddr, "publish_state", "Published"),
		},
	}})
	// Then refresh tracks it and nothing is written.
	if n := e.fake.CountRequests(http.MethodPut, ""); n != 0 {
		t.Fatalf("PUT count = %d", n)
	}
}

func TestPublishState_whenConfiguredAndChangedOutOfBand(t *testing.T) {
	// Given a configured Published state.
	e := newTestEnv(t, "Blueprint")
	cfg := publishStateConfig("Published")
	// When it is set to Draft outside Terraform.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
		{Config: cfg, ConfigPlanChecks: emptyAfterApply()},
		{
			PreConfig: func() {
				e.fake.ResetRequests()
				e.fake.Mutate(e.ref("ws", "", "1.2.3"), func(o map[string]any) {
					o["metadata"].(map[string]any)["publishState"] = "Draft"
				})
			},
			Config:           cfg,
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(publishStateAddr, plancheck.ResourceActionUpdate)}},
			ExpectError:      modifiedRegex(),
		},
		// Then refresh reports the drift and the server keeps Draft.
		{RefreshState: true, ExpectNonEmptyPlan: true, Check: resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttr(publishStateAddr, "publish_state", "Draft"), storedPublishState(e, "Draft"))},
	}})

	if n := e.fake.CountRequests(http.MethodPut, ""); n != 0 {
		t.Fatalf("PUT count = %d", n)
	}
}

func TestPublishState_whenImported(t *testing.T) {
	for _, tc := range []struct{ name, stored, configured string }{
		{"draft", "Draft", "Draft"},
		{"blank", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given a server object with a Draft or blank publish state.
			e := newTestEnv(t, "Blueprint")
			e.fake.Seed(e.ref("ws", "", "1.2.3"), map[string]any{
				"metadata": map[string]any{"publishState": tc.stored},
				"spec":     map[string]any{"source": map[string]any{"repository": "https://example.com/repo"}},
			})

			check := resource.TestCheckNoResourceAttr(publishStateAddr, "publish_state")
			if tc.stored != "" {
				check = resource.TestCheckResourceAttr(publishStateAddr, "publish_state", tc.stored)
			}

			cfg := publishStateConfig(tc.configured)
			// When importing it and planning.
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
				{Config: cfg, ResourceName: publishStateAddr, ImportState: true, ImportStateId: publishStateID, ImportStatePersist: true},
				{Config: cfg, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}, Check: check},
			}})
			// Then the state is adopted (null when blank) without writes.
			if n := e.fake.CountRequests(http.MethodPost, "") + e.fake.CountRequests(http.MethodPut, ""); n != 0 {
				t.Fatalf("adoption writes = %d", n)
			}
		})
	}
}

func TestPublishState_whenDataSourceRead(t *testing.T) {
	for _, tc := range []struct{ name, stored string }{{"published", "Published"}, {"blank", ""}} {
		t.Run(tc.name, func(t *testing.T) {
			// Given a server object with a Published or blank publish state.
			e := newTestEnv(t, "Blueprint")
			e.fake.Seed(e.ref("ws", "", "1.2.3"), map[string]any{"metadata": map[string]any{"publishState": tc.stored}, "spec": map[string]any{}})

			addr := "data." + publishStateAddr

			check := resource.TestCheckNoResourceAttr(addr, "publish_state")
			if tc.stored != "" {
				check = resource.TestCheckResourceAttr(addr, "publish_state", tc.stored)
			}
			// When the data source reads it.
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{{
				Config: "data \"wayfinder_blueprint\" \"example\" {\n name = \"example\"\n workspace = \"ws\"\n version = \"1.2.3\"\n}\n",
				// Then publish_state is the server value, null when blank.
				Check: check,
			}}})
		})
	}
}

func TestPublishState_whenInvalidValue(t *testing.T) {
	// Given a lower-case publish state.
	e := newTestEnv(t, "Blueprint")
	// When planning.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{{
		Config:      publishStateConfig("published"),
		ExpectError: regexp.MustCompile(`(?s)publish_state.*value must be one of`),
	}}})
	// Then validation fails before any write.
	if n := e.fake.CountRequests(http.MethodPost, "") + e.fake.CountRequests(http.MethodPut, ""); n != 0 {
		t.Fatalf("writes = %d", n)
	}
}
