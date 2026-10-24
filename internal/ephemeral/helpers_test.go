// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package ephemeral_test

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi/fakekinds"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/appvia/terraform-provider-wayfinder/internal/provider"
)

type actionEnv struct {
	fake  *fakeapi.Server
	ref   client.Ref
	url   string
	token string
}

func newActionEnv(t *testing.T, workspace string) actionEnv {
	t.Helper()
	fake := fakeapi.New(t)
	token := "integration-" + t.Name()
	fake.AcceptToken(token)
	fake.RevokeToken(fakeapi.DefaultToken)

	embedded, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}

	fakekinds.RegisterAll(fake, embedded)

	k, ok := embedded.FindKind("integrations", "Webhook")
	if !ok {
		t.Fatal("embedded Webhook missing")
	}

	return actionEnv{
		fake: fake,
		ref: client.Ref{
			Group: k.Group, Version: k.Version, APIPath: k.APIPath, Versioned: k.Versioned,
			Name: "hook", Workspace: workspace,
		},
		// Different per-server credentials expose accidental sharing between parallel tests.
		url:   fake.URL + "/ingest/acme/private-credential",
		token: token,
	}
}

func (e actionEnv) config(resourceType string) string {
	scope := ""
	if e.ref.Workspace != "" {
		scope = fmt.Sprintf("workspace = %q", e.ref.Workspace)
	}

	extra := ""
	if resourceType == "resource_action" {
		extra = fmt.Sprintf("type = %q\n action = \"ingesturl\"\n response_export_values = [\"url\"]",
			e.ref.Group+"/"+e.ref.Version+"/Webhook")
	}

	return fmt.Sprintf(`
provider "wayfinder" {
  server  = %q
  tenant  = "acme"
  token   = %q
  use_cli = false
}
ephemeral "wayfinder_%s" "test" {
  name = "hook"
  %s
  %s
}
`, e.fake.URL, e.token, resourceType, scope, extra)
}

func actionOutput(resourceType string) string {
	if resourceType == "resource_action" {
		return "ephemeral.wayfinder_resource_action.test.output.url"
	}

	return "ephemeral.wayfinder_webhook_ingest_url.test.url"
}

func (e actionEnv) testCase(steps ...resource.TestStep) resource.TestCase {
	return resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"wayfinder": providerserver.NewProtocol6WithError(provider.New("test")()),
			"echo":      echoprovider.NewProviderServer(),
		},
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		Steps:                  steps,
	}
}

func checkActionRequests(t *testing.T, e actionEnv, status int) {
	t.Helper()

	item, err := e.ref.ItemPath("acme")
	if err != nil {
		t.Fatal(err)
	}

	requests := e.fake.Requests()
	if len(requests) == 0 {
		t.Fatal("no authenticated action request reached fakeapi")
	}

	for _, request := range requests {
		if request.Method != http.MethodGet || request.Path != item+"/ingesturl" || request.Status != status {
			t.Fatalf("unexpected request: %s %s -> %d", request.Method, request.Path, request.Status)
		}

		if len(request.Body) != 0 {
			t.Fatal("request bodies must not be recorded")
		}
	}

	// The fake rejects missing/incorrect bearer headers before routing. A 200/403/404
	// action response after revoking its default token proves the configured token arrived.
	t.Logf("%d authenticated GET requests to %s/ingesturl -> %d", len(requests), item, status)
}

func echoAction(t *testing.T, resourceType, workspace string) {
	t.Helper()
	e := newActionEnv(t, workspace)
	e.fake.SetSubresource(e.ref, "ingesturl", http.StatusOK, map[string]string{"url": e.url, "ignored": "not exported"})
	cfg := e.config(resourceType) + fmt.Sprintf(`
provider "echo" { data = %s }
resource "echo" "result" {}
`, actionOutput(resourceType))
	control := credentialScan{literal: e.url, present: true}

	// Echo deliberately persists the credential: value correctness and real negative
	// controls for state and saved-plan scanners, never a non-persistence scenario.
	resource.UnitTest(t, e.testCase(resource.TestStep{
		Config: cfg,
		ConfigStateChecks: []statecheck.StateCheck{
			statecheck.ExpectKnownValue("echo.result", tfjsonpath.New("data"), knownvalue.StringExact(e.url)), control,
		},
		ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{control}},
	}))
	checkActionRequests(t, e, http.StatusOK)
	t.Log("echo value verified; state and saved-plan negative controls detected the credential")
}

func failedAction(t *testing.T, resourceType string) {
	t.Helper()

	for _, tc := range []struct {
		name   string
		status int
		want   string
	}{
		{"not_rotated", http.StatusNotFound, `(?s)Wayfinder action not found.*wf rotate webhook hook`},
		{"forbidden", http.StatusForbidden, `(?s)Wayfinder action permission denied.*webhooks/ingesturl permission`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given an action whose error body must not become a diagnostic.
			e := newActionEnv(t, "")
			e.fake.SetSubresource(e.ref, "ingesturl", tc.status, map[string]string{"message": e.url})
			// When Terraform opens it, then the specific actionable diagnostic is required.
			resource.UnitTest(t, e.testCase(resource.TestStep{
				Config: e.config(resourceType), PlanOnly: true, ExpectError: regexp.MustCompile(tc.want),
			}))
			checkActionRequests(t, e, tc.status)
		})
	}
}

func unknownAction(t *testing.T, resourceType string) {
	t.Helper()
	// Given a name computed only after creation of a Terraform built-in resource.
	e := newActionEnv(t, "")
	cfg := strings.Replace(e.config(resourceType), `name = "hook"`, `name = terraform_data.name.output`, 1)
	cfg += fmt.Sprintf(`
resource "terraform_data" "name" { input = "hook" }
provider "echo" { data = %s }
resource "echo" "result" {}
`, actionOutput(resourceType))
	// When only planning, then the downstream value remains unknown without an API call.
	resource.UnitTest(t, e.testCase(resource.TestStep{
		Config: cfg, PlanOnly: true, ExpectNonEmptyPlan: true,
		ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPreRefresh: []plancheck.PlanCheck{
			plancheck.ExpectUnknownValue("echo.result", tfjsonpath.New("data")),
		}},
	}))

	if len(e.fake.Requests()) != 0 {
		t.Fatal("unknown name caused an API request during plan")
	}

	t.Log("unknown dependency planned successfully; downstream data unknown; zero API requests")
}
