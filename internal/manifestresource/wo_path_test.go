// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package manifestresource

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseWOPath_whenValid(t *testing.T) {
	for _, tc := range []struct{ in, canonical string }{
		{"spec.a", "spec.a"},
		{"spec.rules[0].token", "spec.rules[0].token"},
		{`spec.headers["X-Api.Key"]`, `spec.headers["X-Api.Key"]`},
		{`["spec"].a`, "spec.a"},
		{"spec.list[10]", "spec.list[10]"},
		{`spec["X-Api-Key"]`, "spec.X-Api-Key"},
	} {
		t.Run(tc.in, func(t *testing.T) {
			// Given a valid version key. When parsing it.
			p, err := parseWOPath(tc.in)
			// Then it parses and renders its canonical form.
			if err != nil {
				t.Fatal(err)
			}
			if p.String() != tc.canonical {
				t.Fatalf("canonical = %s, want %s", p.String(), tc.canonical)
			}
		})
	}
}

func TestParseWOPath_whenInvalid(t *testing.T) {
	for _, in := range []string{"", ".spec", "spec..a", "spec[", "spec[-1]", "spec[x]", `spec["a]`, "spec]", "[0].a", `spec[""]`, "spec.a.", "spec[0]x"} {
		t.Run(in, func(t *testing.T) {
			// Given an invalid version key. When parsing it.
			_, err := parseWOPath(in)
			// Then the error names the key and the grammar problem.
			if err == nil || !strings.Contains(err.Error(), "invalid manifest_wo_version key") {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestLookup_whenPresentAbsentOrNil(t *testing.T) {
	doc := map[string]any{
		"spec": map[string]any{
			"rules":   []any{map[string]any{"token": "t"}},
			"headers": map[string]any{"X-Api.Key": "k"},
			"empty":   nil,
		},
	}
	for _, tc := range []struct {
		path string
		want any
		ok   bool
	}{
		{"spec.rules[0].token", "t", true},
		{`spec.headers["X-Api.Key"]`, "k", true},
		{"spec.rules[1].token", nil, false},
		{"spec.missing", nil, false},
		{"spec.empty", nil, false},
		{"spec.rules.token", nil, false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			// Given a document. When looking up a path.
			p, err := parseWOPath(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := lookup(doc, p)
			// Then the value and presence match.
			if ok != tc.ok || got != tc.want {
				t.Fatalf("lookup = %v, %v", got, ok)
			}
		})
	}
}

func TestSetAt_whenIntermediateMissing(t *testing.T) {
	// Given a document without spec. When setting a nested key.
	doc := map[string]any{}
	p, err := parseWOPath(`spec.headers["X-Api.Key"]`)
	if err != nil {
		t.Fatal(err)
	}
	setAt(doc, p, "k")
	// Then the missing maps are created.
	want := map[string]any{"spec": map[string]any{"headers": map[string]any{"X-Api.Key": "k"}}}
	if !reflect.DeepEqual(doc, want) {
		t.Fatalf("doc = %v", doc)
	}

	// Given a list index that does not exist. When setting it.
	doc = map[string]any{"spec": map[string]any{"rules": []any{}}}
	p, err = parseWOPath("spec.rules[2].token")
	if err != nil {
		t.Fatal(err)
	}
	setAt(doc, p, "t")
	// Then nothing changes.
	if !reflect.DeepEqual(doc, map[string]any{"spec": map[string]any{"rules": []any{}}}) {
		t.Fatalf("doc = %v", doc)
	}
}

func TestSetAt_whenPathCannotBeSet(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  map[string]any
		path string
		want map[string]any // nil means the document must stay unchanged
	}{
		{"missing ancestor before index", map[string]any{}, "spec.rules[0].token", nil},
		{"missing list", map[string]any{"spec": map[string]any{}}, "spec.rules[0].token", nil},
		{"null list", map[string]any{"spec": map[string]any{"rules": nil}}, "spec.rules[0].token", nil},
		{"out of range index", map[string]any{"spec": map[string]any{"rules": []any{}}}, "spec.rules[0].token", nil},
		{"non-list at index", map[string]any{"spec": map[string]any{"rules": map[string]any{}}}, "spec.rules[0].token", nil},
		{"existing list item", map[string]any{"spec": map[string]any{"rules": []any{map[string]any{"token": "old"}}}}, "spec.rules[0].token",
			map[string]any{"spec": map[string]any{"rules": []any{map[string]any{"token": "new"}}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given a document. When setting a path that may or may not resolve.
			p, err := parseWOPath(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			want := tc.want
			if want == nil {
				want = deepCopy(tc.doc).(map[string]any)
			}
			setAt(tc.doc, p, "new")
			// Then the document is either updated or left exactly unchanged.
			if !reflect.DeepEqual(tc.doc, want) {
				t.Fatalf("doc = %v, want %v", tc.doc, want)
			}
		})
	}
}

func TestDeepCopy_whenMutated(t *testing.T) {
	// Given a nested source document.
	src := map[string]any{"spec": map[string]any{"rules": []any{map[string]any{"token": "t"}}}}
	// When the copy is mutated.
	cp := deepCopy(src).(map[string]any)
	cp["spec"].(map[string]any)["rules"].([]any)[0].(map[string]any)["token"] = "changed"
	cp["spec"].(map[string]any)["added"] = true
	// Then the source is unchanged.
	want := map[string]any{"spec": map[string]any{"rules": []any{map[string]any{"token": "t"}}}}
	if !reflect.DeepEqual(src, want) {
		t.Fatalf("src = %v", src)
	}
}
