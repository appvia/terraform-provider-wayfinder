package convert_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func nestedMapping(t *testing.T) *convert.Mapping {
	t.Helper()
	str := &manifest.Field{Type: "string"}
	k := manifest.Kind{Kind: "Example", Group: "example.io", Version: "v1", Scopes: []string{"tenant"}, Spec: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
		"name":    str,
		"token":   {Type: "string", Sensitive: true},
		"entries": {Type: "map", Values: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{"displayName": str, "extra": str, "payload": {Type: "json"}, "token": {Type: "string", Sensitive: true}}}},
		"rules":   {Type: "array", Items: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{"displayName": str, "payload": {Type: "json"}, "token": {Type: "string", Sensitive: true}}}},
	}}}
	m, err := convert.NewMapping(k)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestNestedRoundTrip_whenListMapJSONAndReservedNameConfigured(t *testing.T) {
	// Given
	m := nestedMapping(t)
	attrs := map[string]any{"name": "resource", "spec_name": "spec-value", "token_wo": "secret", "token_wo_version": "1"}
	for key, text := range map[string]string{"rules": `[{"display_name":"second","payload_json":"{\"x\":[true]}","token_wo":null,"token_wo_version":null},{"display_name":"first","payload_json":"[1,2]","token_wo":null,"token_wo_version":null}]`, "entries": `{"key":{"display_name":"map","extra":null,"payload_json":"{\"y\":1}","token_wo":null,"token_wo_version":null}}`} {
		v, err := fixtureValue(m.TFType().AttributeTypes[key], text)
		if err != nil {
			t.Fatal(err)
		}
		attrs[key] = v
	}
	x := object(t, `{"apiVersion":"example.io/v1","kind":"Example","metadata":{"name":"resource"},"spec":{"name":"spec-value","rules":[{"displayName":"second","payload":{"x":[true]}},{"displayName":"first","payload":[1,2]}],"entries":{"key":{"displayName":"map","payload":{"y":1}}}}}`)
	// When
	state, err := convert.FromObject(m, x, config(t, m, attrs))
	if err != nil {
		t.Fatal(err)
	}
	got, err := convert.ToObject(m, state, convert.ToOpts{})
	if err != nil {
		t.Fatal(err)
	}
	// Then
	if !reflect.DeepEqual(got, x) {
		t.Fatalf("got %#v, want %#v", got, x)
	}
}

func TestTokenCreate_whenWriteOnlyAttributeSelected(t *testing.T) {
	// Given
	m := nestedMapping(t)
	// When
	got, err := convert.ToObject(m, config(t, m, map[string]any{"token_wo": "send", "token_wo_version": "v2"}), convert.ToOpts{IncludeWriteOnly: map[string]bool{"token_wo": true}})
	// Then
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got["spec"], map[string]any{"token": "send"}) {
		t.Fatalf("spec: %#v", got["spec"])
	}
}

func TestNestedRead_whenOptionalChildNotConfigured(t *testing.T) {
	// Given
	m := nestedMapping(t)
	v, err := fixtureValue(m.TFType().AttributeTypes["entries"], `{"key":{"display_name":"map","extra":null,"payload_json":null,"token_wo":"SECRET","token_wo_version":"v1"}}`)
	if err != nil {
		t.Fatal(err)
	}
	x := object(t, `{"spec":{"entries":{"key":{"displayName":"new","extra":"server default","token":"LEAK"}}}}`)
	// When
	state, err := convert.FromObject(m, x, config(t, m, map[string]any{"entries": v}))
	// Then
	if err != nil {
		t.Fatal(err)
	}
	entry := values(t, values(t, values(t, state)["entries"])["key"])
	if !entry["extra"].IsNull() || !entry["token_wo"].IsNull() {
		t.Fatal("unconfigured child or secret leaked")
	}
	if !entry["token_wo_version"].Equal(tftypes.NewValue(tftypes.String, "v1")) {
		t.Fatal("nested version missing")
	}
}

