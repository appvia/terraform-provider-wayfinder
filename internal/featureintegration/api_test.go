// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/featureintegration"
)

func TestAPIError_whenValidationFieldHasAttributePath(t *testing.T) {
	for _, tc := range []struct {
		field string
		path  path.Path
	}{
		{"name", path.Root("name")},
		{"spec.params", path.Root("params")},
		{"spec.params.token", path.Root("params").AtMapKey("token")},
	} {
		t.Run(tc.field, func(t *testing.T) {
			// Given a server validation field.
			err := &client.ValidationError{Fields: []client.FieldError{{Field: tc.field, Message: "rejected"}}}

			var diags diag.Diagnostics
			// When mapping it onto the resource.
			featureintegration.AddAPIError(&diags, err, "azure")
			// Then the error points at the matching Terraform attribute.
			errs := diags.Errors()
			if len(errs) != 1 {
				t.Fatalf("diagnostics = %v", diags)
			}

			got, ok := errs[0].(diag.DiagnosticWithPath)
			if !ok || !got.Path().Equal(tc.path) {
				t.Fatalf("diagnostic = %v, want path %s", errs[0], tc.path)
			}
		})
	}
}

func TestAPIError_whenNotFoundMessageLacksTrailingPeriod(t *testing.T) {
	// Given a not-found error whose message has no trailing full stop.
	err := fmt.Errorf("%w: feature integration does not exist", client.ErrNotFound)

	var diags diag.Diagnostics
	// When mapping it onto the resource.
	featureintegration.AddAPIError(&diags, err, "azure")
	// Then the detail adds a period before the guidance sentence.
	errs := diags.Errors()
	if len(errs) != 1 {
		t.Fatalf("diagnostics = %v", diags)
	}

	msg := "not found: feature integration does not exist"
	if got := errs[0].Detail(); !strings.Contains(got, msg+".") {
		t.Fatalf("detail = %q, want it to contain %q", got, msg+".")
	}
}

func TestRead_whenOnlyConfiguredParamsAreManaged(t *testing.T) {
	// Given changed and missing managed parameters alongside an unmanaged key.
	prior, diags := types.MapValue(jsontypes.NormalizedType{}, map[string]attr.Value{
		"changed": jsontypes.NewNormalizedValue("true"),
		"missing": jsontypes.NewNormalizedValue("1"),
	})
	if diags.HasError() {
		t.Fatal(diags)
	}

	obj := featureObject("azure", true, map[string]any{"changed": false, "outside": "preserved"})
	// When refreshing the managed parameter projection.
	got, diags := featureintegration.ProjectParams(prior, obj)
	// Then changed values are refreshed, missing keys disappear, and outside keys stay unmanaged.
	if diags.HasError() {
		t.Fatal(diags)
	}

	if values := got.Elements(); len(values) != 1 || !values["changed"].Equal(jsontypes.NewNormalizedValue("false")) {
		t.Fatalf("projected params = %v", got)
	}
}

func TestRead_whenParamsNullOrEmpty(t *testing.T) {
	for _, prior := range []types.Map{
		types.MapNull(jsontypes.NormalizedType{}),
		types.MapValueMust(jsontypes.NormalizedType{}, map[string]attr.Value{}),
		types.MapValueMust(jsontypes.NormalizedType{}, map[string]attr.Value{"missing": jsontypes.NewNormalizedValue("1")}),
	} {
		// Given either null, empty, or missing managed parameters.
		obj := featureObject("azure", true, map[string]any{"outside": "preserved"})
		// When refreshing.
		got, diags := featureintegration.ProjectParams(prior, obj)
		// Then null stays null, and a non-null map stays non-null without adopting keys.
		if diags.HasError() || got.IsNull() != prior.IsNull() || len(got.Elements()) != 0 {
			t.Fatalf("prior = %v, projected = %v, diagnostics = %v", prior, got, diags)
		}
	}
}
