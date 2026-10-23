// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package generated_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appvia/terraform-provider-wayfinder/internal/generated"
)

func TestPublishStateSchema_whenGeneratedPlanKinds(t *testing.T) {
	ctx := context.Background()
	useState := stringplanmodifier.UseStateForUnknown().Description(ctx)

	for _, tc := range []struct {
		name     string
		resource rschema.Schema
		data     dschema.Schema
	}{
		{"wayfinder_cloud_resource_plan", generated.ResourceSchema_cloud_resource_plan(ctx), generated.DataSourceSchema_cloud_resource_plan(ctx)},
		{"wayfinder_kube_resource_plan", generated.ResourceSchema_kube_resource_plan(ctx), generated.DataSourceSchema_kube_resource_plan(ctx)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given
			a, ok := tc.resource.Attributes["publish_state"].(rschema.StringAttribute)
			if !ok {
				t.Fatal("resource has no string publish_state")
			}
			// Then: optional+computed, no default, only UseStateForUnknown (never RequiresReplace)
			if !a.Optional || !a.Computed || a.Required || a.Default != nil {
				t.Fatalf("publish_state optional=%v computed=%v required=%v default=%v", a.Optional, a.Computed, a.Required, a.Default)
			}

			if len(a.PlanModifiers) != 1 || a.PlanModifiers[0].Description(ctx) != useState {
				t.Fatalf("plan modifiers = %d, want only UseStateForUnknown", len(a.PlanModifiers))
			}

			for value, valid := range map[string]bool{"Draft": true, "Published": true, "Deprecated": true, "published": false, "": false} {
				var failed bool

				for _, v := range a.Validators {
					var resp validator.StringResponse
					v.ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringValue(value)}, &resp)
					failed = failed || resp.Diagnostics.HasError()
				}

				if failed == valid {
					t.Errorf("publish_state %q: validation failed=%v, want valid=%v", value, failed, valid)
				}
			}

			d, ok := tc.data.Attributes["publish_state"].(dschema.StringAttribute)
			if !ok {
				t.Fatal("data source has no string publish_state")
			}

			if !d.Computed || d.Optional || d.Required {
				t.Fatalf("data source publish_state computed=%v optional=%v required=%v", d.Computed, d.Optional, d.Required)
			}
		})
	}

	t.Run("wayfinder_workspace", func(t *testing.T) {
		t.Parallel()
		// Given a non-versioned kind, Then it has no publish_state
		if _, ok := generated.ResourceSchema_workspace(ctx).Attributes["publish_state"]; ok {
			t.Fatal("non-versioned resource has publish_state")
		}

		if _, ok := generated.DataSourceSchema_workspace(ctx).Attributes["publish_state"]; ok {
			t.Fatal("non-versioned data source has publish_state")
		}
	})
}

func TestPublishState_whenGeneratedPlanKindConfigured(t *testing.T) {
	for _, typ := range []string{"wayfinder_cloud_resource_plan", "wayfinder_kube_resource_plan"} {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()
			// Given the generated fixture with publish_state configured
			c := planCase(t, typ)
			// When / Then: runCase applies, re-plans empty, imports with publish_state verified
			// (it is not in the ignore list) and re-plans empty after a config-driven import.
			runCase(t, c, func(e caseEnv, cfg string) []resource.TestStep {
				return []resource.TestStep{{
					Config:           cfg,
					ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(e.addr, "publish_state", "Published"),
						e.publishStateStored("Published"),
					),
				}}
			})
		})
	}
}

// planCase returns the generated case for typ with publish_state = "Published" configured.
func planCase(t *testing.T, typ string) generatedCase {
	t.Helper()

	const version = "  version = \"1.0.0\"\n"

	for _, c := range generatedCases {
		if c.Type != typ {
			continue
		}

		if c.Skip != "" || !strings.Contains(c.Config, version) {
			t.Fatalf("%s fixture is skipped or has no version line", typ)
		}

		c.Config = strings.Replace(c.Config, version, version+"  publish_state = \"Published\"\n", 1)

		return c
	}

	t.Fatalf("no generated case for %s", typ)

	return generatedCase{}
}

// publishStateStored proves the API object carries metadata.publishState == want.
func (e caseEnv) publishStateStored(want string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		obj, ok := e.fake.Object(e.ref)
		if !ok {
			return fmt.Errorf("%s not stored at %+v", e.addr, e.ref)
		}

		meta, _ := obj["metadata"].(map[string]any)
		if got := meta["publishState"]; got != want {
			return fmt.Errorf("metadata.publishState = %v, want %q", got, want)
		}

		return nil
	}
}
