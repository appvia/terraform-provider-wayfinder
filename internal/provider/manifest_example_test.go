package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

const exampleWebhookSecret = "example-webhook-secret-value"

func TestManifestExample_yamlencode(t *testing.T) {
	// Given: the documented wayfinder_manifest example, read from disk so docs and tests cannot drift.
	example, err := os.ReadFile("../../examples/resources/wayfinder_manifest/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	e := newTestEnv(t)

	// When: applying it against the fake API, supplying the ephemeral webhook_secret variable.
	// Then: it applies, the objects exist, the follow-up plan is empty, and the fake stored the secret.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{{
			Config:           e.providerBlock() + string(example),
			ConfigVariables:  config.Variables{"webhook_secret": config.StringVariable(exampleWebhookSecret)},
			ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("wayfinder_manifest.developers", "id"),
				resource.TestCheckResourceAttrSet("wayfinder_manifest.github_webhook", "id"),
				checkExampleWebhookSecretStored(e),
			),
		}},
	})
}

func checkExampleWebhookSecretStored(e testEnv) resource.TestCheckFunc {
	return func(*terraform.State) error {
		var k manifest.Kind
		for _, c := range e.m.Kinds {
			if c.Kind == "Webhook" {
				k = c
				break
			}
		}
		obj, ok := e.fake.Object(e.ref(k, "plat", "github"))
		if !ok {
			return fmt.Errorf("fake has no Webhook plat/github")
		}
		security, err := nestedMap(obj, "spec", "security")
		if err != nil {
			return err
		}
		if got := security["secret"]; got != exampleWebhookSecret {
			return fmt.Errorf("stored webhook secret = %v; want the value supplied via webhook_secret", got)
		}
		return nil
	}
}
