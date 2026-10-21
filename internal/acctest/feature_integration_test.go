// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package acctest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/engine"
)

func TestAccFeatureIntegration_ai(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run acceptance tests")
	}

	PreCheck(t)

	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	// Given a fixed built-in feature name, NamePrefix and sweepers do not apply.
	// Cleanup restores its prior state, including a disabled feature's parameters.
	ref := client.Ref{Group: "integrations", Version: "v3", APIPath: "integrations", Name: "wayfinder.ai"}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	before, err := c.Get(ctx, ref)
	if err != nil && !errors.Is(err, client.ErrNotFound) {
		t.Fatalf("snapshot ai integration: %v", err)
	}

	existed := err == nil

	var (
		enabled   bool
		params    json.RawMessage
		hadParams bool
	)

	if existed {
		spec := before["spec"].(map[string]any)
		enabled = spec["enabled"].(bool)
		_, hadParams = spec["params"]

		params, err = json.Marshal(spec["params"])
		if err != nil {
			t.Fatalf("snapshot ai parameters: %v", err)
		}
	}

	t.Cleanup(func() {
		// The test context is canceled before cleanup; restoration needs its own bound.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		obj, err := c.Get(ctx, ref)
		if !existed && errors.Is(err, client.ErrNotFound) {
			return
		}

		if err != nil {
			t.Errorf("read ai integration for restoration: %v", err)
			return
		}

		if existed {
			spec := obj["spec"].(map[string]any)
			spec["enabled"] = enabled

			if hadParams {
				spec["params"] = params
			} else {
				delete(spec, "params")
			}

			if _, err := c.Update(ctx, ref, obj); err != nil {
				t.Errorf("restore ai integration: %v", err)
				return
			}

			restored, err := c.Get(ctx, ref)
			if err != nil {
				t.Errorf("verify restored ai integration: %v", err)
				return
			}

			restoredSpec := restored["spec"].(map[string]any)
			_, restoredHadParams := restoredSpec["params"]
			restoredParams, err := json.Marshal(restoredSpec["params"])

			// API reads decode JSON first; marshaling sorts keys and removes whitespace.
			if err != nil || restoredSpec["enabled"] != enabled || restoredHadParams != hadParams || !bytes.Equal(restoredParams, params) {
				t.Errorf("ai integration was not restored to its prior enabled state and exact parameters (err=%v)", err)
			}

			return
		}

		// DELETE is fixture cleanup only, never the provider's destroy behavior.
		// Wayfinder's Integration CheckDelete permits restoring an initially absent object.
		if err := c.Delete(ctx, ref); err != nil {
			t.Errorf("remove ai integration created by test: %v", err)
			return
		}

		waiter := engine.Waiter{Client: c, Poll: engine.DefaultPoll(), Target: engine.ErrorTarget{Kind: "Integration", ID: ref.Name}}
		if diags := waiter.Gone(ctx, ref); diags.HasError() {
			t.Errorf("wait for ai integration cleanup: %v", diags)
			return
		}

		if _, err := c.Get(ctx, ref); !errors.Is(err, client.ErrNotFound) {
			t.Errorf("ai integration still exists after restoring absence (err=%v)", err)
		}
	})

	const address = "wayfinder_feature_integration.test"

	const configured = `
resource "wayfinder_feature_integration" "test" {
  name   = "ai"
  params = { enable_wayfinder = jsonencode(false) }
}
`

	const withoutParams = `
resource "wayfinder_feature_integration" "test" {
  name = "ai"
}
`

	// When Terraform enables, repairs drift, removes a parameter, imports and destroys.
	// Then each apply stabilizes, and destroy disables rather than deletes the object.
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		CheckDestroy: func(*terraform.State) error {
			obj, err := c.Get(ctx, ref)
			if err != nil {
				return fmt.Errorf("read ai integration after destroy: %w", err)
			}

			if obj["spec"].(map[string]any)["enabled"] != false {
				return fmt.Errorf("ai integration remains enabled after destroy")
			}

			return nil
		},
		Steps: []resource.TestStep{
			{
				Config:           configured,
				ConfigPlanChecks: emptyAfterApply(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "id", "wayfinder.ai"),
					resource.TestCheckResourceAttr(address, "enabled", "true"),
					resource.TestCheckResourceAttr(address, "type", "feature"),
				),
			},
			{
				PreConfig: func() {
					obj, err := c.Get(ctx, ref)
					if err != nil {
						t.Fatalf("read ai integration to disable out of band: %v", err)
					}

					obj["spec"].(map[string]any)["enabled"] = false

					if _, err := c.Update(ctx, ref, obj); err != nil {
						t.Fatalf("disable ai integration out of band: %v", err)
					}
				},
				Config:           configured,
				ConfigPlanChecks: emptyAfterApply(plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)),
				Check:            resource.TestCheckResourceAttr(address, "enabled", "true"),
			},
			{
				Config:           withoutParams,
				ConfigPlanChecks: emptyAfterApply(plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)),
				Check: func(*terraform.State) error {
					obj, err := c.Get(ctx, ref)
					if err != nil {
						return fmt.Errorf("read ai integration after parameter removal: %w", err)
					}

					params, _ := obj["spec"].(map[string]any)["params"].(map[string]any)
					if _, ok := params["enable_wayfinder"]; ok {
						return fmt.Errorf("ai integration still has the removed enable_wayfinder parameter")
					}

					return nil
				},
			},
			{
				Config: withoutParams, ResourceName: address, ImportState: true, ImportStateId: "ai",
				ImportStateVerify: true, ImportStateVerifyIgnore: []string{"params", "keep_enabled_on_destroy"},
			},
		},
	})
}
