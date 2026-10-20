// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakekinds

import (
	"fmt"
	"slices"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

func TestSpec_whenKindHasNestedSensitiveFields(t *testing.T) {
	// Given a kind with sensitive string leaves (top level, nested, inside an array) and a map.
	secret := func() *manifest.Field { return &manifest.Field{Type: "string", Sensitive: true} }
	k := manifest.Kind{Kind: "Thing", Group: "core", Version: "v3", APIPath: "things", Scopes: []string{"tenant", "workspace"}, Versioned: true, HasStatus: true,
		Spec: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
			"apiKey":  secret(),
			"headers": {Type: "map", Sensitive: true, Values: &manifest.Field{Type: "string"}},
			"bearer":  {Type: "object", Properties: map[string]*manifest.Field{"token": secret(), "user": {Type: "string"}}},
			"rules":   {Type: "array", Items: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{"password": secret()}}},
		}}}
	// When deriving its fake registration.
	s := Spec(k)
	// Then identity, scopes and every sensitive path follow the fakeapi conventions.
	if s.Kind != "Thing" || s.APIPath != "things" || !s.Versioned || !s.HasStatus || !slices.Equal(s.Scopes, []fakeapi.Scope{fakeapi.ScopeTenant, fakeapi.ScopeWorkspace}) {
		t.Fatalf("spec = %+v", s)
	}
	if want := []string{"spec.apiKey", "spec.bearer.token", "spec.rules.password"}; !slices.Equal(s.SensitiveStrings, want) {
		t.Fatalf("SensitiveStrings = %v, want %v", s.SensitiveStrings, want)
	}
	if want := []string{"spec.headers"}; !slices.Equal(s.SensitiveMaps, want) {
		t.Fatalf("SensitiveMaps = %v, want %v", s.SensitiveMaps, want)
	}
}

func TestServerDefaults_whenComputedFieldsAtVariousDepths(t *testing.T) {
	// Given computed fields at the top level, under a required object, under an optional object
	// and inside an array, plus a computed enum and a computed list.
	computed := func(typ string) *manifest.Field { return &manifest.Field{Type: typ, Computed: true} }
	k := manifest.Kind{Kind: "Thing", Spec: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
		"slug":     computed("string"),
		"enabled":  computed("boolean"),
		"tier":     {Type: "string", Computed: true, Enum: []string{"gold", "silver"}},
		"audience": {Type: "array", Computed: true, Items: &manifest.Field{Type: "string"}},
		"subject":  {Type: "object", Required: true, Properties: map[string]*manifest.Field{"kind": computed("string")}},
		"azure":    {Type: "object", Properties: map[string]*manifest.Field{"tenantID": computed("string")}},
		"rules":    {Type: "array", Items: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{"id": computed("string")}}},
	}}}
	// When deriving the server defaults.
	got := ServerDefaults(k)
	// Then only fields whose parents are always present get a typed default.
	want := map[string]any{
		"spec.slug": "server-default", "spec.enabled": true, "spec.tier": "gold",
		"spec.audience": []any{"server-default"}, "spec.subject.kind": "server-default",
	}
	if len(got) != len(want) {
		t.Fatalf("defaults = %v, want %v", got, want)
	}
	for path, v := range want {
		if fmt.Sprint(got[path]) != fmt.Sprint(v) {
			t.Fatalf("%s = %v, want %v", path, got[path], v)
		}
	}
}

func TestServerDefaults_whenComputedListOfObjects(t *testing.T) {
	// Given a computed list of objects with optional, non-computed children, like
	// ExternalIdentity spec.usableBy that the server fills with the creator.
	k := manifest.Kind{Kind: "Thing", Spec: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
		"usableBy": {Type: "array", Computed: true, Items: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
			"canGrant": {Type: "boolean"},
			"subject":  {Type: "object", Properties: map[string]*manifest.Field{"kind": {Type: "string"}}},
			"token":    {Type: "string", Sensitive: true},
		}}},
	}}}
	// When deriving the server defaults.
	got := ServerDefaults(k)
	// Then the list holds one element with every non-secret child filled in.
	want := []any{map[string]any{"canGrant": true, "subject": map[string]any{"kind": "server-default"}}}
	if fmt.Sprint(got["spec.usableBy"]) != fmt.Sprint(want) {
		t.Fatalf("spec.usableBy = %v, want %v", got["spec.usableBy"], want)
	}
}
