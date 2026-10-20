// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package convert

import (
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

func TestAttributePath_whenNestedValidationField(t *testing.T) {
	// Given an API path with a list index and a reserved-name collision.
	m, err := NewMapping(manifest.Kind{Spec: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
		"name": {Type: "string"}, "rules": {Type: "array", Items: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{"apiKey": {Type: "string", Sensitive: true}}}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		json string
		want path.Path
	}{
		{"spec.name", path.Root("spec_name")},
		{"spec.rules[0].apiKey", path.Root("rules").AtListIndex(0).AtName("api_key_wo")},
		{"metadata.labels.team", path.Root("labels").AtMapKey("team")},
	} {
		// When mapping a server validation field back to Terraform.
		got, ok := m.AttributePath(tc.json)
		// Then the diagnostic is attached to the actual configured attribute.
		if !ok || !got.Equal(tc.want) {
			t.Fatalf("%s = %s (%t), want %s", tc.json, got, ok, tc.want)
		}
	}
}

func TestAttributePath_whenIdentityAndMetadataFields(t *testing.T) {
	spec := &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{"mode": {Type: "string"}}}
	for _, tc := range []struct {
		name   string
		kind   manifest.Kind
		field  string
		want   path.Path
		wantOK bool
	}{
		{"tenant bare name", manifest.Kind{Scopes: []string{"tenant"}, Spec: spec}, "name", path.Root("name"), true},
		{"tenant metadata name", manifest.Kind{Scopes: []string{"tenant"}, Spec: spec}, "metadata.name", path.Root("name"), true},
		{"tenant has no workspace", manifest.Kind{Scopes: []string{"tenant"}, Spec: spec}, "workspace", path.Empty(), false},
		{"workspace bare workspace", manifest.Kind{Scopes: []string{"workspace"}, Spec: spec}, "workspace", path.Root("workspace"), true},
		{"workspace metadata workspace", manifest.Kind{Scopes: []string{"workspace"}, Spec: spec}, "metadata.workspace", path.Root("workspace"), true},
		{"environment bare", manifest.Kind{Scopes: []string{"workspace", "environment"}, Spec: spec}, "environment", path.Root("environment"), true},
		{"workspace has no environment", manifest.Kind{Scopes: []string{"workspace"}, Spec: spec}, "metadata.environment", path.Empty(), false},
		{"versioned bare version", manifest.Kind{Scopes: []string{"tenant"}, Versioned: true, Spec: spec}, "version", path.Root("version"), true},
		{"versioned metadata version", manifest.Kind{Scopes: []string{"tenant"}, Versioned: true, Spec: spec}, "metadata.version", path.Root("version"), true},
		{"unversioned has no version", manifest.Kind{Scopes: []string{"tenant"}, Spec: spec}, "version", path.Empty(), false},
		{"labels map", manifest.Kind{Scopes: []string{"tenant"}, Spec: spec}, "metadata.labels", path.Root("labels"), true},
		{"label key with dots and slash", manifest.Kind{Scopes: []string{"tenant"}, Spec: spec}, "metadata.labels.app.kubernetes.io/name", path.Root("labels").AtMapKey("app.kubernetes.io/name"), true},
		{"annotation quoted key", manifest.Kind{Scopes: []string{"tenant"}, Spec: spec}, `metadata.annotations["a.b/c"]`, path.Root("annotations").AtMapKey("a.b/c"), true},
		{"unknown bare path", manifest.Kind{Scopes: []string{"tenant"}, Spec: spec}, "bogus", path.Empty(), false},
		{"unknown metadata path", manifest.Kind{Scopes: []string{"tenant"}, Spec: spec}, "metadata.bogus", path.Empty(), false},
		{"name with subpath", manifest.Kind{Scopes: []string{"tenant"}, Spec: spec}, "name.x", path.Empty(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given a mapping for the kind.
			m, err := NewMapping(tc.kind)
			if err != nil {
				t.Fatal(err)
			}
			// When mapping the server field.
			got, ok := m.AttributePath(tc.field)
			// Then it resolves to the attribute, or is unmappable.
			if ok != tc.wantOK || (ok && !got.Equal(tc.want)) {
				t.Fatalf("%s = %s (%t), want %s (%t)", tc.field, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}
