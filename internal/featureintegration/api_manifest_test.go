// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"slices"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/featureintegration"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

func TestIntegrationRef_matchesManifest(t *testing.T) {
	// Given the embedded API contract.
	m, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}

	// When constructing a feature Integration reference.
	ref := featureintegration.IntegrationRef("azure")
	// Then its route and scope match the manifest.
	for _, k := range m.Kinds {
		if k.Kind == "Integration" {
			if ref.Group != k.Group || ref.Version != k.Version || ref.APIPath != k.APIPath || !slices.Contains(k.Scopes, "tenant") {
				t.Fatalf("reference %+v does not match Integration %+v", ref, k)
			}

			return
		}
	}

	t.Fatal("Integration missing from embedded manifest")
}

func TestEnablePath_whenSegmentsNeedEscaping(t *testing.T) {
	// Given a tenant and name containing path delimiters.
	// When constructing the general API action route.
	got := featureintegration.EnablePath("acme/one", "with space/?")
	// Then each value stays within one path segment.
	if want := "/api/v3/general/tenants/acme%2Fone/integrations/with%20space%2F%3F/enable"; got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}
