// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakeapi_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi/fakekinds"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

const (
	integrationCollection = "/api/v3/resources/tenants/acme/integrations/v3/integrations"
	integrationGeneral    = "/api/v3/general/tenants/acme/integrations/"
)

func integrationKind(t *testing.T) fakeapi.KindSpec {
	t.Helper()

	m, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}

	for _, k := range m.Kinds {
		if k.Group == "integrations" && k.Version == "v3" && k.APIPath == "integrations" {
			return fakekinds.Spec(k)
		}
	}

	t.Fatal("Integration missing from embedded manifest")

	return fakeapi.KindSpec{}
}

func newFeatureServer(t *testing.T, features ...fakeapi.Feature) *fakeapi.Server {
	t.Helper()

	s := newServer(t, integrationKind(t))
	for _, f := range features {
		s.RegisterFeature(f)
	}

	return s
}

func featureRef(name string) fakeapi.Ref {
	return fakeapi.Ref{Tenant: tenant, Group: "integrations", Version: "v3", APIPath: "integrations", Name: "wayfinder." + name}
}

func featureObject(name string, enabled bool) map[string]any {
	return map[string]any{
		"metadata": map[string]any{"name": "wayfinder." + name},
		"spec":     map[string]any{"type": "feature", "enabled": enabled},
	}
}

func wantField(t *testing.T, got response, field string) {
	t.Helper()
	wantCode(t, got, http.StatusBadRequest)

	fields, _ := got.body["fieldErrors"].([]any)
	if len(fields) != 1 || dig(fields[0].(map[string]any), "field") != field || dig(fields[0].(map[string]any), "errCode") == "" {
		t.Fatalf("validation body = %s, want field %s", got.raw, field)
	}
}

func TestEnable_whenFeatureNotYetEnabled(t *testing.T) {
	for _, body := range []any{nil, map[string]any{}} {
		t.Run(map[bool]string{true: "empty_body", false: "empty_object"}[body == nil], func(t *testing.T) {
			// Given
			s := newFeatureServer(t, fakeapi.Feature{Name: "azure"})
			// When
			got := call(t, s, http.MethodPost, integrationGeneral+"azure/enable", body)
			// Then
			wantCode(t, got, http.StatusOK)

			if dig(got.body, "metadata", "name") != "wayfinder.azure" || dig(got.body, "metadata", "generation") != float64(1) ||
				dig(got.body, "spec", "type") != "feature" || dig(got.body, "spec", "enabled") != true || dig(got.body, "spec", "params") != nil {
				t.Errorf("created integration = %s", got.raw)
			}

			if stored, ok := s.Object(featureRef("azure")); !ok || !reflect.DeepEqual(stored, got.body) {
				t.Errorf("stored integration = %v, exists = %v", stored, ok)
			}
		})
	}
}

func TestEnable_whenAlreadyEnabledMergesParams(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(map[bool]string{true: "enabled", false: "disabled"}[enabled], func(t *testing.T) {
			// Given
			s := newFeatureServer(t, fakeapi.Feature{Name: "azure"})
			obj := featureObject("azure", enabled)
			obj["spec"].(map[string]any)["params"] = map[string]any{"keep": "old", "replace": "old", "clear": "old"}
			s.Seed(featureRef("azure"), obj)
			before, _ := s.Object(featureRef("azure"))
			// When
			got := call(t, s, http.MethodPost, integrationGeneral+"azure/enable", map[string]any{"params": map[string]any{"replace": "new", "clear": nil}})
			// Then
			wantCode(t, got, http.StatusOK)

			if dig(got.body, "spec", "enabled") != true || dig(got.body, "metadata", "generation") != float64(2) ||
				dig(got.body, "metadata", "uid") != dig(before, "metadata", "uid") {
				t.Errorf("updated integration = %s", got.raw)
			}

			if want := map[string]any{"keep": "old", "replace": "new", "clear": nil}; !reflect.DeepEqual(dig(got.body, "spec", "params"), want) {
				t.Errorf("merged params = %v, want %v", dig(got.body, "spec", "params"), want)
			}
		})
	}
}

func TestEnable_whenUnknownFeature(t *testing.T) {
	// Given no catalogue entries, including no implicit built-ins.
	s := newFeatureServer(t)
	// When
	got := call(t, s, http.MethodPost, integrationGeneral+"azure/enable", nil)
	// Then
	wantCode(t, got, http.StatusNotFound)
}

func TestEnable_whenCustomTemplate(t *testing.T) {
	// Given
	s := newFeatureServer(t, fakeapi.Feature{Name: "custom", Custom: true})
	// When
	got := call(t, s, http.MethodPost, integrationGeneral+"custom/enable", nil)
	// Then
	wantField(t, got, "name")
}

