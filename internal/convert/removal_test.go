package convert_test

import (
	"reflect"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestToObject_whenManagedValuesRemoved(t *testing.T) {
	// Given: optional, computed and secret fields at each collection boundary.
	str := &manifest.Field{Type: "string"}
	obj := &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
		"keep": str, "remove": str, "computed": {Type: "string", Computed: true}, "token": {Type: "string", Sensitive: true},
	}}
	m, err := convert.NewMapping(manifest.Kind{Kind: "Example", Group: "example.io", Version: "v1", Scopes: []string{"tenant"}, Spec: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
		"description": str, "nested": obj, "entries": {Type: "map", Values: obj}, "rules": {Type: "array", Items: obj},
		"computed": {Type: "string", Computed: true}, "required": {Type: "string", Required: true}, "token": {Type: "string", Sensitive: true},
		"headers": {Type: "map", Sensitive: true, Values: str},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, prior, plan, base, want string
		unset, nullPrior              bool
	}{
		{name: "top level", prior: `{"description":"first"}`, plan: `{}`, base: `{"spec":{"description":"first","unmanaged":true}}`, want: `{"unmanaged":true}`},
		{name: "nested field", prior: `{"nested":{"keep":"yes","remove":"old"}}`, plan: `{"nested":{"keep":"yes"}}`, base: `{"spec":{"nested":{"keep":"yes","remove":"old","server":"added"}}}`, want: `{"nested":{"keep":"yes","server":"added"}}`},
		{name: "map entry and nested field", prior: `{"entries":{"a":{"keep":"yes","remove":"old"},"b":{"keep":"old"}}}`, plan: `{"entries":{"a":{"keep":"yes"}}}`, base: `{"spec":{"entries":{"a":{"keep":"yes","remove":"old"},"b":{"keep":"old"},"server":{"keep":"added"}}}}`, want: `{"entries":{"a":{"keep":"yes"},"server":{"keep":"added"}}}`},
		{name: "null map entry", prior: `{"entries":{"a":{"keep":"old"}}}`, plan: `{"entries":{"a":null}}`, base: `{"spec":{"entries":{"a":{"keep":"old"}}}}`, want: `{"entries":{}}`},
		{name: "newly configured map preserves server entry", prior: `{}`, plan: `{"entries":{"a":{"keep":"yes"}}}`, base: `{"spec":{"entries":{"server":{"keep":"added"}}}}`, want: `{"entries":{"a":{"keep":"yes"},"server":{"keep":"added"}}}`},
		{name: "whole optional object and map", prior: `{"nested":{"keep":"yes"},"entries":{"a":{"keep":"yes"}}}`, plan: `{}`, base: `{"spec":{"nested":{"keep":"yes"},"entries":{"a":{"keep":"yes"}}}}`, want: `{}`},
		{name: "computed required and secret", prior: `{"computed":"old","required":"old","token_wo":"secret"}`, plan: `{}`, base: `{"spec":{"computed":"server","required":"old","token":""}}`, want: `{"computed":"server","required":"old","token":""}`},
		{name: "list replaces elements", prior: `{"rules":[{"keep":"yes","remove":"old"},{"keep":"gone"}]}`, plan: `{"rules":[{"keep":"yes"}]}`, base: `{"spec":{"rules":[{"keep":"yes","remove":"old","server":"added","computed":"server","token":""},{"keep":"gone"}]}}`, want: `{"rules":[{"keep":"yes","computed":"server","token":""}]}`},
		{name: "prior unset", unset: true, prior: `{}`, plan: `{}`, base: `{"spec":{"description":"first"}}`, want: `{"description":"first"}`},
		{name: "prior null", nullPrior: true, prior: `{}`, plan: `{}`, base: `{"spec":{"description":"first"}}`, want: `{"description":"first"}`},
		{name: "unknown plan", prior: `{"description":"first"}`, plan: `{}`, base: `{"spec":{"description":"first"}}`, want: `{"description":"first"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prior, err := fixtureValue(m.TFType(), tc.prior)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := fixtureValue(m.TFType(), tc.plan)
			if err != nil {
				t.Fatal(err)
			}
			if tc.unset {
				prior = tftypes.Value{}
			}
			if tc.nullPrior {
				prior = tftypes.NewValue(m.TFType(), nil)
			}
			if tc.name == "unknown plan" {
				attrs := values(t, plan)
				attrs["description"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
				plan = tftypes.NewValue(m.TFType(), attrs)
			}
			// When
			got, err := convert.ToObject(m, plan, convert.ToOpts{Base: object(t, tc.base), Prior: prior})
			// Then
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got["spec"], object(t, tc.want)) {
				t.Fatalf("spec = %#v; want %s", got["spec"], tc.want)
			}
		})
	}
}

func TestToObject_whenWriteOnlyMapRotatedWithPrior(t *testing.T) {
	// Given
	m := mapping(t, "Integration")
	prior := config(t, m, map[string]any{"headers_wo_version": "1"})
	plan := config(t, m, map[string]any{"headers_wo_version": "2", "headers_wo": map[string]tftypes.Value{"new": tftypes.NewValue(tftypes.String, "secret")}})
	// When
	got, err := convert.ToObject(m, plan, convert.ToOpts{Base: object(t, `{"spec":{"headers":{"old":""}}}`), Prior: prior, IncludeWriteOnly: map[string]bool{"headers_wo": true}})
	// Then
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got["spec"].(map[string]any)["headers"], map[string]any{"new": "secret"}) {
		t.Fatalf("rotated headers = %#v", got["spec"])
	}
}

func TestToObject_whenManagedMetadataKeysRemoved(t *testing.T) {
	for _, name := range []string{"labels", "annotations"} {
		for _, plan := range []string{`{"a":"1"}`, `null`} {
			t.Run(name+plan, func(t *testing.T) {
				// Given
				m := mapping(t, "Workspace")
				prior, err := fixtureValue(m.TFType(), `{"`+name+`":{"a":"1","b":"2"}}`)
				if err != nil {
					t.Fatal(err)
				}
				cfg, err := fixtureValue(m.TFType(), `{"`+name+`":`+plan+`}`)
				if err != nil {
					t.Fatal(err)
				}
				base := object(t, `{"metadata":{"generation":7,"uid":"id","`+name+`":{"a":"1","b":"2","server":"added"}}}`)
				// When
				got, err := convert.ToObject(m, cfg, convert.ToOpts{Base: base, Prior: prior})
				// Then
				if err != nil {
					t.Fatal(err)
				}
				want := map[string]any{"server": "added"}
				if plan != "null" {
					want["a"] = "1"
				}
				meta := got["metadata"].(map[string]any)
				if !reflect.DeepEqual(meta[name], want) || meta["generation"] != float64(7) || meta["uid"] != "id" {
					t.Fatalf("metadata = %#v", meta)
				}
				if !reflect.DeepEqual(base["metadata"].(map[string]any)[name], map[string]any{"a": "1", "b": "2", "server": "added"}) {
					t.Fatal("mutated Base")
				}
			})
		}
	}
}
