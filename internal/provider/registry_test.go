// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"regexp"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/appvia/terraform-provider-wayfinder/internal/generated"
)

func TestProvider_registers_whenListingTypes(t *testing.T) {
	// Given the provider.
	ctx := context.Background()
	p := New("test")()

	var resources, dataSources, texts []string
	// When listing every resource and data source with its schema.
	for _, f := range p.Resources(ctx) {
		r := f()

		var m resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "wayfinder"}, &m)

		var s resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &s)

		if s.Diagnostics.HasError() {
			t.Fatalf("%s schema: %v", m.TypeName, s.Diagnostics)
		}

		resources = append(resources, m.TypeName)
		texts = append(texts, m.TypeName, s.Schema.Description, s.Schema.MarkdownDescription)
	}

	for _, f := range p.DataSources(ctx) {
		d := f()

		var m datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "wayfinder"}, &m)

		var s datasource.SchemaResponse
		d.Schema(ctx, datasource.SchemaRequest{}, &s)

		if s.Diagnostics.HasError() {
			t.Fatalf("%s schema: %v", m.TypeName, s.Diagnostics)
		}

		dataSources = append(dataSources, m.TypeName)
		texts = append(texts, m.TypeName, s.Schema.Description, s.Schema.MarkdownDescription)
	}

	var ps fwprovider.SchemaResponse
	p.Schema(ctx, fwprovider.SchemaRequest{}, &ps)

	texts = append(texts, ps.Schema.MarkdownDescription)
	for name, a := range ps.Schema.Attributes {
		texts = append(texts, name, a.GetMarkdownDescription())
	}
	// Then every generated type plus the hand-written ones is registered exactly once.
	wantResources := append(generated.ResourceTypeNames(), "wayfinder_manifest", "wayfinder_variable", "wayfinder_secret", "wayfinder_feature_integration", "wayfinder_webhook_rotation")
	wantDataSources := append(generated.ResourceTypeNames(), "wayfinder_variable")

	assertSameSet(t, "resources", resources, wantResources)
	assertSameSet(t, "data sources", dataSources, wantDataSources)
	// And no name or description mentions internal storage or Wayfinder's own installation
	// settings. Helm charts deployed by tenants (KubeResourcePlan) are legitimate content.
	for _, text := range texts {
		if banned := bannedText.FindString(text); banned != "" {
			t.Fatalf("%q mentions %q", text, banned)
		}
	}

	t.Logf("resources: %v", resources)
	t.Logf("data sources: %v", dataSources)
}

var bannedText = regexp.MustCompile(`(?i)varset|your administrator|this instance|\bWF_[A-Z]|wayfinder.{0,20}helm values|helm values.{0,20}wayfinder`)

func assertSameSet(t *testing.T, what string, got, want []string) {
	t.Helper()

	g, w := slices.Sorted(slices.Values(got)), slices.Sorted(slices.Values(want))
	if !slices.Equal(g, w) {
		t.Fatalf("%s = %v, want %v", what, g, w)
	}
}
