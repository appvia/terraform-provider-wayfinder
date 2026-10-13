package fakekinds

import (
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
