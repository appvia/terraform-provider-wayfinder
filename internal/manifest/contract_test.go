// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"bytes"
	"errors"
	"go/parser"
	"go/token"
	"os"
	"regexp/syntax"
	"strconv"
	"strings"
	"testing"
)

const (
	goldenPath  = "testdata/golden-manifest.json"
	contractSrc = "contract.go"
)

// nonStdlibImports returns the import paths in src that are not standard
// library packages. Stdlib paths have no dot in their first path element.
func nonStdlibImports(src []byte) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), contractSrc, src, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	var bad []string

	for _, spec := range f.Imports {
		p, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}

		first := strings.SplitN(p, "/", 2)[0]
		if strings.Contains(first, ".") {
			bad = append(bad, p)
		}
	}

	return bad, nil
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return b
}

func TestGoldenRoundTripIsByteIdentical(t *testing.T) {
	raw := mustRead(t, goldenPath)

	m, err := Decode(raw)
	if err != nil {
		t.Fatalf("Decode golden: %v", err)
	}

	got, err := Encode(m)
	if err != nil {
		t.Fatalf("Encode golden: %v", err)
	}

	if !bytes.Equal(got, raw) {
		t.Fatalf("golden round-trip differs\n--- want (%d bytes)\n%s\n--- got (%d bytes)\n%s",
			len(raw), raw, len(got), got)
	}
}

func TestGoldenIsValid(t *testing.T) {
	if _, err := Parse(mustRead(t, goldenPath)); err != nil {
		t.Fatalf("golden fixture failed Parse/Validate: %v", err)
	}
}

func TestGoldenHasExpectedShape(t *testing.T) {
	m, err := Decode(mustRead(t, goldenPath))
	if err != nil {
		t.Fatalf("Decode golden: %v", err)
	}

	if len(m.Kinds) != 4 {
		t.Fatalf("golden kinds = %d, want 4", len(m.Kinds))
	}

	var typed, manifestOnly, versioned int

	for _, k := range m.Kinds {
		if k.Typed {
			typed++
		} else {
			manifestOnly++
		}

		if k.Versioned {
			versioned++
		}
	}

	if manifestOnly != 1 || versioned != 1 || typed != 3 {
		t.Fatalf("golden typed=%d manifestOnly=%d versioned=%d, want 3/1/1", typed, manifestOnly, versioned)
	}
}

func TestContractImportsAreStdlib(t *testing.T) {
	bad, err := nonStdlibImports(mustRead(t, contractSrc))
	if err != nil {
		t.Fatalf("parse %s: %v", contractSrc, err)
	}

	if len(bad) != 0 {
		t.Fatalf("%s imports non-stdlib packages: %v", contractSrc, bad)
	}
}

func TestImportCheckRejectsThirdPartyImport(t *testing.T) {
	src := "package manifest\n\nimport \"github.com/x/y\"\n"

	bad, err := nonStdlibImports([]byte(src))
	if err != nil {
		t.Fatalf("parse scratch: %v", err)
	}

	if len(bad) != 1 || bad[0] != "github.com/x/y" {
		t.Fatalf("bad imports = %v, want [github.com/x/y]", bad)
	}
}

func TestImportCheckAcceptsStdlibImports(t *testing.T) {
	src := "package manifest\n\nimport (\n\t\"encoding/json\"\n\t\"go/parser\"\n)\n"

	bad, err := nonStdlibImports([]byte(src))
	if err != nil {
		t.Fatalf("parse scratch: %v", err)
	}

	if len(bad) != 0 {
		t.Fatalf("stdlib imports flagged: %v", bad)
	}
}

func TestImportCheckFailsOnScratchCopyOfContract(t *testing.T) {
	orig := mustRead(t, contractSrc)

	marker := "import ("
	if !bytes.Contains(orig, []byte(marker)) {
		t.Fatalf("%s has no import block to mutate", contractSrc)
	}

	mutated := bytes.Replace(orig, []byte(marker), []byte(marker+"\n\t\"github.com/x/y\""), 1)

	bad, err := nonStdlibImports(mutated)
	if err != nil {
		t.Fatalf("parse mutated copy: %v", err)
	}

	if len(bad) == 0 {
		t.Fatalf("mutated contract copy with third-party import was not flagged")
	}
}

func TestFieldPropertyOrderFollowsDocument(t *testing.T) {
	in := `{"type":"object","properties":{"zeta":{"type":"string"},"alpha":{"type":"string"}}}`

	var f Field
	if err := f.UnmarshalJSON([]byte(in)); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got := strings.Join(f.PropertyOrder, ","); got != "zeta,alpha" {
		t.Fatalf("PropertyOrder = %q, want zeta,alpha", got)
	}

	out, err := f.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if string(out) != in {
		t.Fatalf("marshal = %s, want %s", out, in)
	}
}

