// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package manifestresource

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/engine"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testManifest is the golden manifest (Blueprint, Integration, Pipeline (manifest-only),
// Workspace; excluded Run, Session) plus two kinds shaped like real Wayfinder examples:
// Group (doc/public/examples/Group-developers.yaml) and ExternalIdentity with its sensitive
// spec.bearerToken.token.
func testManifest(t *testing.T) *manifest.Manifest {
	t.Helper()
	m, err := manifest.Load("../manifest/testdata/golden-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	for i := range m.Kinds {
		// The historical fixture predates the relative apiPath contract.
		m.Kinds[i].APIPath = strings.TrimPrefix(m.Kinds[i].APIPath, "/api/v3/")
	}
	str := func() *manifest.Field { return &manifest.Field{Type: "string"} }
	subject := &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{"kind": str(), "name": str()}}
	m.Kinds = append(m.Kinds,
		manifest.Kind{Kind: "Group", Group: "org", Version: "v3", APIPath: "groups", Typed: true, Scopes: []string{"workspace"}, HasStatus: true, WaitForReady: true, TerraformName: "group",
			Spec: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
				"description": str(),
				"managers":    {Type: "array", Items: subject},
				"members":     {Type: "array", Items: subject},
			}}},
		manifest.Kind{Kind: "ExternalIdentity", Group: "integrations", Version: "v3", APIPath: "externalidentities", Typed: true, Scopes: []string{"tenant", "workspace"}, HasStatus: true, WaitForReady: true, TerraformName: "external_identity",
			Spec: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
				"type":        str(),
				"bearerToken": {Type: "object", Properties: map[string]*manifest.Field{"token": {Type: "string", Sensitive: true}}},
				"usableFor":   {Type: "array", Items: str()},
			}}},
	)
	return m
}

func findKind(t *testing.T, m *manifest.Manifest, name string) manifest.Kind {
	t.Helper()
	for _, k := range m.Kinds {
		if k.Kind == name {
			return k
		}
	}
	t.Fatalf("kind %s missing", name)
	return manifest.Kind{}
}

type testToken string

func (s testToken) Token(context.Context) (string, error) { return string(s), nil }

type testProvider struct {
	c *client.Client
	m *manifest.Manifest
}

func (*testProvider) Metadata(_ context.Context, _ provider.MetadataRequest, r *provider.MetadataResponse) {
	r.TypeName = "wayfinder"
}
func (*testProvider) Schema(context.Context, provider.SchemaRequest, *provider.SchemaResponse) {}
func (p *testProvider) Configure(_ context.Context, _ provider.ConfigureRequest, r *provider.ConfigureResponse) {
	r.ResourceData = p.c
}

func (p *testProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{func() resource.Resource {
		return newResource(p.m, engine.Poll{Initial: time.Millisecond, Maximum: 2 * time.Millisecond})
	}}
}

func (*testProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }

type testEnv struct {
	fake      *fakeapi.Server
	m         *manifest.Manifest
	factories map[string]func() (tfprotov6.ProviderServer, error)
}

// newTestEnv serves every test-manifest kind from the HTTP fake. specs adjust a kind's fake
// registration (defaults, validation) before it is registered.
func newTestEnv(t *testing.T, specs ...func(*fakeapi.KindSpec)) testEnv {
	t.Helper()
	m := testManifest(t)
	f := fakeapi.New(t)
	for _, k := range m.Kinds {
		s := fakeapi.KindSpec{Kind: k.Kind, Group: k.Group, Version: k.Version, APIPath: k.APIPath, Versioned: k.Versioned, HasStatus: k.HasStatus}
		for _, scope := range k.Scopes {
			s.Scopes = append(s.Scopes, fakeapi.Scope(scope))
		}
		switch k.Kind {
		case "Integration":
			s.SensitiveStrings, s.SensitiveMaps = []string{"spec.apiKey"}, []string{"spec.headers"}
		case "ExternalIdentity":
			s.SensitiveStrings = []string{"spec.bearerToken.token"}
		}
		for _, fn := range specs {
			fn(&s)
		}
		f.RegisterKind(s)
	}
	c := &client.Client{BaseURL: f.URL, Tenant: "acme", TokenSource: testToken(fakeapi.DefaultToken)}
	return testEnv{fake: f, m: m, factories: map[string]func() (tfprotov6.ProviderServer, error){"wayfinder": providerserver.NewProtocol6WithError(&testProvider{c: c, m: m})}}
}

func (e testEnv) ref(t *testing.T, kind, workspace, environment, name string) fakeapi.Ref {
	t.Helper()
	k := findKind(t, e.m, kind)
	return fakeapi.Ref{Tenant: "acme", Workspace: workspace, Environment: environment, Group: k.Group, Version: k.Version, APIPath: k.APIPath, Name: name}
}

// config renders one wayfinder_manifest.this resource holding text as a heredoc.
func config(text string) string {
	return "resource \"wayfinder_manifest\" \"this\" {\n  manifest = <<-EOT\n" + indent(text) + "  EOT\n}\n"
}

func indent(text string) string {
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		b.WriteString("    " + line + "\n")
	}
	return b.String()
}
