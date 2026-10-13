package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

func TestCompiledSchemas_whenGoldenOrRealManifest(t *testing.T) {
	// Given
	input := os.Getenv("TFGEN_SMOKE_MANIFEST")
	if input == "" {
		input = "../../internal/manifest/testdata/golden-manifest.json"
	}
	m, err := manifest.Load(input)
	if err != nil {
		t.Fatal(err)
	}
	if selected := os.Getenv("TFGEN_SMOKE_KIND"); selected != "" {
		var kinds []manifest.Kind
		for _, k := range m.Kinds {
			if k.Kind == selected {
				kinds = append(kinds, k)
			}
		}
		if len(kinds) == 0 {
			t.Fatalf("smoke kind %s absent", selected)
		}
		m.Kinds = kinds
	}
	root, err := repositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(filepath.Join(root, "internal"), "generated_smoke_")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})
	files, err := render(m)
	if err != nil {
		t.Fatal(err)
	}
	if out := os.Getenv("TFGEN_SMOKE_OUT"); out != "" {
		if err := generate(m, options{out: filepath.Join(out, "generated"), examples: filepath.Join(out, "examples")}); err != nil {
			t.Fatal(err)
		}
	}
	for name, data := range files {
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	support, err := os.ReadFile(filepath.Join(root, "internal/generated/manifest.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.go"), support, 0644); err != nil {
		t.Fatal(err)
	}
	encoded, err := manifest.Encode(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), encoded, 0644); err != nil {
		t.Fatal(err)
	}
	var test strings.Builder
	test.WriteString(`package generated
import("context";"testing";"github.com/appvia/terraform-provider-wayfinder/internal/convert";"github.com/appvia/terraform-provider-wayfinder/internal/manifest")
func TestSmoke(t *testing.T) { ctx:=context.Background(); m,err:=manifest.Load("manifest.json"); if err!=nil {t.Fatal(err)}
`)
	for _, k := range m.TypedKinds() {
		id := strings.TrimPrefix(typeName(k), "wayfinder_")
		fmt.Fprintf(&test, `{k,_:=m.FindKind(%q,%q); mapping,err:=convert.NewMapping(*k); if err!=nil {t.Fatal(err)}; r:=ResourceSchema_%s(ctx); if d:=r.ValidateImplementation(ctx);d.HasError(){t.Fatal(%q,d)}; if !r.Type().TerraformType(ctx).Equal(mapping.TFType()){t.Fatal(%q)}; ds:=DataSourceSchema_%s(ctx);if d:=ds.ValidateImplementation(ctx);d.HasError(){t.Fatal(%q,d)};t.Log(%q)}
`, k.Group, k.Kind, id, k.Kind, k.Kind+" mapping differs", id, k.Kind, k.Kind+" schemas valid; mapping equal")
	}
	test.WriteString("}\n")
	if err := os.WriteFile(filepath.Join(dir, "smoke_test.go"), []byte(test.String()), 0644); err != nil {
		t.Fatal(err)
	}
	// When
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-v", "./internal/"+filepath.Base(dir))
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	// Then
	t.Log(string(output))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("manifest kinds=%d; resources=%d; data sources=%d", len(m.Kinds), len(m.TypedKinds()), len(m.TypedKinds()))
}
