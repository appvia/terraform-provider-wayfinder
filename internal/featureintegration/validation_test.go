// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCreate_whenUnknownFeature(t *testing.T) {
	// Given a name absent from the feature catalogue.
	e := newEnv(t)
	// When applying, then the name diagnostic explains how to list available features.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{{
			Config:      featureHCL("unknown", ""),
			ExpectError: regexp.MustCompile("(?s)Unknown feature integration.*wf enable integration"),
		}},
	})

	if got := e.fake.CountRequests(http.MethodPost, ""); got != 1 {
		t.Errorf("POST requests = %d, want 1", got)
	}
}

func TestCreate_whenCustomIntegrationName(t *testing.T) {
	// Given a custom integration template rather than a feature.
	e := newEnv(t)
	// When applying, then the server's name validation is preserved.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{{
			Config:      featureHCL("datadog", ""),
			ExpectError: regexp.MustCompile(`(?s)Wayfinder validation failed.*custom integration.*installed, not enabled`),
		}},
	})
}

func TestCreate_whenRequiredParamMissing(t *testing.T) {
	// Given a feature whose token parameter is required.
	e := newEnv(t)
	// When enabling without it, then the parameter validation is returned.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{{
			Config:      featureHCL("needsconf", ""),
			ExpectError: regexp.MustCompile(`(?s)Wayfinder validation failed.*token is required`),
		}},
	})
}

func TestCreate_whenParamTypeCoerced(t *testing.T) {
	// Given a JSON string where the feature expects a boolean.
	e := newEnv(t)
	// When enabling, then the apply fails with the correctly typed value to configure.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		// Cleanup must see the tainted resource in state and disable it, not leave it untracked.
		CheckDestroy: expectEnabled(e, "ai", false),
		Steps: []resource.TestStep{{
			Config:      featureHCL("ai", `params = { enable_wayfinder = jsonencode("false") }`),
			ExpectError: regexp.MustCompile(`(?s)Wayfinder stored a different value.*jsonencode\(false\)`),
		}},
	})
}

func TestValidate_whenNamePrefixed(t *testing.T) {
	// Given an object name instead of a bare feature name.
	e := newEnv(t)
	// When planning, then validation rejects it before any API request.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{{
			Config:      featureHCL("wayfinder.azure", ""),
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`Use the bare feature name`),
		}},
	})

	if got := len(e.fake.Requests()); got != 0 {
		t.Errorf("API requests = %d, want 0", got)
	}
}

func TestValidate_whenNameEmpty(t *testing.T) {
	// Given an empty name.
	e := newEnv(t)
	// When planning, then validation rejects it before any API request.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{{
			Config:      featureHCL("", ""),
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`Use the bare feature name`),
		}},
	})

	if got := len(e.fake.Requests()); got != 0 {
		t.Errorf("API requests = %d, want 0", got)
	}
}
