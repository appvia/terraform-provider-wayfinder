// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func testDataSchema(k manifest.Kind) dsschema.Schema {
	a := map[string]dsschema.Attribute{
		"id":          dsschema.StringAttribute{Computed: true},
		"name":        dsschema.StringAttribute{Required: true},
		"labels":      dsschema.MapAttribute{Computed: true, ElementType: types.StringType},
		"annotations": dsschema.MapAttribute{Computed: true, ElementType: types.StringType},
		"status_json": dsschema.StringAttribute{Computed: true},
	}

	for _, name := range []string{"workspace", "environment", "version"} {
		if field, ok := testSchema(k).Attributes[name]; ok {
			a[name] = dsschema.StringAttribute{Required: field.IsRequired(), Optional: field.IsOptional()}
		}
	}

	if k.Versioned {
		a["publish_state"] = dsschema.StringAttribute{Computed: true, MarkdownDescription: "Publish state of this version: `Draft`, `Published` or `Deprecated`; null when none is set."}
	}

	switch k.Kind {
	case "Workspace":
		a["display_name"] = dsschema.StringAttribute{Computed: true}
	case "Integration":
		a["endpoint"] = dsschema.StringAttribute{Computed: true}
	case "Blueprint":
		a["source"] = dsschema.ObjectAttribute{Computed: true, AttributeTypes: map[string]attr.Type{"repository": types.StringType, "ref": types.StringType}}
	}

	return dsschema.Schema{Attributes: a}
}

func TestDataSource_whenGoldenKindRead(t *testing.T) {
	for _, name := range []string{"Workspace", "Integration", "Blueprint"} {
		t.Run(name, func(t *testing.T) {
			// Given a seeded object and a data source schema without resource-only inputs.
			e := newTestEnv(t, name)

			workspace, version, scope := "", "", ""
			if name != "Workspace" {
				workspace = "ws"
				scope = "workspace = \"ws\"\n"
			}

			if name == "Blueprint" {
				version = "1.2.3"
				scope += "version = \"1.2.3\"\n"
			}

			e.fake.Seed(e.ref(workspace, "", version), map[string]any{"metadata": map[string]any{"labels": map[string]any{"server": "yes"}}, "spec": map[string]any{"displayName": "Server value", "endpoint": "https://server", "source": map[string]any{"repository": "https://server/repo"}}})

			field, want := "display_name", "Server value"
			if name == "Integration" {
				field, want = "endpoint", "https://server"
			}

			if name == "Blueprint" {
				field, want = "source.repository", "https://server/repo"
			}
			// When Terraform reads the single object.
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{{
				Config:           "data \"wayfinder_" + e.k.TerraformName + "\" \"example\" {\n name=\"example\"\n" + scope + "}\n",
				ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				Check:            resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttr("data.wayfinder_"+e.k.TerraformName+".example", "labels.server", "yes"), resource.TestCheckResourceAttr("data.wayfinder_"+e.k.TerraformName+".example", field, want), checkSecretAbsent(t)),
			}}})
			// Then versioned GETs use their version route and computed values are populated.
			for _, req := range e.fake.Requests() {
				if name == "Blueprint" && !strings.HasSuffix(req.Path, "/versions/1.2.3") {
					t.Fatalf("unversioned data source request: %s", req.Path)
				}
			}
		})
	}
}

func TestConfigure_whenNilOrWrongProviderData(t *testing.T) {
	// Given factories whose provider configuration may not yet be available.
	e := newTestEnv(t, "Workspace")
	r := NewResource(e.k, testSchema(e.k)).(fwresource.ResourceWithConfigure)
	s := NewDataSource(e.k, testDataSchema(e.k)).(datasource.DataSourceWithConfigure)
	// When configuration is absent, then supplied with the wrong type.
	for _, data := range []any{nil, "wrong"} {
		rr := fwresource.ConfigureResponse{}
		sr := datasource.ConfigureResponse{}

		r.Configure(t.Context(), fwresource.ConfigureRequest{ProviderData: data}, &rr)
		s.Configure(t.Context(), datasource.ConfigureRequest{ProviderData: data}, &sr)
		// Then only the wrong type is diagnosed.
		if rr.Diagnostics.HasError() != (data != nil) || sr.Diagnostics.HasError() != (data != nil) {
			t.Fatal("provider data contract violated")
		}
	}
}

func TestDataSource_whenMissingObjectReturnsError(t *testing.T) {
	// Given a configured data source with no corresponding server object.
	e := newTestEnv(t, "Workspace")
	s := NewDataSource(e.k, testDataSchema(e.k)).(*kindDataSource)
	s.client = e.c
	typ := s.schema.Type().TerraformType(context.Background()).(tftypes.Object)

	attrs := map[string]tftypes.Value{}
	for name, child := range typ.AttributeTypes {
		attrs[name] = tftypes.NewValue(child, nil)
	}

	attrs["name"] = tftypes.NewValue(tftypes.String, "missing")
	resp := datasource.ReadResponse{}
	// When reading.
	s.Read(t.Context(), datasource.ReadRequest{Config: tfsdk.Config{Raw: tftypes.NewValue(typ, attrs), Schema: s.schema}}, &resp)
	// Then data source lookup fails rather than removing a resource from state.
	if !resp.Diagnostics.HasError() {
		t.Fatal("missing data source succeeded")
	}
}

func readMissingByName(t *testing.T, e testEnv, name string) datasource.ReadResponse {
	t.Helper()

	s := NewDataSource(e.k, testDataSchema(e.k)).(*kindDataSource)
	s.client = e.c
	typ := s.schema.Type().TerraformType(context.Background()).(tftypes.Object)

	attrs := map[string]tftypes.Value{}
	for attr, child := range typ.AttributeTypes {
		attrs[attr] = tftypes.NewValue(child, nil)
	}

	attrs["name"] = tftypes.NewValue(tftypes.String, name)
	resp := datasource.ReadResponse{}
	s.Read(t.Context(), datasource.ReadRequest{Config: tfsdk.Config{Raw: tftypes.NewValue(typ, attrs), Schema: s.schema}}, &resp)

	return resp
}

func TestDataSource_whenBareFeatureNameMissing(t *testing.T) {
	// Given an Integration data source and no object for a bare feature name.
	e := newFeatureEnv(t)
	// When reading it.
	resp := readMissingByName(t, e, "azure")
	// Then the error points at the wayfinder.-prefixed name and the feature resource.
	got := resp.Diagnostics.Errors()
	if len(got) != 1 || got[0].Summary() != "Integration not found" ||
		!strings.Contains(got[0].Detail(), `"wayfinder.azure"`) || !strings.Contains(got[0].Detail(), "wayfinder_feature_integration") {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
}

func TestDataSource_whenPrefixedNameMissing(t *testing.T) {
	// Given an Integration data source and no object for a prefixed name.
	e := newFeatureEnv(t)
	// When reading it.
	resp := readMissingByName(t, e, "wayfinder.azure")
	// Then it fails without the bare-name hint.
	if !resp.Diagnostics.HasError() || strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "wayfinder_feature_integration") {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
}
