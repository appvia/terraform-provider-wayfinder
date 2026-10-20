// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

//go:build wfmanifestgen

package main

import (
	"encoding/json"
	"reflect"
	"testing"

	corev3 "github.com/appvia/wayfinder/pkg/apis/core/v3"
	"github.com/stretchr/testify/require"
)

type documentedEnum string

func (documentedEnum) EnumValues() []corev3.EnumValue {
	return []corev3.EnumValue{{Value: "a", Description: "A"}, {Value: "b", Description: "B"}}
}

type emptyEnum string

func (emptyEnum) EnumValues() []corev3.EnumValue { return nil }

type EmbeddedFixture struct {
	Promoted string `json:"promoted"`
}
type metadataFixture struct {
	EmbeddedFixture `json:",inline"`
	Named           EmbeddedFixture `json:"named"`
	Choice          documentedEnum  `json:"choice"`
	Time            corev3.Time     `json:"time"`
	Opaque          json.RawMessage `json:"opaque"`
	Skipped         string          `json:"-"`
	private         string
	Count           int `json:"count"`
}

func (metadataFixture) SwaggerDoc() map[string]string {
	return map[string]string{"": "Fixture", "choice": "Select"}
}
func (metadataFixture) SwaggerValidation() map[string]corev3.FieldValidation {
	return map[string]corev3.FieldValidation{
		"choice": {Enum: []string{"b"}, Pattern: "^[ab]$", MinLength: corev3.IntPtr(1), MaxLength: corev3.IntPtr(2)},
		"count":  {Minimum: corev3.Float64Ptr(0), Maximum: corev3.Float64Ptr(10)},
		"opaque": {JSONType: "object"},
	}
}

func TestReflection_preserves_metadata_and_JSON_names(t *testing.T) {
	// Given generated metadata and JSON embedding rules.
	typ := reflect.TypeFor[metadataFixture]()
	// When reflected.
	f, err := reflectField(typ, make(map[reflect.Type]bool))
	// Then names, metadata and opaque JSON follow the contract.
	require.NoError(t, err)
	require.Equal(t, []string{"promoted", "named", "choice", "time", "opaque", "count"}, f.PropertyOrder)
	require.Equal(t, "string", f.Properties["named"].Properties["promoted"].Type)
	require.Equal(t, "Fixture", f.Description)
	choice := f.Properties["choice"]
	require.Equal(t, "Select", choice.Description)
	require.Equal(t, []string{"b"}, choice.Enum)
	require.Equal(t, "b", choice.EnumDocs[0].Value)
	require.Equal(t, "^[ab]$", choice.Pattern)
	require.Equal(t, 1, *choice.MinLength)
	require.Equal(t, 2, *choice.MaxLength)
	require.Equal(t, "date-time", f.Properties["time"].Format)
	require.Equal(t, "json", f.Properties["opaque"].Type)
	require.Equal(t, float64(0), *f.Properties["count"].Minimum)
	require.Equal(t, float64(10), *f.Properties["count"].Maximum)
}

func TestReflection_rejects_enum_without_values(t *testing.T) {
	// Given an enum provider without values.
	typ := reflect.TypeFor[emptyEnum]()
	// When reflected.
	_, err := reflectField(typ, make(map[reflect.Type]bool))
	// Then generation fails loudly.
	require.ErrorContains(t, err, "enum with no values")
}

type plainObject struct {
	corev3.Object `json:"-"`
	Spec          struct {
		Value string `json:"value"`
	} `json:"spec"`
}

func (*plainObject) GroupVersionKind() corev3.GroupVersionKind {
	return corev3.GroupVersionKind{Group: "fixtures", Version: "v3", Kind: "ExternalIdentity"}
}
func (*plainObject) APIPath() string                     { return "identities" }
func (*plainObject) SupportedScopes() corev3.ObjectScope { return corev3.ObjectScopeTenant }

func TestGenerate_fails_when_pin_lacks_encryption_metadata(t *testing.T) {
	// Given ExternalIdentity with no generated encryption methods anywhere.
	g, kinds := fixtureGenerator()
	g.registry = registeredKinds([]corev3.Object{&plainObject{}})
	// When generated.
	_, err := g.generate(kinds, FieldsFile{})
	// Then secrets cannot silently lose sensitivity.
	require.ErrorContains(t, err, "pinned Wayfinder tag lacks EncryptedFields metadata; pin a newer tag")
}

func TestGenerate_applies_flags_and_rejects_nested_missing_path(t *testing.T) {
	for _, path := range []string{"spec.rules.name", "spec.raw.missing.deep"} {
		t.Run(path, func(t *testing.T) {
			// Given an override into an array or opaque field.
			g, kinds := fixtureGenerator()
			fields := FieldsFile{Fields: []FieldEntry{{Group: "fixtures", Kind: "ExternalIdentity", Path: path, Immutable: true, Computed: true}}}
			// When generated.
			m, err := g.generate(kinds, fields)
			// Then flags reach array items, but traversal through JSON fails.
			if path == "spec.raw.missing.deep" {
				require.ErrorContains(t, err, path)
				return
			}
			require.NoError(t, err)
			field := m.Kinds[0].Spec.Properties["rules"].Items.Properties["name"]
			require.True(t, field.Immutable)
			require.True(t, field.Computed)
		})
	}
}