func TestNestedWriteOnly_whenPathSelected(t *testing.T) {
	// Given
	m := nestedMapping(t)
	v, err := fixtureValue(m.TFType().AttributeTypes["rules"], `[{"display_name":"rule","payload_json":null,"token_wo":"new","token_wo_version":"v2"}]`)
	if err != nil {
		t.Fatal(err)
	}
	base := object(t, `{"spec":{"rules":[{"displayName":"old","token":"","serverKey":true}]}}`)
	// When
	got, err := convert.ToObject(m, config(t, m, map[string]any{"rules": v}), convert.ToOpts{Base: base, IncludeWriteOnly: map[string]bool{"rules[0].token_wo": true}})
	// Then
	if err != nil {
		t.Fatal(err)
	}
	want := object(t, `{"rules":[{"displayName":"rule","token":"new","serverKey":true}]}`)
	if !reflect.DeepEqual(got["spec"], want) {
		t.Fatalf("spec: %#v", got["spec"])
	}
}

func TestMappingRejectsCollision_whenNamesNormalizeIdentically(t *testing.T) {
	// Given
	k := manifest.Kind{Spec: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{"fooBar": {Type: "string"}, "foo_bar": {Type: "string"}}}}
	// When
	_, err := convert.NewMapping(k)
	// Then
	if !errors.Is(err, convert.ErrMapping) {
		t.Fatalf("error: %v", err)
	}
}

func TestConversionOmitsUnknown_whenConfigured(t *testing.T) {
	// Given
	m := mapping(t, "Integration")
	cfg := config(t, m, map[string]any{"endpoint": tftypes.NewValue(tftypes.String, tftypes.UnknownValue), "region": nil})
	// When
	got, err := convert.ToObject(m, cfg, convert.ToOpts{})
	// Then
	if err != nil {
		t.Fatal(err)
	}
	if len(got["spec"].(map[string]any)) != 0 {
		t.Fatal("unknown or null sent")
	}
}

func TestReadErrorsWithNestedPath_whenServerTypeInvalid(t *testing.T) {
	// Given
	m := mapping(t, "Blueprint")
	x := object(t, `{"spec":{"source":{"repository":true}}}`)
	// When
	_, err := convert.FromObject(m, x, config(t, m, nil))
	// Then
	if err == nil || !strings.Contains(err.Error(), "source") || !strings.Contains(err.Error(), "repository") {
		t.Fatalf("error: %v", err)
	}
}

func TestFrameworkInputsRetained_whenRead(t *testing.T) {
	// Given
	m := mapping(t, "Workspace")
	prior := config(t, m, map[string]any{"wait_for_ready": false, "timeouts": map[string]tftypes.Value{"create": tftypes.NewValue(tftypes.String, "1m"), "update": tftypes.NewValue(tftypes.String, nil), "delete": tftypes.NewValue(tftypes.String, "2m")}})
	// When
	state, err := convert.FromObject(m, object(t, `{"metadata":{"name":"demo"}}`), prior)
	// Then
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"timeouts", "wait_for_ready"} {
		if !values(t, state)[key].Equal(values(t, prior)[key]) {
			t.Fatal(key)
		}
	}
}

func TestJSONNumberRoundTrip_whenLargeInteger(t *testing.T) {
	// Given
	m := mapping(t, "Integration")
	x := map[string]any{"spec": map[string]any{"timeoutSeconds": json.Number("9007199254740993")}}
	// When
	state, err := convert.FromObject(m, x, config(t, m, map[string]any{"timeout_seconds": 1}))
	if err != nil {
		t.Fatal(err)
	}
	got, err := convert.ToObject(m, state, convert.ToOpts{})
	// Then
	if err != nil {
		t.Fatal(err)
	}
	if got["spec"].(map[string]any)["timeoutSeconds"] != int64(9007199254740993) {
		t.Fatal("precision lost")
	}
}
