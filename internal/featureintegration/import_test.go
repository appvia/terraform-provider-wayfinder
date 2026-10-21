// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appvia/terraform-provider-wayfinder/internal/featureintegration"
)

func TestImport_whenBareAndPrefixedID(t *testing.T) {
	for _, id := range []string{"azure", "wayfinder.azure"} {
		t.Run(id, func(t *testing.T) {
			// Given an enabled feature with managed and unmanaged parameters.
			e := newEnv(t)
			e.fake.Seed(featureRef("azure"), featureObject("azure", true, map[string]any{"outside": "preserved"}))
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: e.factories,
				Steps: []resource.TestStep{
					{Config: featureHCL("azure", `params = { owned = jsonencode(true) }`)},
					{
						// When importing either supported ID form.
						ResourceName: address, ImportState: true, ImportStateId: id,
						ImportStateVerify: true, ImportStateVerifyIgnore: []string{"params"},
						// Then identity is hydrated, but no remote parameter is adopted.
						ImportStateCheck: func(states []*terraform.InstanceState) error {
							for key, value := range states[0].Attributes {
								if strings.HasPrefix(key, "params.") && (key != "params.%" || value != "0") {
									return fmt.Errorf("import adopted %s = %s", key, value)
								}
							}

							return nil
						},
					},
				},
			})
		})
	}
}

func TestImport_whenNotEnabled(t *testing.T) {
	for _, exists := range []bool{false, true} {
		t.Run(fmt.Sprintf("object_exists_%t", exists), func(t *testing.T) {
			// Given either no object or an existing disabled feature.
			e := newEnv(t)
			if exists {
				e.fake.Seed(featureRef("azure"), featureObject("azure", false, nil))
			}

			// When importing, then an actionable not-enabled error is returned.
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: e.factories,
				Steps: []resource.TestStep{{
					Config: featureHCL("azure", ""), ResourceName: address,
					ImportState: true, ImportStateId: "azure",
					ExpectError: regexp.MustCompile(`(?s)Feature integration not enabled.*not enabled.*apply to enable`),
				}},
			})
		})
	}
}

func TestImport_whenIDInvalid(t *testing.T) {
	for _, id := range []string{"", "wayfinder.", "workspace/azure", "/azure", "azure/"} {
		t.Run(fmt.Sprintf("%q", id), func(t *testing.T) {
			// Given a malformed ID and a configured resource.
			e := newEnv(t)
			r := featureintegration.NewResource()

			var configured fwresource.ConfigureResponse
			r.(fwresource.ResourceWithConfigure).Configure(t.Context(), fwresource.ConfigureRequest{ProviderData: e.client}, &configured)

			if configured.Diagnostics.HasError() {
				t.Fatal(configured.Diagnostics)
			}

			var schema fwresource.SchemaResponse
			r.Schema(t.Context(), fwresource.SchemaRequest{}, &schema)

			resp := fwresource.ImportStateResponse{State: tfsdk.State{
				Schema: schema.Schema, Raw: tftypes.NewValue(schema.Schema.Type().TerraformType(t.Context()), nil),
			}}
			// When importing, including the empty ID Terraform's CLI cannot pass through.
			r.(fwresource.ResourceWithImportState).ImportState(t.Context(), fwresource.ImportStateRequest{ID: id}, &resp)
			// Then the ID fails before any API call and no state is added.
			errs := resp.Diagnostics.Errors()
			if len(errs) != 1 || errs[0].Summary() != "Invalid import ID" || !resp.State.Raw.IsNull() {
				t.Fatalf("diagnostics = %v, state = %v", resp.Diagnostics, resp.State.Raw)
			}

			if got := len(e.fake.Requests()); got != 0 {
				t.Errorf("API requests = %d, want 0", got)
			}
		})
	}
}
