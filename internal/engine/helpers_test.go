package engine

import (
	"context"
	"testing"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

type testToken string

func (s testToken) Token(context.Context) (string, error) { return string(s), nil }

type testProvider struct {
	c     *client.Client
	kinds []manifest.Kind
}

func (p *testProvider) Metadata(_ context.Context, _ provider.MetadataRequest, r *provider.MetadataResponse) {
	r.TypeName = "wayfinder"
}
func (*testProvider) Schema(context.Context, provider.SchemaRequest, *provider.SchemaResponse) {}
func (p *testProvider) Configure(_ context.Context, _ provider.ConfigureRequest, r *provider.ConfigureResponse) {
	r.ResourceData, r.DataSourceData = p.c, p.c
}

func (p *testProvider) Resources(context.Context) []func() resource.Resource {
	var out []func() resource.Resource
	for _, k := range p.kinds {
		out = append(out, func() resource.Resource {
			return newResource(k, testSchema(k), pollOptions{initial: time.Millisecond, maximum: 2 * time.Millisecond})
		})
	}
	return out
}

func (p *testProvider) DataSources(context.Context) []func() datasource.DataSource {
	var out []func() datasource.DataSource
	for _, k := range p.kinds {
		out = append(out, func() datasource.DataSource { return NewDataSource(k, testDataSchema(k)) })
	}
	return out
}

type testEnv struct {
	fake      *fakeapi.Server
	c         *client.Client
	k         manifest.Kind
	factories map[string]func() (tfprotov6.ProviderServer, error)
}

func newTestEnv(t *testing.T, name string) testEnv {
	t.Helper()
	k := goldenKind(t, name)
	f := fakeapi.New(t)
	f.RecordBodies(true)
	f.RegisterKind(kindSpec(k))
	c := &client.Client{BaseURL: f.URL, Tenant: "acme", TokenSource: testToken(fakeapi.DefaultToken)}
	return testEnv{fake: f, c: c, k: k, factories: map[string]func() (tfprotov6.ProviderServer, error){"wayfinder": providerserver.NewProtocol6WithError(&testProvider{c: c, kinds: []manifest.Kind{k}})}}
}

func kindSpec(k manifest.Kind) fakeapi.KindSpec {
	s := fakeapi.KindSpec{Kind: k.Kind, Group: k.Group, Version: k.Version, APIPath: k.APIPath, Versioned: k.Versioned, HasStatus: k.HasStatus}
	for _, scope := range k.Scopes {
		s.Scopes = append(s.Scopes, fakeapi.Scope(scope))
	}
	if k.Kind == "Integration" {
		s.SensitiveStrings = []string{"spec.apiKey"}
		s.SensitiveMaps = []string{"spec.headers"}
	}
	return s
}

func (e testEnv) ref(workspace, environment, version string) fakeapi.Ref {
	return fakeapi.Ref{Tenant: "acme", Workspace: workspace, Environment: environment, Group: e.k.Group, Version: e.k.Version, APIPath: e.k.APIPath, Name: "example", ObjectVersion: version}
}
