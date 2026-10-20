// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package vars

import (
	"context"
	"errors"
	"maps"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
)

var _ datasource.DataSourceWithConfigure = (*variableDataSource)(nil)

// NewVariableDataSource returns the wayfinder_variable data source.
func NewVariableDataSource() datasource.DataSource { return &variableDataSource{} }

type variableDataSource struct{ client *client.Client }

func (d *variableDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_variable"
}

func (d *variableDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := dataSourceScopeAttributes()
	maps.Copy(attrs, map[string]schema.Attribute{
		"value": schema.StringAttribute{
			Computed:            true,
			CustomType:          jsontypes.NormalizedType{},
			MarkdownDescription: "Value of the variable as a JSON document. Use `jsondecode` to read it.",
		},
	})
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads one variable from Wayfinder's variable and secret hierarchy at tenant, " +
			"workspace, environment or stack instance scope. Only the value stored at exactly that scope is returned. " +
			"Secrets cannot be read.",
		Attributes: attrs,
	}
}

func (d *variableDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := clientFrom(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *variableDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m variableModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k := m.scoped()
	item, err := get(ctx, d.client, k, kindVariables)
	if errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Variable not found", "Variable "+k.ID()+" not found.")
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read variable "+k.ID(), err.Error())
		return
	}
	m.ID = types.StringValue(k.ID())
	m.Value = jsontypes.NewNormalizedValue(string(item.Value))
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
