// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

// Package provider implements the Wayfinder Terraform provider: the provider block,
// authentication, and registration of every resource and data source.
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/appvia/terraform-provider-wayfinder/internal/generated"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifestresource"
	"github.com/appvia/terraform-provider-wayfinder/internal/vars"
)

var _ provider.Provider = (*wayfinderProvider)(nil)

type wayfinderProvider struct {
	version string
}

// New returns a provider factory for the given provider version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &wayfinderProvider{version: version}
	}
}

func (p *wayfinderProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "wayfinder"
	resp.Version = p.version
}

func (p *wayfinderProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerSchema()
}

// Resources registers one generated resource per typed kind, wayfinder_manifest, and the
// per-key wayfinder_variable and wayfinder_secret resources.
func (p *wayfinderProvider) Resources(_ context.Context) []func() resource.Resource {
	out := append([]func() resource.Resource{}, generated.Resources()...)
	out = append(out, manifestresource.NewResource)

	return append(out, vars.Resources()...)
}

// DataSources registers one generated data source per typed kind and wayfinder_variable.
func (p *wayfinderProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	out := append([]func() datasource.DataSource{}, generated.DataSources()...)
	return append(out, vars.DataSources()...)
}

func (p *wayfinderProvider) Functions(_ context.Context) []func() function.Function {
	return nil
}
