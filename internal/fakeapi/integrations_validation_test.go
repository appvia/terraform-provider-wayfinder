// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakeapi_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

func TestEnable_whenRequiredParamAlreadyStored(t *testing.T) {
	// Given a disabled feature whose required bool param is already configured as false.
	s := newFeatureServer(t, fakeapi.Feature{Name: "ai", RequiredParams: []string{"enable_wayfinder"}})
	obj := featureObject("ai", false)
	obj["spec"].(map[string]any)["params"] = map[string]any{"enable_wayfinder": false}
	s.Seed(featureRef("ai"), obj)
	// When
	got := call(t, s, http.MethodPost, integrationGeneral+"ai/enable", nil)
	// Then
	wantCode(t, got, http.StatusOK)

	if dig(got.body, "spec", "enabled") != true || dig(got.body, "spec", "params", "enable_wayfinder") != false {
		t.Errorf("enabled integration = %s", got.raw)
	}
}

func TestUpdate_whenRequiredParamMissing(t *testing.T) {
	for _, params := range []map[string]any{nil, {"required": nil}} {
		t.Run(map[bool]string{true: "absent", false: "null"}[params == nil], func(t *testing.T) {
			// Given
			s := newFeatureServer(t, fakeapi.Feature{Name: "azure", RequiredParams: []string{"required"}})
			obj := featureObject("azure", true)
			obj["spec"].(map[string]any)["params"] = map[string]any{"required": "old"}
			s.Seed(featureRef("azure"), obj)
			before, _ := s.Object(featureRef("azure"))
			obj["spec"].(map[string]any)["params"] = params
			// When
			got := call(t, s, http.MethodPut, integrationCollection+"/wayfinder.azure", obj)
			// Then
			wantField(t, got, "spec.params.required")

			if after, _ := s.Object(featureRef("azure")); !reflect.DeepEqual(after, before) {
				t.Errorf("failed update changed store: before %v, after %v", before, after)
			}
		})
	}
}

func TestUpdate_whenDisablingConfiguredFeature(t *testing.T) {
	// Given disabling does not require params.
	s := newFeatureServer(t, fakeapi.Feature{Name: "azure", RequiredParams: []string{"required"}})
	s.Seed(featureRef("azure"), featureObject("azure", true))
	// When
	got := call(t, s, http.MethodPut, integrationCollection+"/wayfinder.azure", featureObject("azure", false))
	// Then
	wantCode(t, got, http.StatusOK)

	if dig(got.body, "spec", "enabled") != false || dig(got.body, "metadata", "generation") != float64(2) {
		t.Errorf("disabled integration = %s", got.raw)
	}
}

func TestCreate_whenCustomIntegrationNameAllowed(t *testing.T) {
	// Given a custom catalogue template, whose bare name is not reserved.
	s := newFeatureServer(t, fakeapi.Feature{Name: "datadog", Custom: true})
	obj := map[string]any{"metadata": map[string]any{"name": "datadog"}, "spec": map[string]any{"type": "custom"}}
	// When
	got := call(t, s, http.MethodPost, integrationCollection, obj)
	// Then
	wantCode(t, got, http.StatusOK)
}

func TestCreate_whenAnotherKindUsesFeatureName(t *testing.T) {
	// Given feature validation is only installed on the Integration kind.
	s := newFeatureServer(t, fakeapi.Feature{Name: "azure"})
	s.RegisterKind(identityKind())
	// When
	got := call(t, s, http.MethodPost, identityCollection, identityObject("wayfinder.azure"))
	// Then
	wantCode(t, got, http.StatusOK)
}

func TestRegisterFeature_whenKindAlreadyHasValidation(t *testing.T) {
	// Given a kind validation hook and several feature registrations.
	k := integrationKind(t)
	k.Validate = func(_ fakeapi.Operation, _ map[string]any) []fakeapi.FieldError {
		return []fakeapi.FieldError{{Field: "spec.other", ErrCode: "invalidValue", Message: "refused"}}
	}
	s := newServer(t, k)
	s.RegisterFeature(fakeapi.Feature{Name: "azure"})
	s.RegisterFeature(fakeapi.Feature{Name: "ai"})
	s.RegisterFeature(fakeapi.Feature{Name: "azure", RequiredParams: []string{"required"}})
	// When
	got := call(t, s, http.MethodPost, integrationCollection, map[string]any{"metadata": map[string]any{"name": "custom"}})
	// Then the original hook still rejects once, rather than being dropped or installed twice.
	wantField(t, got, "spec.other")
}

func TestEnable_whenTenantAndNameAreEscaped(t *testing.T) {
	// Given a catalogue entry containing an escaped path separator.
	s := newFeatureServer(t, fakeapi.Feature{Name: "azure/preview"})
	// When
	got := call(t, s, http.MethodPost, "/api/v3/general/tenants/acme%2Fother/integrations/azure%2Fpreview/enable", nil)
	// Then
	wantCode(t, got, http.StatusOK)

	ref := featureRef("azure/preview")
	ref.Tenant = "acme/other"

	if obj, exists := s.Object(ref); !exists || dig(obj, "metadata", "name") != "wayfinder.azure/preview" {
		t.Errorf("escaped route stored %v, exists = %v", obj, exists)
	}

	if _, exists := s.Object(featureRef("azure/preview")); exists {
		t.Error("enable crossed tenant boundary")
	}
}
