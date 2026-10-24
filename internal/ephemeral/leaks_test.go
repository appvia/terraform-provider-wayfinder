// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package ephemeral_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

// credentialScan checks the entire JSON artifact. The credential is supplied only
// by fakeapi, never as a configuration literal, so no plan section is excluded.
type credentialScan struct {
	literal string
	present bool
}

func (s credentialScan) check(data []byte, where string) error {
	found := bytes.Contains(data, []byte(s.literal))
	if found != s.present {
		return fmt.Errorf("credential presence in %s = %t, want %t", where, found, s.present)
	}

	return nil
}

func (s credentialScan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	data, err := json.Marshal(req.Plan)
	if err != nil {
		resp.Error = err
		return
	}

	resp.Error = s.check(data, "saved plan JSON")
}

func (s credentialScan) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	data, err := json.Marshal(req.State)
	if err != nil {
		resp.Error = err
		return
	}

	resp.Error = s.check(data, "state JSON")
}

func writeOnlyAction(t *testing.T, resourceType string) {
	t.Helper()
	// Given a credential known only to this fake and a write-only consumer.
	e := newActionEnv(t, "")
	e.fake.SetSubresource(e.ref, "ingesturl", http.StatusOK, map[string]string{"url": e.url})

	logPath := filepath.Join(t.TempDir(), "provider.log")
	t.Setenv("TF_LOG", "TRACE")
	t.Setenv("TF_LOG_PATH", logPath)
	t.Setenv("TF_LOG_PATH_MASK", "")
	t.Setenv("TF_ACC_LOG", "TRACE")
	t.Setenv("TF_LOG_CORE", "")
	t.Setenv("TF_ACC_LOG_PATH", logPath)

	scan := credentialScan{literal: e.url}
	cfg := e.config(resourceType) + fmt.Sprintf(`
resource "wayfinder_secret" "sink" {
  key = "ingest-url"
  value_wo = jsonencode(%s)
  value_wo_version = "1"
}
`, actionOutput(resourceType))

	// When Terraform plans/applies with TRACE logging, then state, saved plans
	// (including prior state/configuration) and provider logs omit the credential.
	resource.UnitTest(t, e.testCase(resource.TestStep{
		Config: cfg, ConfigStateChecks: []statecheck.StateCheck{scan},
		ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{scan}, PostApplyPostRefresh: []plancheck.PlanCheck{scan},
		},
		Check: func(state *terraform.State) error {
			stored, ok := e.fake.Secret(fakeapi.VarScope{Tenant: "acme"}, "ingest-url")

			want, err := json.Marshal(e.url)
			if err != nil {
				return err
			}

			if !ok || !bytes.Equal(stored, want) {
				return fmt.Errorf("write-only consumer did not receive the credential")
			}

			data, err := json.Marshal(state)
			if err != nil {
				return err
			}

			return scan.check(data, "flatmap state")
		},
	}))

	logs, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	// Reject an empty or unrelated log file, not just the absence of the literal.
	if !bytes.Contains(logs, []byte("OpenEphemeralResource")) || !bytes.Contains(logs, []byte("[TRACE]")) {
		t.Fatal("TRACE provider OpenEphemeralResource logs were not captured")
	}

	if err := scan.check(logs, "TRACE logs"); err != nil {
		t.Fatal(err)
	}

	// A log leak must fail the same scanner; never print the injected credential.
	if err := scan.check(append(logs, []byte(e.url)...), "negative control logs"); err == nil {
		t.Fatal("log negative control did not detect the credential")
	}

	t.Log("write-only delivery verified; state, saved plan and TRACE logs credential-free; log negative control detected")
}
