// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package convert_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func mapping(t *testing.T, name string) *convert.Mapping {
	t.Helper()

	m, err := manifest.Load("../manifest/testdata/golden-manifest.json")
	if err != nil {
		t.Fatal(err)
	}

	k, ok := m.FindKind("core.appvia.io", name)
	if !ok {
		t.Fatal(name)
	}

	out, err := convert.NewMapping(*k)
	if err != nil {
		t.Fatal(err)
	}

	return out
}

func config(t *testing.T, m *convert.Mapping, fields map[string]any) tftypes.Value {
	t.Helper()

	attrs := map[string]tftypes.Value{}

	for name, typ := range m.TFType().AttributeTypes {
		if v, ok := fields[name].(tftypes.Value); ok {
			attrs[name] = v
		} else {
			attrs[name] = tftypes.NewValue(typ, fields[name])
		}
	}

	return tftypes.NewValue(m.TFType(), attrs)
}

func values(t *testing.T, v tftypes.Value) map[string]tftypes.Value {
	t.Helper()

	var out map[string]tftypes.Value
	if err := v.As(&out); err != nil {
		t.Fatal(err)
	}

	return out
}

func object(t *testing.T, text string) map[string]any {
	t.Helper()

	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatal(err)
	}

	return out
}

func TestGoldenRoundTrip_whenAllKindsConfigured(t *testing.T) {
	for _, tc := range []struct {
		kind, spec string
		cfg        map[string]any
	}{
		{"Blueprint", `{"source":{"repository":"https://git.example","ref":"main"},"inputs":[{"name":"z","type":"string","required":true},{"name":"a","type":"number","required":false}],"publishedAt":"today"}`, map[string]any{"workspace": "ws", "version": "1.2.3", "source": map[string]tftypes.Value{"repository": tftypes.NewValue(tftypes.String, "https://git.example"), "ref": tftypes.NewValue(tftypes.String, "main")}, "inputs": []tftypes.Value{tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"name": tftypes.String, "type": tftypes.String, "required": tftypes.Bool}}, map[string]tftypes.Value{"name": tftypes.NewValue(tftypes.String, "z"), "type": tftypes.NewValue(tftypes.String, "string"), "required": tftypes.NewValue(tftypes.Bool, true)}), tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"name": tftypes.String, "type": tftypes.String, "required": tftypes.Bool}}, map[string]tftypes.Value{"name": tftypes.NewValue(tftypes.String, "a"), "type": tftypes.NewValue(tftypes.String, "number"), "required": tftypes.NewValue(tftypes.Bool, false)})}}},
		{"Integration", `{"endpoint":"https://example","options":{"nested":[1,true]},"timeoutSeconds":30,"region":"eu-west-1","lastSyncedAt":"today"}`, map[string]any{"workspace": "ws", "environment": "env", "endpoint": "https://example", "options_json": `{"nested":[1,true]}`, "timeout_seconds": 30, "region": "eu-west-1"}},
		{"Pipeline", `{"trigger":"manual","steps":[]}`, map[string]any{"workspace": "ws", "environment": "env", "trigger": "manual", "steps": []tftypes.Value{}}},
		{"Workspace", `{"displayName":"Demo","cloudProvider":"aws","members":[],"createdAt":"today"}`, map[string]any{"display_name": "Demo", "cloud_provider": "aws", "members": []tftypes.Value{}}},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			// Given: API object and an independent configured prior.
			m := mapping(t, tc.kind)
			tc.cfg["name"] = "demo"
			tc.cfg["labels"] = map[string]tftypes.Value{"team": tftypes.NewValue(tftypes.String, "dev")}
			x := object(t, `{"apiVersion":"core.appvia.io/v3","kind":"`+tc.kind+`","metadata":{"name":"demo","labels":{"team":"dev"}},"spec":`+tc.spec+`}`)
			meta := x["metadata"].(map[string]any)

			for _, key := range []string{"workspace", "environment", "version"} {
				if v, ok := tc.cfg[key]; ok {
					meta[key] = v
				}
			}
			// When
			state, err := convert.FromObject(m, x, config(t, m, tc.cfg))
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
				t.Fatalf("got %#v, want %#v", got, x)
			}
		})
	}
}

