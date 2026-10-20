// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

//go:build wfmanifestgen

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func varScopeFixtureRoot(t *testing.T, source []byte) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "pkg/apiserver/handlers")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "handlers.go"), source, 0o644))
	return root
}

func TestVarScopeInventory_whenParsingFixtures(t *testing.T) {
	// Given source-only fixtures, without importing any handler package.
	cases := []struct {
		fixture string
		want    []string
		wantErr string
	}{
		{"ok", []string{"environment", "instance", "tenant", "workspace"}, ""},
		{"none", nil, "no registered varsHandler scopes found in pkg/apiserver/handlers; the vars API changed shape - update tools/manifestgen/varscopes.go"},
		{"nonliteral", nil, "registered varsHandler constructor newTenant: scopeName must be a string literal"},
		{"unregistered", []string{"tenant"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join("testdata/varscopes", tc.fixture, "handlers.go"))
			require.NoError(t, err)
			root := varScopeFixtureRoot(t, source)
			// When the inventory is parsed.
			scopes, err := varScopeInventory(root)
			// Then only registered constructors contribute sorted unique scopes.
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, scopes)
		})
	}
}

func TestVarScopeInventory_whenScopeNameInvalid(t *testing.T) {
	// Given a registered constructor with a missing or non-string scope.
	for name, fields := range map[string]string{"missing": "", "numeric": "scopeName: 42"} {
		t.Run(name, func(t *testing.T) {
			root := varScopeFixtureRoot(t, []byte("package handlers\n"+
				"func init() { apiregister.RegisterHandler(newTenant()) }\n"+
				"func newTenant() *varsHandler { return &varsHandler{"+fields+"} }\n"))
			// When the inventory is parsed.
			_, err := varScopeInventory(root)
			// Then discovery fails instead of silently dropping the constructor.
			require.ErrorContains(t, err, "newTenant: scopeName must be a string literal")
		})
	}
}

func TestVarScopeInventory_whenTestFilesAndSubdirectoriesPresent(t *testing.T) {
	// Given valid handlers with unparseable test and nested files.
	source, err := os.ReadFile("testdata/varscopes/ok/handlers.go")
	require.NoError(t, err)
	root := varScopeFixtureRoot(t, source)
	dir := filepath.Join(root, "pkg/apiserver/handlers")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "invalid_test.go"), []byte("not Go"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nested/invalid.go"), []byte("not Go"), 0o644))
	// When the inventory is parsed.
	scopes, err := varScopeInventory(root)
	// Then only top-level non-test Go files are inspected.
	require.NoError(t, err)
	require.Equal(t, []string{"environment", "instance", "tenant", "workspace"}, scopes)
}

func TestVarScopeInventory_whenSourceMalformed(t *testing.T) {
	// Given malformed handler source.
	root := varScopeFixtureRoot(t, []byte("package handlers\nfunc"))
	// When the inventory is parsed.
	_, err := varScopeInventory(root)
	// Then the source filename is reported.
	require.ErrorContains(t, err, "parse vars handler handlers.go:")
}

func TestCheckVarScopeCoverage_whenScopesDiffer(t *testing.T) {
	// Given missing, stale, or matching scope classifications.
	cases := []struct {
		name       string
		listed     []VarScopeEntry
		discovered []string
		missing    []string
		stale      []string
		message    string
	}{
		{"match", []VarScopeEntry{{Scope: "tenant", Class: ClassExclude, Reason: "unsupported"}},
			[]string{"tenant"}, nil, nil, ""},
		{"missing", nil, []string{"instance"}, []string{"instance"}, nil,
			"varScopes in kinds.yaml do not match the Wayfinder vars API:\n" +
				"  - scope \"instance\" is served at v3.4.0 but missing from kinds.yaml varScopes"},
		{"stale", []VarScopeEntry{{Scope: "instance"}}, nil, nil, []string{"instance"},
			"varScopes in kinds.yaml do not match the Wayfinder vars API:\n" +
				"  - scope \"instance\" is listed in kinds.yaml varScopes but no longer served at v3.4.0"},
		{"both sorted", []VarScopeEntry{{Scope: "workspace"}, {Scope: "tenant"}},
			[]string{"instance", "environment"}, []string{"environment", "instance"}, []string{"tenant", "workspace"},
			"varScopes in kinds.yaml do not match the Wayfinder vars API:\n" +
				"  - scope \"environment\" is served at v3.4.0 but missing from kinds.yaml varScopes\n" +
				"  - scope \"instance\" is served at v3.4.0 but missing from kinds.yaml varScopes\n" +
				"  - scope \"tenant\" is listed in kinds.yaml varScopes but no longer served at v3.4.0\n" +
				"  - scope \"workspace\" is listed in kinds.yaml varScopes but no longer served at v3.4.0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kinds := KindsFile{VarScopes: tc.listed}
			// When coverage is checked.
			err := CheckVarScopeCoverage(kinds, tc.discovered, "v3.4.0")
			// Then every difference is reported in sorted order.
			if tc.message == "" {
				require.NoError(t, err)
				return
			}
			var cerr *VarScopeCoverageError
			require.ErrorAs(t, err, &cerr)
			assert.Equal(t, tc.missing, cerr.Missing)
			assert.Equal(t, tc.stale, cerr.Stale)
			assert.EqualError(t, err, tc.message)
		})
	}
}

func TestGenerate_whenVarScopesClassified(t *testing.T) {
	// Given typed scopes surrounding an excluded scope.
	g, kinds := fixtureGenerator()
	kinds.VarScopes = []VarScopeEntry{
		{Scope: "environment", Class: ClassTyped},
		{Scope: "instance", Class: ClassExclude, Reason: "unsupported"},
		{Scope: "tenant", Class: ClassTyped},
	}
	// When generated.
	m, err := g.generate(kinds, FieldsFile{})
	// Then only typed scopes are recorded in sorted order.
	require.NoError(t, err)
	require.Equal(t, []string{"environment", "tenant"}, m.VarScopes)
}
