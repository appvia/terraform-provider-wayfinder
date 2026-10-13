// Package manifestresource implements wayfinder_manifest: one Wayfinder object managed from a
// YAML or JSON manifest, for any kind in the provider manifest (including kinds without a
// dedicated resource). CRUD, readiness and error handling follow the generic engine.
package manifestresource

import (
	"context"
	"fmt"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/engine"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.ResourceWithConfigure      = (*manifestResource)(nil)
	_ resource.ResourceWithImportState    = (*manifestResource)(nil)
	_ resource.ResourceWithModifyPlan     = (*manifestResource)(nil)
	_ resource.ResourceWithValidateConfig = (*manifestResource)(nil)
)

// statusTypes is the shape of the computed status attribute.
var statusTypes = map[string]attr.Type{"status": types.StringType, "message": types.StringType, "detail": types.StringType}

type manifestResource struct {
	manifest    *manifest.Manifest
	manifestErr error
	client      *client.Client
	poll        engine.Poll
}

// model is the Terraform state of wayfinder_manifest.
type model struct {
	ID           types.String         `tfsdk:"id"`
	Manifest     types.String         `tfsdk:"manifest"`
	WaitForReady types.Bool           `tfsdk:"wait_for_ready"`
	Timeouts     timeouts.Value       `tfsdk:"timeouts"`
	ObjectJSON   jsontypes.Normalized `tfsdk:"object_json"`
	Status       types.Object         `tfsdk:"status"`
}

// NewResource returns wayfinder_manifest bound to the manifest compiled into the provider.
func NewResource() resource.Resource {
	m, err := manifest.LoadEmbedded()
	return &manifestResource{manifest: m, manifestErr: err, poll: engine.DefaultPoll()}
}

// NewResourceWithManifest returns wayfinder_manifest bound to m (used by tests).
func NewResourceWithManifest(m *manifest.Manifest) resource.Resource {
	return newResource(m, engine.DefaultPoll())
}

func newResource(m *manifest.Manifest, poll engine.Poll) *manifestResource {
	return &manifestResource{manifest: m, poll: poll}
}

func (r *manifestResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_manifest"
}

func (r *manifestResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages one Wayfinder object from a YAML or JSON manifest. Use it for kinds without a dedicated resource. Only the fields set in the manifest are managed; secrets cannot be set here.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "Import ID: <group>/<Kind>/[workspace/[environment/]]name[@version], for example org/Workspace/plat."},
			"manifest": schema.StringAttribute{
				Required:    true,
				Description: "YAML or JSON of one Wayfinder object with apiVersion, kind and metadata (name, and workspace, environment or version where the kind needs them). Changing the kind or any of those metadata fields replaces the object. Secret fields must be left blank.",
			},
			"wait_for_ready": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Wait for Wayfinder to finish reconciling the object after each change. Defaults to true."},
			"timeouts":       timeouts.Attributes(ctx, timeouts.Opts{Create: true, Update: true, Delete: true}),
			"object_json":    schema.StringAttribute{Computed: true, CustomType: jsontypes.NormalizedType{}, Description: "The object as read from Wayfinder, limited to the fields set in manifest."},
			"status": schema.SingleNestedAttribute{Computed: true, Description: "Summary of the object's status.", Attributes: map[string]schema.Attribute{
				"status":  schema.StringAttribute{Computed: true},
				"message": schema.StringAttribute{Computed: true},
				"detail":  schema.StringAttribute{Computed: true},
			}},
		},
	}
}

func (r *manifestResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

// ready reports configuration problems that prevent any API call.
func (r *manifestResource) ready(d *diag.Diagnostics, needClient bool) bool {
	if r.manifestErr != nil {
		d.AddError("Provider manifest unavailable", r.manifestErr.Error())
	}
	if needClient && r.client == nil {
		d.AddError("Wayfinder client unavailable", "The provider must be configured before reading or writing resources.")
	}
	return !d.HasError()
}
