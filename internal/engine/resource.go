// Package engine implements manifest-driven Wayfinder resource lifecycles.
package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var (
	_ resource.ResourceWithConfigure   = (*kindResource)(nil)
	_ resource.ResourceWithImportState = (*kindResource)(nil)
)

type (
	pollOptions  struct{ initial, maximum time.Duration }
	kindResource struct {
		kind       manifest.Kind
		schema     schema.Schema
		mapping    *convert.Mapping
		mappingErr error
		client     *client.Client
		poll       pollOptions
	}
)

// NewResource binds a manifest kind to the caller's generated framework schema.
func NewResource(kind manifest.Kind, supplied schema.Schema) resource.Resource {
	return newResource(kind, supplied, pollOptions{initial: 2 * time.Second, maximum: 15 * time.Second})
}

func newResource(kind manifest.Kind, supplied schema.Schema, poll pollOptions) *kindResource {
	m, err := convert.NewMapping(kind)
	return &kindResource{kind: kind, schema: supplied, mapping: m, mappingErr: err, poll: poll}
}

func typeSuffix(k manifest.Kind) string {
	if k.TerraformName != "" {
		return k.TerraformName
	}
	return convert.SnakeCase(k.Kind)
}

func (r *kindResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + typeSuffix(r.kind)
}

func (r *kindResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.schema
}

func (r *kindResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := configuredClient(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.client = c
}

func configuredClient(data any) (*client.Client, diag.Diagnostics) {
	if data == nil {
		return nil, nil
	}
	c, ok := data.(*client.Client)
	if !ok {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", data))}
	}
	return c, nil
}

func (r *kindResource) prepared(d *diag.Diagnostics) bool {
	if r.mappingErr != nil {
		d.AddError("Invalid kind mapping", r.mappingErr.Error())
	}
	if r.client == nil {
		d.AddError("Wayfinder client unavailable", "The provider must be configured before reading or writing resources.")
	}
	return !d.HasError()
}
