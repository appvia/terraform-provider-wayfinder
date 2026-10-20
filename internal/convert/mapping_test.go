// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package convert_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestMappingNames_whenAcronymsAndReservedNamesOccur(t *testing.T) {
	for _, tc := range []struct{ json, snake, attr string }{
		{"HTTPServerURL", "http_server_url", "http_server_url"},
		{"apiKey", "api_key", "api_key"},
		{"name", "name", "spec_name"},
		{"statusJSON", "status_json", "spec_status_json"},
		{"waitForReady", "wait_for_ready", "spec_wait_for_ready"},
		{"connection", "connection", "spec_connection"},
		{"count", "count", "spec_count"},
		{"dependsOn", "depends_on", "spec_depends_on"},
		{"forEach", "for_each", "spec_for_each"},
		{"lifecycle", "lifecycle", "spec_lifecycle"},
		{"provider", "provider", "spec_provider"},
		{"provisioner", "provisioner", "spec_provisioner"},
		{"locals", "locals", "spec_locals"},
		{"iamPolicyARNs", "iam_policy_arns", "iam_policy_arns"},
		{"privateZoneVPCs", "private_zone_vpcs", "private_zone_vpcs"},
		{"vnetIDs", "vnet_ids", "vnet_ids"},
		{"URLsToCheck", "urls_to_check", "urls_to_check"},
	} {
		t.Run(tc.json, func(t *testing.T) {
			// Given / When / Then
			if convert.SnakeCase(tc.json) != tc.snake || convert.AttrName(tc.json) != tc.attr {
				t.Fatal(tc.json)
			}
		})
	}
}

func TestMappingTypeIsIndependent_whenCallerMutatesReturnedType(t *testing.T) {
	// Given
	m := mapping(t, "Blueprint")
	typ := m.TFType()
	nested := typ.AttributeTypes["source"].(tftypes.Object)
	// When
	delete(typ.AttributeTypes, "name")
	delete(nested.AttributeTypes, "repository")
	// Then
	if m.TFType().AttributeTypes["name"] == nil || m.TFType().AttributeTypes["source"].(tftypes.Object).AttributeTypes["repository"] == nil {
		t.Fatal("mapping was mutated")
	}
}

func TestMappingRejectsInvalidFields_whenManifestMalformed(t *testing.T) {
	cycle := &manifest.Field{Type: "object"}
	cycle.Properties = map[string]*manifest.Field{"cycle": cycle}
	for _, field := range []*manifest.Field{{Type: "invalid"}, {Type: "array"}, {Type: "map"}, cycle} {
		t.Run(field.Type, func(t *testing.T) {
			// Given
			kind := manifest.Kind{Spec: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{"broken": field}}}
			// When
			_, err := convert.NewMapping(kind)
			// Then
			if !errors.Is(err, convert.ErrMapping) {
				t.Fatalf("error: %v", err)
			}
		})
	}
}

func TestScalarListMapRoundTrip_whenZeroAndEmptyValuesConfigured(t *testing.T) {
	// Given
	str := &manifest.Field{Type: "string"}
	kind := manifest.Kind{Kind: "Example", Group: "example.io", Version: "v1", Spec: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
		"enabled": {Type: "boolean"}, "ratio": {Type: "number"}, "names": {Type: "array", Items: str}, "tags": {Type: "map", Values: str},
	}}}
	m, err := convert.NewMapping(kind)
	if err != nil {
		t.Fatal(err)
	}
	prior := config(t, m, map[string]any{"enabled": false, "ratio": 0.0, "names": []tftypes.Value{tftypes.NewValue(tftypes.String, ""), tftypes.NewValue(tftypes.String, "last")}, "tags": map[string]tftypes.Value{"key": tftypes.NewValue(tftypes.String, "")}})
	x := object(t, `{"apiVersion":"example.io/v1","kind":"Example","metadata":{},"spec":{"enabled":false,"ratio":0,"names":["","last"],"tags":{"key":""}}}`)
	// When
	state, err := convert.FromObject(m, x, prior)
	if err != nil {
		t.Fatal(err)
	}
	got, err := convert.ToObject(m, state, convert.ToOpts{})
	if err != nil {
		t.Fatal(err)
	}
	// Then
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(object(t, string(encoded)), x) {
		t.Fatalf("object: %#v", got)
	}
}
