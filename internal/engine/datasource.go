// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var _ datasource.DataSourceWithConfigure = (*kindDataSource)(nil)

type kindDataSource struct {
	kind       manifest.Kind
	schema     schema.Schema
	mapping    *convert.Mapping
	mappingErr error
	client     *client.Client
}

// NewDataSource binds a single-object GET to the caller's data source schema.
func NewDataSource(kind manifest.Kind, supplied schema.Schema) datasource.DataSource {
	m, err := convert.NewMapping(kind)
	return &kindDataSource{kind: kind, schema: supplied, mapping: m, mappingErr: err}
}

func (s *kindDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + typeSuffix(s.kind)
}

func (s *kindDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = s.schema
}

func (s *kindDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, d := configuredClient(req.ProviderData)
	resp.Diagnostics.Append(d...)

	s.client = c
}

func (s *kindDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	r := kindResource{kind: s.kind, client: s.client, mapping: s.mapping, mappingErr: s.mappingErr}
	if !r.prepared(&resp.Diagnostics) {
		return
	}

	i, err := identityOf(s.kind, req.Config.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource identity", err.Error())
		return
	}

	obj, err := s.client.Get(ctx, reference(s.kind, i))
	if err != nil {
		r.apiDiagnostics(&resp.Diagnostics, err, i.ID())
		return
	}

	configured, err := attributes(req.Config.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource configuration", err.Error())
		return
	}

	prior := map[string]tftypes.Value{}
	for name, typ := range s.mapping.TFType().AttributeTypes {
		prior[name] = tftypes.NewValue(typ, nil)
		if value, ok := configured[name]; ok && value.Type().Equal(typ) {
			prior[name] = value
		}
	}

	state, err := convert.FromDataSource(s.mapping, obj, tftypes.NewValue(s.mapping.TFType(), prior))
	if err != nil {
		resp.Diagnostics.AddError("Unable to read resource", err.Error())
		return
	}

	values, err := attributes(state)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read resource", err.Error())
		return
	}

	typ := s.schema.Type().TerraformType(ctx).(tftypes.Object)

	projected := map[string]tftypes.Value{}
	for name, child := range typ.AttributeTypes {
		projected[name] = tftypes.NewValue(child, nil)
		if value, ok := values[name]; ok && value.Type().Equal(child) {
			projected[name] = value
		}
	}

	resp.State.Raw = tftypes.NewValue(typ, projected)
}