func TestEnable_whenBoolParamGivenAsString(t *testing.T) {
	for _, value := range []string{"true", "false"} {
		t.Run(value, func(t *testing.T) {
			// Given
			s := newFeatureServer(t, fakeapi.Feature{Name: "ai", BoolParams: []string{"enable_wayfinder"}})
			// When
			got := call(t, s, http.MethodPost, integrationGeneral+"ai/enable", map[string]any{"params": map[string]any{"enable_wayfinder": value, "other": value}})
			// Then
			wantCode(t, got, http.StatusOK)

			stored, _ := s.Object(featureRef("ai"))
			if dig(stored, "spec", "params", "enable_wayfinder") != (value == "true") || dig(stored, "spec", "params", "other") != value {
				t.Errorf("stored params = %v", dig(stored, "spec", "params"))
			}
		})
	}
}

func TestEnable_whenRequiredParamMissing(t *testing.T) {
	for _, exists := range []bool{false, true} {
		for _, params := range []map[string]any{nil, {"required": nil}} {
			t.Run(map[bool]string{true: "existing", false: "new"}[exists]+map[bool]string{true: "_absent", false: "_null"}[params == nil], func(t *testing.T) {
				// Given
				s := newFeatureServer(t, fakeapi.Feature{Name: "azure", RequiredParams: []string{"required"}})

				if exists {
					obj := featureObject("azure", false)
					obj["spec"].(map[string]any)["params"] = map[string]any{"untouched": "old"}
					s.Seed(featureRef("azure"), obj)
				}

				before, _ := s.Object(featureRef("azure"))
				// When
				got := call(t, s, http.MethodPost, integrationGeneral+"azure/enable", map[string]any{"params": params})
				// Then
				wantField(t, got, "spec.params.required")

				if after, found := s.Object(featureRef("azure")); found != exists || !reflect.DeepEqual(after, before) {
					t.Errorf("failed enable changed store: before %v, after %v", before, after)
				}
			})
		}
	}
}

func TestCreate_whenReservedFeatureName(t *testing.T) {
	for _, tc := range []struct{ name, typ, field string }{
		{"wayfinder.azure", "custom", "metadata.name"},
		{"azure", "custom", "metadata.name"},
		{"forged", "feature", "spec.type"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			s := newFeatureServer(t, fakeapi.Feature{Name: "azure"})
			obj := map[string]any{"metadata": map[string]any{"name": tc.name}, "spec": map[string]any{"type": tc.typ}}
			// When
			got := call(t, s, http.MethodPost, integrationCollection, obj)
			// Then
			wantField(t, got, tc.field)
		})
	}
}

func TestUpdate_whenDisablingAlwaysEnabledFeature(t *testing.T) {
	assertDisableRejected(t, fakeapi.Feature{Name: "dns", AlwaysEnabled: true})
}

func TestUpdate_whenDisablingWithDependents(t *testing.T) {
	assertDisableRejected(t, fakeapi.Feature{Name: "azure", BlockDisable: true})
}

func assertDisableRejected(t *testing.T, feature fakeapi.Feature) {
	t.Helper()
	// Given
	s := newFeatureServer(t, feature)
	s.Seed(featureRef(feature.Name), featureObject(feature.Name, true))
	before, _ := s.Object(featureRef(feature.Name))
	// When
	got := call(t, s, http.MethodPut, integrationCollection+"/wayfinder."+feature.Name, featureObject(feature.Name, false))
	// Then
	wantField(t, got, "spec.enabled")

	if after, _ := s.Object(featureRef(feature.Name)); !reflect.DeepEqual(after, before) {
		t.Errorf("failed disable changed store: before %v, after %v", before, after)
	}
}

func TestEffective_whenAlwaysEnabledFeature(t *testing.T) {
	for _, exists := range []bool{false, true} {
		t.Run(map[bool]string{true: "configured", false: "projected"}[exists], func(t *testing.T) {
			// Given
			s := newFeatureServer(t, fakeapi.Feature{Name: "dns", AlwaysEnabled: true})
			if exists {
				s.Seed(featureRef("dns"), featureObject("dns", true))
			}
			// When
			got := call(t, s, http.MethodGet, integrationGeneral+"dns/effective", nil)
			// Then
			wantCode(t, got, http.StatusOK)

			if got.body["alwaysEnabled"] != true || got.body["available"] != true || got.body["exists"] != exists ||
				dig(got.body, "integration", "spec", "type") != "feature" || dig(got.body, "integration", "spec", "enabled") != exists {
				t.Errorf("effective integration = %s", got.raw)
			}

			if _, found := s.Object(featureRef("dns")); found != exists {
				t.Error("projection unexpectedly stored an object")
			}
		})
	}
}

func TestEffective_whenUnknownFeature(t *testing.T) {
	// Given
	s := newFeatureServer(t)
	// When
	got := call(t, s, http.MethodGet, integrationGeneral+"unknown/effective", nil)
	// Then
	wantCode(t, got, http.StatusNotFound)
}
