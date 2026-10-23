// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

func diffFixture(t *testing.T, newName string) diffReport {
	t.Helper()

	r, err := diffFiles(filepath.Join("testdata", "diff", "old.json"), filepath.Join("testdata", "diff", newName))
	if err != nil {
		t.Fatal(err)
	}

	return r
}

func firstLine(r diffReport) string {
	var b bytes.Buffer
	r.write(&b)
	line, _, _ := strings.Cut(b.String(), "\n")

	return line
}

func TestDiff_whenAttributeRemoved_isBreaking(t *testing.T) {
	// Given/When
	r := diffFixture(t, "removed-attribute.json")
	// Then
	want := []attrChange{{Resource: "wayfinder_workspace", Attribute: "summary"}}
	if !slices.Equal(r.Removed, want) {
		t.Fatalf("removed = %v, want %v", r.Removed, want)
	}

	if !r.breaking() || firstLine(r) != "BREAKING" {
		t.Fatalf("want BREAKING first, got %q", firstLine(r))
	}
}

func TestDiff_whenAttributeBecameRequired_isBreaking(t *testing.T) {
	// Given/When
	r := diffFixture(t, "newly-required.json")
	// Then
	want := []attrChange{{Resource: "wayfinder_workspace", Attribute: "summary"}}
	if !slices.Equal(r.NowRequired, want) {
		t.Fatalf("now required = %v, want %v", r.NowRequired, want)
	}

	if firstLine(r) != "BREAKING" {
		t.Fatalf("want BREAKING first, got %q", firstLine(r))
	}
}

func TestDiff_whenNestedTypeChanged_isBreakingInTerraformTerms(t *testing.T) {
	// Given/When
	r := diffFixture(t, "type-change.json")
	// Then
	want := []attrChange{{Resource: "wayfinder_workspace", Attribute: "quota.max_cpu", Detail: "number -> string"}}
	if !slices.Equal(r.TypeChanged, want) {
		t.Fatalf("type changed = %v, want %v", r.TypeChanged, want)
	}

	if firstLine(r) != "BREAKING" {
		t.Fatalf("want BREAKING first, got %q", firstLine(r))
	}
}

func TestDiff_whenOnlyKindAdded_isNotBreaking(t *testing.T) {
	// Given/When
	r := diffFixture(t, "added-kind.json")
	// Then
	if !slices.Equal(r.AddedKinds, []string{"wayfinder_team"}) {
		t.Fatalf("added kinds = %v", r.AddedKinds)
	}

	if r.breaking() || firstLine(r) == "BREAKING" {
		t.Fatalf("added kind must not be breaking: %+v", r)
	}

	if len(r.Added)+len(r.Removed)+len(r.NowRequired)+len(r.TypeChanged)+len(r.RemovedKinds) != 0 {
		t.Fatalf("unexpected attribute changes: %+v", r)
	}
}

func TestDiff_whenKindRemoved_isBreaking(t *testing.T) {
	// Given: added-kind as old, original as new
	r, err := diffFiles(filepath.Join("testdata", "diff", "added-kind.json"), filepath.Join("testdata", "diff", "old.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Then
	if !slices.Equal(r.RemovedKinds, []string{"wayfinder_team"}) || firstLine(r) != "BREAKING" {
		t.Fatalf("removed kinds = %v, first line %q", r.RemovedKinds, firstLine(r))
	}
}

func TestDiff_whenIdentical_reportsNoChanges(t *testing.T) {
	// Given/When
	r := diffFixture(t, "old.json")
	// Then
	if r.breaking() || r.changed() {
		t.Fatalf("identical manifests reported changes: %+v", r)
	}
}

func TestResourceSchemas_whenKindVersionedHasPublishState(t *testing.T) {
	// Given
	m, err := manifest.Load("../../internal/manifest/testdata/golden-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	// When
	schemas, err := resourceSchemas(m)
	if err != nil {
		t.Fatal(err)
	}
	// Then
	if got := schemas["wayfinder_blueprint"].types["publish_state"]; got != "string" {
		t.Fatalf("versioned publish_state type = %q, want string", got)
	}

	if _, ok := schemas["wayfinder_workspace"].types["publish_state"]; ok {
		t.Fatal("non-versioned kind must not have publish_state")
	}

	r, err := diffFiles("../../internal/manifest/testdata/golden-manifest.json", "../../internal/manifest/testdata/golden-manifest.json")
	if err != nil {
		t.Fatal(err)
	}

	if r.breaking() || r.changed() {
		t.Fatalf("golden manifest against itself reported changes: %+v", r)
	}
}
