// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi/fakekinds"
	"github.com/appvia/terraform-provider-wayfinder/internal/featureintegration"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

const (
	tenant  = "acme"
	address = "wayfinder_feature_integration.example"
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
	return []func() resource.Resource{featureintegration.NewResource}
}

func (p *testProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }

type env struct {
	fake      *fakeapi.Server
	client    *client.Client
	factories map[string]func() (tfprotov6.ProviderServer, error)
}

func newEnv(t *testing.T) env {
	t.Helper()

	m, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}

	fake := fakeapi.New(t)
	found := false

	for _, k := range m.Kinds {
		if k.Kind == "Integration" {
			fake.RegisterKind(fakekinds.Spec(k))

			found = true

			break
		}
	}

	if !found {
		t.Fatal("Integration missing from embedded manifest")
	}

	for _, f := range []fakeapi.Feature{
		{Name: "azure"}, {Name: "ai", BoolParams: []string{"enable_wayfinder"}},
		{Name: "dns", AlwaysEnabled: true}, {Name: "datadog", Custom: true},
		{Name: "needsconf", RequiredParams: []string{"token"}},
	} {
		fake.RegisterFeature(f)
	}

	c := &client.Client{BaseURL: fake.URL, Tenant: tenant, TokenSource: staticToken(fakeapi.DefaultToken)}

	return env{
		fake: fake, client: c,
		factories: map[string]func() (tfprotov6.ProviderServer, error){
			"wayfinder": providerserver.NewProtocol6WithError(&testProvider{c: c}),
		},
	}
}

func featureHCL(name, body string) string {
	return fmt.Sprintf("resource \"wayfinder_feature_integration\" \"example\" {\n name = %q\n%s\n}\n", name, body)
}

func featureRef(name string) fakeapi.Ref {
	return fakeapi.Ref{Tenant: tenant, Group: "integrations", Version: "v3", APIPath: "integrations", Name: "wayfinder." + name}
}

func featureObject(name string, enabled bool, params map[string]any) map[string]any {
	return map[string]any{
		"metadata": map[string]any{"name": "wayfinder." + name},
		"spec":     map[string]any{"type": "feature", "enabled": enabled, "params": params},
	}
}

func expectEnabled(e env, name string, enabled bool) testresource.TestCheckFunc {
	return func(*terraform.State) error {
		obj, ok := e.fake.Object(featureRef(name))
		if !ok {
			return fmt.Errorf("integration wayfinder.%s is absent", name)
		}

		if got := obj["spec"].(map[string]any)["enabled"]; got != enabled {
			return fmt.Errorf("integration enabled = %v, want %v", got, enabled)
		}

		return nil
	}
}
