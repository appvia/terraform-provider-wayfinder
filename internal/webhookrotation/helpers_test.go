// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package webhookrotation_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi/fakekinds"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/appvia/terraform-provider-wayfinder/internal/webhookrotation"
)

const (
	tenant  = "acme"
	address = "wayfinder_webhook_rotation.test"
	// leakCanary is the URL the fake rotate response returns; it must never surface anywhere.
	leakCanary = "https://fake.invalid/ingest/LEAKCANARY"
	canaryWord = "LEAKCANARY"
)

type staticToken string

func (s staticToken) Token(context.Context) (string, error) { return string(s), nil }

type testProvider struct{ c *client.Client }

func (p *testProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "wayfinder"
}

func (p *testProvider) Schema(context.Context, provider.SchemaRequest, *provider.SchemaResponse) {}

func (p *testProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.ResourceData = p.c
}

func (p *testProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{webhookrotation.NewResource}
}

func (p *testProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }

type model struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Webhook    types.String `tfsdk:"webhook"`
	Workspace  types.String `tfsdk:"workspace"`
	Triggers   types.Map    `tfsdk:"triggers"`
	RotationID types.String `tfsdk:"rotation_id"`
	RotatedAt  types.String `tfsdk:"rotated_at"`
}

type env struct {
	fake      *fakeapi.Server
	client    *client.Client
	webhook   manifest.Kind
	factories map[string]func() (tfprotov6.ProviderServer, error)
}

func newEnv(t *testing.T) env {
	t.Helper()

	embedded, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}

	k, ok := embedded.FindKind("integrations", "Webhook")
	if !ok {
		t.Fatal("embedded Webhook missing")
	}

	fake := fakeapi.New(t)
	fakekinds.RegisterAll(fake, embedded)
	c := &client.Client{BaseURL: fake.URL, Tenant: tenant, TokenSource: staticToken(fakeapi.DefaultToken)}

	return env{
		fake: fake, client: c, webhook: *k,
		factories: map[string]func() (tfprotov6.ProviderServer, error){
			"wayfinder": providerserver.NewProtocol6WithError(&testProvider{c: c}),
		},
	}
}

func (e env) ref(workspace, name string) client.Ref {
	return client.Ref{
		Group: e.webhook.Group, Version: e.webhook.Version, APIPath: e.webhook.APIPath,
		Versioned: e.webhook.Versioned, Workspace: workspace, Name: name,
	}
}

func (e env) item(t *testing.T, workspace, name string) string {
	t.Helper()

	item, err := e.ref(workspace, name).ItemPath(tenant)
	if err != nil {
		t.Fatal(err)
	}

	return item
}

func (e env) rotatePath(t *testing.T, workspace, name string) string {
	t.Helper()

	return e.item(t, workspace, name) + "/rotate"
}

func (e env) answerRotate(workspace, name string, status int, body any) {
	e.fake.SetSubresourceMethod(e.ref(workspace, name), http.MethodPost, "rotate", status, body)
}

func (e env) allowRotate(workspace, name string) {
	e.answerRotate(workspace, name, http.StatusOK, map[string]string{"url": leakCanary})
}

func (e env) seedWebhook(workspace, name string) {
	e.fake.Seed(fakeapi.Ref{
		Tenant: tenant, Workspace: workspace,
		Group: e.webhook.Group, Version: e.webhook.Version, APIPath: e.webhook.APIPath, Name: name,
	}, map[string]any{"spec": map[string]any{}})
}

func (e env) resource(t *testing.T) (resource.Resource, schema.Schema) {
	t.Helper()

	r := webhookrotation.NewResource()

	var configured resource.ConfigureResponse
	r.(resource.ResourceWithConfigure).Configure(t.Context(), resource.ConfigureRequest{ProviderData: e.client}, &configured)

	if configured.Diagnostics.HasError() {
		t.Fatal(configured.Diagnostics)
	}

	var s resource.SchemaResponse
	r.Schema(t.Context(), resource.SchemaRequest{}, &s)

	return r, s.Schema
}

func nullState(s schema.Schema) tfsdk.State {
	return tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
}

func stateOf(t *testing.T, s schema.Schema, m model) tfsdk.State {
	t.Helper()

	st := nullState(s)
	if diags := st.Set(t.Context(), &m); diags.HasError() {
		t.Fatal(diags)
	}

	return st
}

func optional(value string) types.String {
	if value == "" {
		return types.StringNull()
	}

	return types.StringValue(value)
}

// create calls Create with the plan Terraform sends for a new resource: computed values unknown.
func create(t *testing.T, r resource.Resource, s schema.Schema, workspace, name string) resource.CreateResponse {
	t.Helper()

	plan := tfsdk.Plan{Schema: s, Raw: nullState(s).Raw}
	m := model{
		ID: types.StringUnknown(), Name: types.StringUnknown(),
		Webhook: types.StringValue(name), Workspace: optional(workspace),
		Triggers:   types.MapValueMust(types.StringType, map[string]attr.Value{"n": types.StringValue("1")}),
		RotationID: types.StringUnknown(), RotatedAt: types.StringUnknown(),
	}

	if diags := plan.Set(t.Context(), &m); diags.HasError() {
		t.Fatal(diags)
	}

	resp := resource.CreateResponse{State: nullState(s)}
	r.Create(t.Context(), resource.CreateRequest{Plan: plan}, &resp)

	return resp
}

func stateModel(t *testing.T, st tfsdk.State) model {
	t.Helper()

	var m model
	if diags := st.Get(t.Context(), &m); diags.HasError() {
		t.Fatal(diags)
	}

	return m
}

func diagText(d diag.Diagnostics) string {
	var b strings.Builder
	for _, item := range d {
		fmt.Fprintf(&b, "%s: %s\n", item.Summary(), item.Detail())
	}

	return b.String()
}

func rotationHCL(triggers string) string {
	return fmt.Sprintf("resource \"wayfinder_webhook_rotation\" \"test\" {\n  webhook  = \"hook\"\n  triggers = { n = %q }\n}\n", triggers)
}
