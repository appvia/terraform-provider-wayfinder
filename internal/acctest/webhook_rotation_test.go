// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package acctest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

func TestAccWebhookRotation(t *testing.T) {
	name, secret := RandomName(), RandomName()
	config := func(trigger string) string {
		return webhookHCL(name) + fmt.Sprintf(`
resource "wayfinder_webhook_rotation" "test" {
  webhook  = wayfinder_webhook.test.name
  triggers = { n = %q }
}

ephemeral "wayfinder_webhook_ingest_url" "test" {
  name = wayfinder_webhook_rotation.test.name
}

resource "wayfinder_secret" "test" {
  key              = %q
  value_wo         = jsonencode(ephemeral.wayfinder_webhook_ingest_url.test.url)
  value_wo_version = wayfinder_webhook_rotation.test.rotation_id
}
`, trigger, secret) + fmt.Sprintf(echoHCL, "ephemeral.wayfinder_webhook_ingest_url.test.url")
	}

	check := &webhookRotationState{}
	tc := ephemeralCase(t,
		resource.TestStep{
			// The first apply must create, rotate and read without an out-of-band rotation.
			Config:            config("1"),
			ConfigStateChecks: []statecheck.StateCheck{check},
		},
		resource.TestStep{
			// Echo is immutable, so recreate only the probe to capture this apply's URL.
			Taint:  []string{"echo.result"},
			Config: config("2"),
			ConfigPlanChecks: emptyAfterApply(
				plancheck.ExpectResourceAction("wayfinder_webhook_rotation.test", plancheck.ResourceActionReplace),
				plancheck.ExpectResourceAction("wayfinder_secret.test", plancheck.ResourceActionUpdate),
			),
			ConfigStateChecks: []statecheck.StateCheck{check},
		},
		resource.TestStep{
			ResourceName:            "wayfinder_webhook_rotation.test",
			ImportState:             true,
			ImportStateId:           name,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: []string{"triggers", "rotation_id", "rotated_at"},
		},
	)
	tc.PreCheck = func() {
		preCheckEphemeral(t)

		c, err := NewClient()
		if err != nil {
			t.Fatal(err)
		}

		check.tenant, check.server = c.Tenant, strings.TrimRight(c.BaseURL, "/")
	}
	tc.CheckDestroy = resource.ComposeTestCheckFunc(checkDestroyed, func(*terraform.State) error {
		c, err := NewClient()
		if err != nil {
			return err
		}

		path := "/api/v3/general/tenants/" + url.PathEscape(c.Tenant) + "/vars/secrets/" + url.PathEscape(secret)
		if err := c.DoJSON(context.Background(), http.MethodGet, path, nil, nil); !errors.Is(err, client.ErrNotFound) {
			return fmt.Errorf("tenant secret still exists after destroy (err=%v)", err)
		}

		t.Log("teardown verified: webhook, integration and tenant secret absent")

		return nil
	})
	resource.Test(t, tc)
}

func TestAccWebhookRotation_webhookGone(t *testing.T) {
	name := RandomName()
	config := webhookHCL(name) + `
resource "wayfinder_webhook_rotation" "test" {
  webhook = wayfinder_webhook.test.name
}
`
	tc := ephemeralCase(t,
		resource.TestStep{Config: config},
		resource.TestStep{
			PreConfig: func() {
				c, err := NewClient()
				if err != nil {
					t.Fatal(err)
				}

				m, err := manifest.LoadEmbedded()
				if err != nil {
					t.Fatal(err)
				}

				k, ok := m.FindKind("integrations", "Webhook")
				if !ok {
					t.Fatal("embedded manifest has no Webhook kind")
				}

				ref := client.Ref{Group: k.Group, Version: k.Version, APIPath: k.APIPath, Name: name, Versioned: k.Versioned}
				if err := c.Delete(context.Background(), ref); err != nil {
					t.Fatal(err)
				}

				if _, err := c.Get(context.Background(), ref); !errors.Is(err, client.ErrNotFound) {
					t.Fatalf("webhook still exists after API deletion (err=%v)", err)
				}
			},
			Config: config,
			ConfigPlanChecks: emptyAfterApply(
				plancheck.ExpectResourceAction("wayfinder_webhook_rotation.test", plancheck.ResourceActionCreate),
			),
		},
	)
	tc.CheckDestroy = func(s *terraform.State) error {
		if err := checkDestroyed(s); err != nil {
			return err
		}

		t.Log("teardown verified: webhook and integration absent")

		return nil
	}
	resource.Test(t, tc)
}

// webhookRotationState captures credentials only in memory; assertion errors never include them.
// Reusing the check across steps proves both the URL and the version change without sleeps.
type webhookRotationState struct {
	tenant, server  string
	url, rotationID string
}

func (c *webhookRotationState) CheckState(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	var nextURL, nextID string

	statecheck.ExpectKnownValue("echo.result", tfjsonpath.New("data"), knownvalue.StringFunc(func(got string) error {
		if !strings.HasPrefix(got, c.server+"/") || !strings.Contains(got, "/"+c.tenant+"/") {
			return errors.New("ingest URL must contain the configured server and tenant")
		}

		if got == c.url {
			return errors.New("ingest URL did not change after rotation")
		}

		nextURL = got

		return nil
	})).CheckState(ctx, req, resp)

	if resp.Error != nil {
		return
	}

	statecheck.ExpectKnownValue("wayfinder_webhook_rotation.test", tfjsonpath.New("rotation_id"), knownvalue.StringFunc(func(got string) error {
		if got == "" || got == c.rotationID {
			return errors.New("rotation_id must be nonempty and change after rotation")
		}

		nextID = got

		return nil
	})).CheckState(ctx, req, resp)

	if resp.Error != nil {
		return
	}

	secrets := []string{nextURL}
	if c.url != "" {
		secrets = append(secrets, c.url)
	}

	for _, rs := range req.State.Values.RootModule.Resources {
		if rs.Address == "wayfinder_webhook_rotation.test" {
			if err := containsAny(fmt.Append(nil, rs.AttributeValues), secrets); err != nil {
				resp.Error = fmt.Errorf("rotation state: %w", err)
				return
			}
		}
	}

	statecheck.ExpectKnownValue("wayfinder_secret.test", tfjsonpath.New("value_wo_version"),
		knownvalue.StringExact(nextID)).CheckState(ctx, req, resp)

	if resp.Error == nil {
		c.url, c.rotationID = nextURL, nextID
	}
}