func TestFieldWithoutPropertyOrderMarshalsSorted(t *testing.T) {
	f := Field{Type: "object", Properties: map[string]*Field{
		"b": {Type: "string"},
		"a": {Type: "string"},
	}}

	out, err := f.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}}}`
	if string(out) != want {
		t.Fatalf("marshal = %s, want %s", out, want)
	}
}

func objectField(props map[string]*Field, order ...string) *Field {
	return &Field{Type: "object", Properties: props, PropertyOrder: order}
}

// baseManifest is a valid manifest; each negative case mutates one thing.
func baseManifest() *Manifest {
	return &Manifest{
		ManifestVersion:  1,
		WayfinderVersion: "v3.3.2",
		WayfinderCommit:  strings.Repeat("a", 40),
		Kinds: []Kind{
			{
				Kind: "Alpha", Group: "core.appvia.io", Version: "v3", APIPath: "/api/v3/alphas",
				Typed: true, Scopes: []string{"tenant"}, HasStatus: true, WaitForReady: true,
				TerraformName: "alpha", Spec: objectField(nil),
			},
			{
				Kind: "Beta", Group: "core.appvia.io", Version: "v3", APIPath: "/api/v3/betas",
				Typed: false, Scopes: []string{"workspace"},
			},
			{
				Kind: "Gamma", Group: "core.appvia.io", Version: "v3", APIPath: "/api/v3/gammas",
				Typed: true, Scopes: []string{"workspace", "environment"}, Versioned: true,
				TerraformName: "gamma",
				Spec:          objectField(map[string]*Field{"foo": {Type: "string"}}, "foo"),
			},
		},
	}
}

func TestValidateBaseManifestPasses(t *testing.T) {
	if err := baseManifest().Validate(); err != nil {
		t.Fatalf("base manifest should be valid: %v", err)
	}
}

func TestValidate_whenPatternInvalid_wrapsRegexpError(t *testing.T) {
	m := baseManifest()
	m.Kinds[2].Spec.Properties["foo"].Pattern = "("

	err := m.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want invalid pattern error")
	}

	if !strings.Contains(err.Error(), "foo") || !strings.Contains(err.Error(), "pattern: invalid regular expression") {
		t.Fatalf("Validate() error = %q, want field path and pattern message", err)
	}

	var synErr *syntax.Error
	if !errors.As(err, &synErr) {
		t.Fatalf("Validate() error = %v, want it to wrap *syntax.Error", err)
	}
}

func TestValidateNegativeCases(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(m *Manifest)
		wantErr string
	}{
		{
			name: "unknown type carries field path",
			mutate: func(m *Manifest) {
				m.Kinds[2].Spec.Properties["foo"].Type = "x"
			},
			wantErr: `kinds[2].spec.properties.foo.type: unknown type "x"`,
		},
		{
			name: "bad scope",
			mutate: func(m *Manifest) {
				m.Kinds[0].Scopes = []string{"system"}
			},
			wantErr: `kinds[0].scopes[0]: unknown scope "system"`,
		},
		{
			name: "manifestVersion 2",
			mutate: func(m *Manifest) {
				m.ManifestVersion = 2
			},
			wantErr: "manifestVersion",
		},
		{
			name: "bad commit",
			mutate: func(m *Manifest) {
				m.WayfinderCommit = "not-a-commit"
			},
			wantErr: "wayfinderCommit",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := baseManifest()
			tc.mutate(m)

			err := m.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tc.wantErr)
			}

			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() error = %q, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestVarScopes_roundTripsThroughEncodeDecode(t *testing.T) {
	m := baseManifest()
	m.VarScopes = []string{"instance", "tenant"}

	raw, err := Encode(m)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	got, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if strings.Join(got.VarScopes, ",") != "instance,tenant" {
		t.Fatalf("VarScopes = %v, want [instance tenant]", got.VarScopes)
	}
}

func TestVarScopes_absentFieldIsValid(t *testing.T) {
	m := baseManifest()

	raw, err := Encode(m)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	if bytes.Contains(raw, []byte("varScopes")) {
		t.Fatalf("absent VarScopes must not be emitted, got %s", raw)
	}

	if _, err := Parse(raw); err != nil {
		t.Fatalf("manifest without varScopes should parse: %v", err)
	}
}

func TestVarScopes_rejectsInvalidLists(t *testing.T) {
	cases := []struct {
		name    string
		scopes  []string
		wantErr string
	}{
		{name: "unsorted", scopes: []string{"workspace", "tenant"}, wantErr: "varScopes[1]: must be sorted and unique"},
		{name: "duplicate", scopes: []string{"tenant", "tenant"}, wantErr: "varScopes[1]: must be sorted and unique"},
		{name: "uppercase", scopes: []string{"Bad"}, wantErr: `varScopes[0]: invalid scope "Bad"`},
		{name: "empty", scopes: []string{""}, wantErr: `varScopes[0]: invalid scope ""`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := baseManifest()
			m.VarScopes = tc.scopes

			err := m.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tc.wantErr)
			}

			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() error = %q, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}
