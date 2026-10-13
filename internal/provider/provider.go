// Package provider implements the Wayfinder Terraform provider.
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
	resp.Schema = schema.Schema{}
}

func (p *wayfinderProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}

func (p *wayfinderProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}

func (p *wayfinderProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func (p *wayfinderProvider) Functions(_ context.Context) []func() function.Function {
	return nil
}
