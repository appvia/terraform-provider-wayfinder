// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package acctest

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

// preCheckEphemeral accepts either the WAYFINDER_* environment or a logged-in wf CLI session,
// because the provider itself resolves credentials either way.
func preCheckEphemeral(t *testing.T) {
	t.Helper()

	if len(LoadEnv(os.Getenv).Missing()) == 0 {
		PreCheck(t)
		return
	}

	if _, err := NewClient(); err != nil {
		t.Fatalf("acceptance tests need WAYFINDER_* variables or a logged-in wf CLI session: %v", err)
	}
}

func webhookHCL(name string) string {
	return fmt.Sprintf(`
resource "wayfinder_integration" "test" {
  name = %q
}

resource "wayfinder_webhook" "test" {
  name            = %q
  integration_ref = wayfinder_integration.test.name
  security = {
    algorithm = "none"
  }
}
`, name, name)
}

// rotateWebhook mints the webhook's ingest URL through a direct API call. Rotation is a write
// and must never happen through an ephemeral resource; the returned URL is the expected value.
func rotateWebhook(t *testing.T, name string) string {
	t.Helper()

	m, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}

	k, ok := m.FindKind("integrations", "Webhook")
	if !ok {
		t.Fatal("embedded manifest has no Webhook kind")
	}

	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	item, err := client.Ref{Group: k.Group, Version: k.Version, APIPath: k.APIPath, Name: name}.ItemPath(c.Tenant)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		URL string `json:"url"`
	}
	if err := c.DoJSON(context.Background(), http.MethodPost, item+"/rotate", struct{}{}, &out); err != nil {
		t.Fatalf("rotating webhook %s: %v", name, err)
	}

	if out.URL == "" {
		t.Fatalf("rotating webhook %s returned no url", name)
	}

	return out.URL
}

// ingestURL checks the echoed ephemeral value against the URL rotate returned.
type ingestURL struct{ want, tenant, server *string }

func (c ingestURL) CheckValue(v any) error {
	got, ok := v.(string)
	if !ok {
		return fmt.Errorf("ingest URL is %T, want string", v)
	}

	switch {
	case got != *c.want:
		return fmt.Errorf("ingest URL (length %d) differs from the URL rotate returned (length %d)", len(got), len(*c.want))
	case !strings.HasPrefix(got, *c.server+"/"):
		return fmt.Errorf("ingest URL does not start with the server URL %s", *c.server)
	case !strings.Contains(got, "/"+*c.tenant+"/"):
		return fmt.Errorf("ingest URL does not contain the tenant %q", *c.tenant)
	}

	return nil
}

func (c ingestURL) String() string { return "webhook ingest URL" }

func ephemeralCase(t *testing.T, steps ...resource.TestStep) resource.TestCase {
	return resource.TestCase{
		PreCheck: func() { preCheckEphemeral(t) },
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"wayfinder": ProtoV6ProviderFactories["wayfinder"],
			"echo":      echoprovider.NewProviderServer(),
		},
		// Ephemeral resources and write-only attributes need Terraform 1.11.
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		CheckDestroy:           checkDestroyed,
		Steps:                  steps,
	}
}

const echoHCL = `
provider "echo" { data = %s }
resource "echo" "result" {}
`

// ingestURLAcc creates a webhook, rotates it, then reads its URL through the ephemeral resource
// described by ephemeralHCL and its output expression.
func ingestURLAcc(t *testing.T, ephemeralHCL, output string) {
	t.Helper()

	name := RandomName()

	var want, tenant, server string

	tc := ephemeralCase(t,
		resource.TestStep{Config: webhookHCL(name)},
		resource.TestStep{
			PreConfig: func() {
				c, err := NewClient()
				if err != nil {
					t.Fatal(err)
				}

				tenant, server = c.Tenant, strings.TrimRight(c.BaseURL, "/")
				want = rotateWebhook(t, name)
			},
			Config: webhookHCL(name) + fmt.Sprintf(ephemeralHCL, name) + fmt.Sprintf(echoHCL, output),
			ConfigStateChecks: []statecheck.StateCheck{statecheck.ExpectKnownValue("echo.result", tfjsonpath.New("data"),
				knownvalue.Check(ingestURL{want: &want, tenant: &tenant, server: &server}))},
		},
	)
	resource.Test(t, tc)
}

func TestAccWebhookIngestURL(t *testing.T) {
	ingestURLAcc(t, `
ephemeral "wayfinder_webhook_ingest_url" "test" {
  name = %q
  depends_on = [wayfinder_webhook.test]
}
`, "ephemeral.wayfinder_webhook_ingest_url.test.url")
}

func TestAccResourceAction_webhookIngestURL(t *testing.T) {
	ingestURLAcc(t, `
ephemeral "wayfinder_resource_action" "test" {
  type                  = "integrations/v3/Webhook"
  name                  = %q
  action                = "ingesturl"
  response_export_values = ["url"]
  depends_on            = [wayfinder_webhook.test]
}
`, "ephemeral.wayfinder_resource_action.test.output.url")
}

// TestAccWebhookIngestURL_notRotated: a webhook that was never rotated has no ingest URL, and
// the 404 carries the rotate hint.
func TestAccWebhookIngestURL_notRotated(t *testing.T) {
	name := RandomName()
	tc := ephemeralCase(t,
		resource.TestStep{Config: webhookHCL(name)},
		resource.TestStep{
			Config: webhookHCL(name) + fmt.Sprintf(`
ephemeral "wayfinder_webhook_ingest_url" "test" {
  name = %q
  depends_on = [wayfinder_webhook.test]
}
`, name) + fmt.Sprintf(echoHCL, "ephemeral.wayfinder_webhook_ingest_url.test.url"),
			ExpectError: regexp.MustCompile(`(?s)Wayfinder action not found.*wf rotate webhook ` + name),
		},
	)
	resource.Test(t, tc)
}
