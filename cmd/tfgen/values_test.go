package main

import (
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

func ptr[T any](v T) *T { return &v }

func TestSynth_whenFixtureHasOptionalNestedSecrets(t *testing.T) {
	// Given a spec with a required field and two optional write-only fields in optional objects.
	spec := &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
		"kind":  {Type: "string", Required: true, Enum: []string{"A", "B"}},
		"basic": {Type: "object", Properties: map[string]*manifest.Field{"password": {Type: "string", Sensitive: true}, "user": {Type: "string"}}},
		"api":   {Type: "object", Properties: map[string]*manifest.Field{"headers": {Type: "map", Sensitive: true, Values: &manifest.Field{Type: "string"}}}},
	}}
	s := &synth{typ: "wayfinder_thing", fixture: true}
	// When rendering the fixture body.
	got := s.body(spec, true, "  ", "")
	// Then every write-only field gets a unique literal and a version, optional siblings stay out.
	for _, want := range []string{`kind = "A"`, `password_wo = "wo-thing-basic-password-wo"`, `password_wo_version = "1"`, `headers_wo = { example = "wo-thing-api-headers-wo" }`} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in\n%s", want, got)
		}
	}
	if strings.Contains(got, "user") || len(s.problems) != 0 {
		t.Fatalf("unexpected optional field or problems %v in\n%s", s.problems, got)
	}
	if strings.Join(s.writeOnly, ",") != "api.headers_wo,basic.password_wo" {
		t.Fatalf("writeOnly = %v", s.writeOnly)
	}
}

func TestSynth_whenExampleOmitsOptionalSecrets(t *testing.T) {
	// Given the same optional secret, When rendering a documentation example.
	spec := &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
		"basic": {Type: "object", Properties: map[string]*manifest.Field{"password": {Type: "string", Sensitive: true}}},
	}}
	s := &synth{typ: "wayfinder_thing"}
	// Then nothing optional is emitted.
	if got := s.body(spec, true, "  ", ""); got != "" {
		t.Fatalf("example body = %q", got)
	}
}

func TestSynth_whenConstrainedStrings(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field manifest.Field
		want  string
	}{
		{"pattern file name", manifest.Field{Type: "string", Pattern: `^[a-z][a-z0-9_-]*\.tf$`}, "example.tf"},
		{"date-time format", manifest.Field{Type: "string", Format: "date-time"}, "2030-01-01T00:00:00Z"},
		{"max length", manifest.Field{Type: "string", MaxLength: ptr(1)}, "a"},
		{"minimum", manifest.Field{Type: "integer", Minimum: ptr(5.0)}, "5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given a required constrained field, When synthesizing its value.
			s := &synth{}
			got := s.value(&tc.field, "", "f")
			// Then the value satisfies the generated validators.
			if strings.Trim(got, `"`) != tc.want || len(s.problems) != 0 {
				t.Fatalf("value = %s problems = %v, want %s", got, s.problems, tc.want)
			}
		})
	}
}

func TestRenderTests_whenNoLiteralSatisfiesPattern(t *testing.T) {
	// Given a typed kind whose required field pattern no candidate literal matches.
	k := manifest.Kind{Kind: "Odd", Group: "core", Version: "v3", APIPath: "odds", Typed: true, Scopes: []string{"workspace"}, Spec: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
		"code": {Type: "string", Required: true, Pattern: `^[0-9]{6}$`},
	}}}
	// When rendering the plan-stability cases.
	got, err := renderTests([]manifest.Kind{k})
	if err != nil {
		t.Fatal(err)
	}
	// Then the case is emitted with an explicit skip reason naming the field.
	if !strings.Contains(string(got), `Skip:`) || !strings.Contains(string(got), "code: no synthesized literal satisfies pattern") || !strings.Contains(string(got), `ImportID: "example/example"`) {
		t.Fatalf("generated cases:\n%s", got)
	}
}