func TestCreateIncludesWriteOnly_whenExplicitlySelected(t *testing.T) {
	// Given
	m := mapping(t, "Integration")
	cfg := config(t, m, map[string]any{"api_key_wo": "secret", "api_key_wo_version": "2", "headers_wo": map[string]tftypes.Value{"X": tftypes.NewValue(tftypes.String, "key")}})
	// When
	got, err := convert.ToObject(m, cfg, convert.ToOpts{IncludeWriteOnly: map[string]bool{"api_key_wo": true, "headers_wo": true}})
	// Then
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]any{"apiKey": "secret", "headers": map[string]any{"X": "key"}}
	if !reflect.DeepEqual(got["spec"], want) {
		t.Fatalf("spec: %#v", got["spec"])
	}
}

func TestUpdatePreservesBlankSecrets_whenVersionsUnchanged(t *testing.T) {
	// Given
	m := mapping(t, "Integration")
	base := object(t, `{"metadata":{"generation":7,"labels":{"x":"y"}},"spec":{"apiKey":"","headers":{"old":""},"unmanaged":true}}`)
	cfg := config(t, m, map[string]any{"api_key_wo": "DO NOT SEND", "api_key_wo_version": "1", "headers_wo": map[string]tftypes.Value{"new": tftypes.NewValue(tftypes.String, "DO NOT SEND")}, "labels": map[string]tftypes.Value{"team": tftypes.NewValue(tftypes.String, "dev")}})
	// When
	got, err := convert.ToObject(m, cfg, convert.ToOpts{Base: base})
	// Then
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got["spec"], base["spec"]) {
		t.Fatalf("spec: %#v", got["spec"])
	}

	if !reflect.DeepEqual(base, object(t, `{"metadata":{"generation":7,"labels":{"x":"y"}},"spec":{"apiKey":"","headers":{"old":""},"unmanaged":true}}`)) {
		t.Fatal("mutated Base")
	}

	labels := got["metadata"].(map[string]any)["labels"]
	if !reflect.DeepEqual(labels, map[string]any{"x": "y", "team": "dev"}) {
		t.Fatalf("labels: %#v", labels)
	}
}

func TestReadFiltersDrift_whenServerReturnsUnconfiguredFields(t *testing.T) {
	// Given
	m := mapping(t, "Integration")
	prior := config(t, m, map[string]any{"api_key_wo": "secret", "api_key_wo_version": "keep", "headers_wo": map[string]tftypes.Value{"x": tftypes.NewValue(tftypes.String, "secret")}, "labels": map[string]tftypes.Value{"team": tftypes.NewValue(tftypes.String, "old")}})
	x := object(t, `{"metadata":{"name":"demo","workspace":"ws","labels":{"team":"new","x":"y"}},"spec":{"apiKey":"LEAK","headers":{"x":"LEAK"},"region":"us-east-1","lastSyncedAt":"server"}}`)
	// When
	got, err := convert.FromObject(m, x, prior)
	// Then
	if err != nil {
		t.Fatal(err)
	}

	v := values(t, got)
	for _, key := range []string{"api_key_wo", "headers_wo", "region"} {
		if !v[key].IsNull() {
			t.Fatal(key + " should be null")
		}
	}

	if !v["api_key_wo_version"].Equal(values(t, prior)["api_key_wo_version"]) {
		t.Fatal("version not retained")
	}

	if !v["last_synced_at"].Equal(tftypes.NewValue(tftypes.String, "server")) {
		t.Fatal("computed field missing")
	}

	if !v["labels"].Equal(tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{"team": tftypes.NewValue(tftypes.String, "new")})) {
		t.Fatal("unmanaged label in state")
	}
}

func TestToObjectErrorsWithPath_whenJSONInvalid(t *testing.T) {
	// Given
	m := mapping(t, "Integration")
	// When
	_, err := convert.ToObject(m, config(t, m, map[string]any{"options_json": "{"}), convert.ToOpts{})
	// Then
	if err == nil || !strings.Contains(err.Error(), "options_json") {
		t.Fatalf("error: %v", err)
	}
}
