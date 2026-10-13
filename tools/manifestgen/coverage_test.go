//go:build wfmanifestgen

package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev3 "github.com/appvia/wayfinder/pkg/apis/core/v3"
)

func fixtureRegistry(gks ...GroupKind) []RegisteredKind {
	out := make([]RegisteredKind, 0, len(gks))
	for _, gk := range gks {
		out = append(out, RegisteredKind{GroupKind: gk})
	}
	return out
}

func fixtureKinds(gks ...GroupKind) KindsFile {
	var kf KindsFile
	for _, gk := range gks {
		kf.Kinds = append(kf.Kinds, KindEntry{Group: gk.Group, Kind: gk.Kind, Class: ClassTyped})
	}
	return kf
}

var (
	gkA = GroupKind{Group: "org", Kind: "Alpha"}
	gkB = GroupKind{Group: "org", Kind: "Beta"}
	gkZ = GroupKind{Group: "zzz", Kind: "FakeKind"}
)

func TestCheckCoverage_passes_on_exact_match(t *testing.T) {
	err := CheckCoverage(fixtureKinds(gkA, gkB), fixtureRegistry(gkB, gkA), "v1.0.0")
	assert.NoError(t, err)
}

func TestCheckCoverage_names_registered_kind_missing_from_kinds(t *testing.T) {
	// Given a registry with an extra fake kind
	reg := fixtureRegistry(gkA, gkB, gkZ)
	// When coverage is checked
	err := CheckCoverage(fixtureKinds(gkA, gkB), reg, "v1.0.0")
	// Then the error names the fake kind as missing
	var cerr *CoverageError
	require.ErrorAs(t, err, &cerr)
	assert.Equal(t, []GroupKind{gkZ}, cerr.Missing)
	assert.Empty(t, cerr.Stale)
	assert.Contains(t, err.Error(), "zzz/FakeKind is registered at v1.0.0 but missing from kinds.yaml")
}

func TestCheckCoverage_names_listed_kind_no_longer_registered(t *testing.T) {
	err := CheckCoverage(fixtureKinds(gkA, gkB), fixtureRegistry(gkA), "v1.2.3")
	var cerr *CoverageError
	require.ErrorAs(t, err, &cerr)
	assert.Equal(t, []GroupKind{gkB}, cerr.Stale)
	assert.Empty(t, cerr.Missing)
	assert.Contains(t, err.Error(), "org/Beta is listed in kinds.yaml but no longer exists at v1.2.3")
}

func TestCheckCoverage_reports_every_difference_sorted(t *testing.T) {
	err := CheckCoverage(fixtureKinds(gkB, gkA), fixtureRegistry(gkZ), "v1.0.0")
	var cerr *CoverageError
	require.ErrorAs(t, err, &cerr)
	assert.Equal(t, []GroupKind{gkZ}, cerr.Missing)
	assert.Equal(t, []GroupKind{gkA, gkB}, cerr.Stale)
}

func TestListKinds_prints_one_line_per_kind(t *testing.T) {
	reg := []RegisteredKind{
		{GroupKind: gkA, APIPath: "alphas", Scopes: corev3.ObjectScopeTenant | corev3.ObjectScopeWorkspace, Alpha: true},
		{GroupKind: gkB, Scopes: corev3.ObjectScopeSystem, Hidden: true, ReadOnly: true},
	}
	var out bytes.Buffer
	require.NoError(t, listKinds(&out, reg))
	assert.Equal(t,
		"org Alpha alphas tenant|workspace alpha=true hidden=false readOnly=false\n"+
			"org Beta - system alpha=false hidden=true readOnly=true\n",
		out.String())
}

// TestCommittedKinds_cover_pinned_registry validates the committed kinds.yaml
// and fields.yaml against the registry compiled from the pinned checkout.
func TestCommittedKinds_cover_pinned_registry(t *testing.T) {
	kindsPath, fieldsPath := os.Getenv("MANIFESTGEN_KINDS"), os.Getenv("MANIFESTGEN_FIELDS")
	require.NotEmpty(t, kindsPath, "MANIFESTGEN_KINDS must be set (run via make manifestgen-test)")
	require.NotEmpty(t, fieldsPath, "MANIFESTGEN_FIELDS must be set (run via make manifestgen-test)")

	kinds, err := LoadKinds(kindsPath)
	require.NoError(t, err)
	fields, err := LoadFields(fieldsPath)
	require.NoError(t, err)
	registry := wayfinderRegistry()

	require.NoError(t, CheckCoverage(kinds, registry, "the pinned tag"))
	require.NoError(t, CheckFieldKinds(kinds, fields))

	typed := 0
	class := make(map[GroupKind]KindClass, len(kinds.Kinds))
	for _, k := range kinds.Kinds {
		class[k.GroupKind()] = k.Class
		if k.Class == ClassTyped {
			typed++
		}
	}
	assert.GreaterOrEqual(t, typed, 30, "typed kinds")

	// Mechanical parts of the classification rule must hold on every bump.
	for _, rk := range registry {
		if rk.APIPath == "" || rk.Scopes.IsSystem() || rk.ReadOnly {
			assert.Equal(t, ClassExclude, class[rk.GroupKind], "%s is unserved, System-only or read-only", rk.GroupKind)
		}
	}
	for _, gk := range []GroupKind{
		{"org", "Tenant"}, {"org", "User"}, {"org", "Identity"}, {"org", "BillingPlan"}, {"org", "System"}, {"secrets", "VarSet"},
	} {
		assert.Equal(t, ClassExclude, class[gk], "%s must be excluded", gk)
	}
}
