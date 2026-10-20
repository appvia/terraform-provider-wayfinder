// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

var update = flag.Bool("update", false, "update generator golden files")

func TestGenerate_whenGoldenManifest(t *testing.T) {
	// Given
	m, err := manifest.Load("../../internal/manifest/testdata/golden-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	// When
	files, err := render(m)
	if err != nil {
		t.Fatal(err)
	}
	// Then
	for name, got := range files {
		path := filepath.Join("testdata", name)
		if *update {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}

			if err := os.WriteFile(path, got, 0644); err != nil {
				t.Fatal(err)
			}
		}

		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(got, want) {
			t.Errorf("golden differs: %s (use -update)", name)
		}

		if strings.Contains(strings.ToLower(string(got)), "varset") {
			t.Errorf("private terminology in %s", name)
		}
	}

	if len(files) != 14 {
		t.Fatalf("got %d files, want 4 Go files + 1 generated test + 9 examples", len(files))
	}
}

func TestGenerate_whenReservedSpecName(t *testing.T) {
	// Given
	m, err := manifest.Load("../../internal/manifest/testdata/golden-manifest.json")
	if err != nil {
		t.Fatal(err)
	}

	m.Kinds[0].Spec.Properties["name"] = &manifest.Field{Type: "string", Required: true}
	// When
	files, err := render(m)
	if err != nil {
		t.Fatal(err)
	}
	// Then
	if !bytes.Contains(files["zz_schemas.go"], []byte(`"spec_name":`)) {
		t.Fatal("reserved spec name not prefixed")
	}
}

func TestGenerate_whenTerraformReservedSpecName(t *testing.T) {
	// Given
	m, err := manifest.Load("../../internal/manifest/testdata/golden-manifest.json")
	if err != nil {
		t.Fatal(err)
	}

	m.Kinds[0].Spec.Properties["dependsOn"] = &manifest.Field{Type: "array", Items: &manifest.Field{Type: "string"}}
	// When
	files, err := render(m)
	// Then
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(files["zz_schemas.go"], []byte(`"spec_depends_on":`)) {
		t.Fatal("Terraform reserved spec name not prefixed")
	}
}

func TestGenerate_whenEnumValueUndocumented(t *testing.T) {
	// Given an enum whose second value has no description.
	m, err := manifest.Load("../../internal/manifest/testdata/golden-manifest.json")
	if err != nil {
		t.Fatal(err)
	}

	m.Kinds[0].Spec.Properties["tier"] = &manifest.Field{Type: "string", Enum: []string{"gold", "bare"},
		EnumDocs: []manifest.EnumDoc{{Value: "gold", Description: "Top tier."}, {Value: "bare"}}}
	// When
	files, err := render(m)
	if err != nil {
		t.Fatal(err)
	}
	// Then the undocumented value is listed without a dangling separator.
	got := files["zz_schemas.go"]
	if !bytes.Contains(got, []byte("- `gold`: Top tier.")) || !bytes.Contains(got, []byte(`- `+"`bare`"+`"`)) || bytes.Contains(got, []byte("- `bare`:")) {
		t.Fatal("undocumented enum value rendered with a dangling separator")
	}
}

func TestGenerate_whenComputedParentHasSecret(t *testing.T) {
	// Given
	m, err := manifest.Load("../../internal/manifest/testdata/golden-manifest.json")
	if err != nil {
		t.Fatal(err)
	}

	m.Kinds[0].Spec.Properties["source"].Computed = true
	m.Kinds[0].Spec.Properties["source"].Properties["token"] = &manifest.Field{Type: "string", Sensitive: true}
	// When
	_, err = render(m)
	// Then
	if err == nil || !strings.Contains(err.Error(), "source.token") {
		t.Fatalf("expected named write-only error, got %v", err)
	}
}

func TestGenerate_whenKindsBecomeManifestOnlyRemovesStaleExamples(t *testing.T) {
	// Given
	m, err := manifest.Load("../../internal/manifest/testdata/golden-manifest.json")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()

	o := options{out: filepath.Join(dir, "generated"), examples: filepath.Join(dir, "examples")}
	if err := generate(m, o); err != nil {
		t.Fatal(err)
	}

	m.Kinds[0].Typed = false
	// When
	if err := generate(m, o); err != nil {
		t.Fatal(err)
	}
	// Then
	if _, err := os.Stat(filepath.Join(o.examples, "resources/wayfinder_blueprint/resource.tf")); !os.IsNotExist(err) {
		t.Fatalf("stale example remains: %v", err)
	}
}

func TestGenerate_whenComputedParentHasChildren(t *testing.T) {
	// Given a computed object whose children are not computed themselves.
	m, err := manifest.Load("../../internal/manifest/testdata/golden-manifest.json")
	if err != nil {
		t.Fatal(err)
	}

	m.Kinds[0].Spec.Properties["owner"] = &manifest.Field{Type: "object", Computed: true, Properties: map[string]*manifest.Field{
		"kind": {Type: "string", Required: true},
	}}
	// When
	files, err := render(m)
	if err != nil {
		t.Fatal(err)
	}
	// Then the server-filled child is Optional+Computed, so Terraform keeps its prior value.
	if !bytes.Contains(files["zz_schemas.go"], []byte(`"kind": rschema.StringAttribute{Optional: true, Computed: true,`)) {
		t.Fatal("child of a computed container is not Optional+Computed")
	}
}
