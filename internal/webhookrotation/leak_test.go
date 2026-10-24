// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package webhookrotation_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// canaryScan fails when the canary appears in a whole JSON artifact.
type canaryScan struct{}

func (canaryScan) check(data []byte, where string) error {
	if bytes.Contains(data, []byte(canaryWord)) {
		return fmt.Errorf("%s contains the rotated URL", where)
	}

	return nil
}

func (s canaryScan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	data, err := json.Marshal(req.Plan)
	if err != nil {
		resp.Error = err
		return
	}

	resp.Error = s.check(data, "saved plan JSON")
}

func (s canaryScan) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	data, err := json.Marshal(req.State)
	if err != nil {
		resp.Error = err
		return
	}

	resp.Error = s.check(data, "state JSON")
}

// Not parallel: it sets TRACE logging through the environment.
func TestLeak_whenRotatedAndReplaced(t *testing.T) {
	// Given a rotate response carrying the canary URL and TRACE logs captured to a file.
	e := newEnv(t)
	e.seedWebhook("", "hook")
	e.allowRotate("", "hook")

	logPath := filepath.Join(t.TempDir(), "provider.log")
	t.Setenv("TF_LOG", "TRACE")
	t.Setenv("TF_LOG_PATH", logPath)
	t.Setenv("TF_LOG_PATH_MASK", "")
	t.Setenv("TF_ACC_LOG", "TRACE")
	t.Setenv("TF_LOG_CORE", "")
	t.Setenv("TF_ACC_LOG_PATH", logPath)

	scan := canaryScan{}
	flatmap := func(state *terraform.State) error {
		data, err := json.Marshal(state)
		if err != nil {
			return err
		}

		return scan.check(data, "flatmap state")
	}
	step := func(triggers string, rotations int) resource.TestStep {
		return resource.TestStep{
			Config:            rotationHCL(triggers),
			ConfigStateChecks: []statecheck.StateCheck{scan},
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{scan}, PostApplyPostRefresh: []plancheck.PlanCheck{scan},
			},
			Check: resource.ComposeAggregateTestCheckFunc(flatmap, expectRotations(t, e, rotations)),
		}
	}

	// When the rotation is created and then replaced through its triggers.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps:                    []resource.TestStep{step("1", 1), step("2", 2)},
	})

	// Then neither state, saved plans nor provider TRACE logs contain the URL.
	logs, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	// Reject an empty or unrelated log: the provider's own rotate request line must be present.
	if !bytes.Contains(logs, []byte("[TRACE]")) || !bytes.Contains(logs, []byte(e.rotatePath(t, "", "hook"))) {
		t.Fatal("TRACE logs with the provider rotate request were not captured")
	}

	if err := scan.check(logs, "TRACE logs"); err != nil {
		t.Fatal(err)
	}

	if err := scan.check(append(logs, []byte(leakCanary)...), "negative control logs"); err == nil {
		t.Fatal("log negative control did not detect the URL")
	}
}
