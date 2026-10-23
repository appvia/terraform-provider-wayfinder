// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func goldenKind(t *testing.T, name string) manifest.Kind {
	t.Helper()

	m, err := manifest.Load("../manifest/testdata/golden-manifest.json")
	if err != nil {
		t.Fatal(err)
	}

	for _, k := range m.Kinds {
		if k.Kind == name {
			// The historical fixture predates the relative apiPath contract.
			k.APIPath = strings.TrimPrefix(k.APIPath, "/api/v3/")
			return k
		}
	}

	t.Fatalf("kind %s missing", name)

	return manifest.Kind{}
}

func testSchema(k manifest.Kind) schema.Schema {
	a := fieldAttributes(k.Spec, true)
	a["id"] = schema.StringAttribute{Computed: true}
	a["name"] = schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}

	for _, name := range []string{"workspace", "environment", "version"} {
		present, required := false, false

		switch name {
		case "workspace":
			present = slices.Contains(k.Scopes, "workspace") || slices.Contains(k.Scopes, "environment")
			required = present && !slices.Contains(k.Scopes, "tenant")
		case "environment":
			present = slices.Contains(k.Scopes, "environment")
			required = present && len(k.Scopes) == 1
		case "version":
			present, required = k.Versioned, k.Versioned
		}

		if present {
			attribute := schema.StringAttribute{Required: required, Optional: !required, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}
			if name == "version" {
				attribute.Validators = []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`), "must be strict semver without a v prefix")}
			}

			a[name] = attribute
		}
	}

	if k.Versioned {
		a["publish_state"] = schema.StringAttribute{
			Optional: true, Computed: true,
			MarkdownDescription: "Publish state of this version: `Draft`, `Published` or `Deprecated`. Only published versions are offered in the catalogue. Leave unset to keep the state Wayfinder holds, for example one set with `wf publish`.",
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Validators:          []validator.String{stringvalidator.OneOf("Draft", "Published", "Deprecated")},
		}
	}

	for _, name := range []string{"labels", "annotations"} {
		a[name] = schema.MapAttribute{Optional: true, ElementType: types.StringType}
	}

	a["status"] = schema.SingleNestedAttribute{Computed: true, Attributes: map[string]schema.Attribute{
		"status": schema.StringAttribute{Computed: true}, "message": schema.StringAttribute{Computed: true}, "detail": schema.StringAttribute{Computed: true},
	}}
	a["status_json"] = schema.StringAttribute{Computed: true, CustomType: jsontypes.NormalizedType{}}

	a["timeouts"] = timeouts.Attributes(context.Background(), timeouts.Opts{Create: true, Update: true, Delete: true})
	if k.HasStatus {
		a["wait_for_ready"] = schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)}
	}

	return schema.Schema{Attributes: a}
}

func fieldAttributes(f *manifest.Field, top bool) map[string]schema.Attribute {
	a := map[string]schema.Attribute{}
	if f == nil {
		return a
	}

	for jsonName, child := range f.Properties {
		name := convert.SnakeCase(jsonName)
		if top {
			name = convert.AttrName(jsonName)
		}

		if child.Type == "json" {
			name += "_json"
		}

		if child.Sensitive {
			name += "_wo"
			a[name+"_version"] = schema.StringAttribute{Optional: true, Description: "Change this value to send a new " + name + " to Wayfinder."}
		}

		a[name] = fieldAttribute(child)
	}

	return a
}

func fieldAttribute(f *manifest.Field) schema.Attribute {
	required, optional, computed := f.Required, !f.Required, f.Computed
	switch f.Type {
	case "string", "json":
		a := schema.StringAttribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, WriteOnly: f.Sensitive}
		if f.Type == "json" {
			a.CustomType = jsontypes.NormalizedType{}
		}

		if len(f.Enum) > 0 {
			a.Validators = append(a.Validators, stringvalidator.OneOf(f.Enum...))
		}

		if f.Pattern != "" {
			a.Validators = append(a.Validators, stringvalidator.RegexMatches(regexp.MustCompile(f.Pattern), "must match the API field pattern"))
		}

		if f.MinLength != nil {
			a.Validators = append(a.Validators, stringvalidator.LengthAtLeast(*f.MinLength))
		}

		if f.MaxLength != nil {
			a.Validators = append(a.Validators, stringvalidator.LengthAtMost(*f.MaxLength))
		}

		if !f.Sensitive && f.Immutable {
			a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.RequiresReplace())
		}

		if !f.Sensitive && f.Computed {
			a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.UseStateForUnknown())
		}

		return a
	case "boolean":
		return schema.BoolAttribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, WriteOnly: f.Sensitive}
	case "integer":
		return schema.Int64Attribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, WriteOnly: f.Sensitive}
	case "number":
		return schema.Float64Attribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, WriteOnly: f.Sensitive}
	case "object":
		return schema.SingleNestedAttribute{Required: required, Optional: optional, Computed: computed, Attributes: fieldAttributes(f, false)}
	case "array":
		if f.Items.Type == "object" {
			return schema.ListNestedAttribute{Required: required, Optional: optional, Computed: computed, NestedObject: schema.NestedAttributeObject{Attributes: fieldAttributes(f.Items, false)}}
		}

		return schema.ListAttribute{Required: required, Optional: optional, Computed: computed, ElementType: fieldAttribute(f.Items).GetType()}
	case "map":
		if f.Values.Type == "object" {
			return schema.MapNestedAttribute{Required: required, Optional: optional, Computed: computed, NestedObject: schema.NestedAttributeObject{Attributes: fieldAttributes(f.Values, false)}}
		}

		return schema.MapAttribute{Required: required, Optional: optional, Computed: computed, Sensitive: f.Sensitive, WriteOnly: f.Sensitive, ElementType: fieldAttribute(f.Values).GetType()}
	default:
		panic("unsupported test schema field: " + f.Type)
	}
}
