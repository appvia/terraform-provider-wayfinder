package vars_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/appvia/terraform-provider-wayfinder/internal/vars"
)

const tenant = "acme"

type staticToken string

func (s staticToken) Token(context.Context) (string, error) { return string(s), nil }

// testProvider registers only the vars resources and hands them a client for the fake.
type testProvider struct{ c *client.Client }

func (p *testProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "wayfinder"
}

func (p *testProvider) Schema(context.Context, provider.SchemaRequest, *provider.SchemaResponse) {}

func (p *testProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.ResourceData = p.c
	resp.DataSourceData = p.c
}

func (p *testProvider) Resources(context.Context) []func() resource.Resource { return vars.Resources() }

func (p *testProvider) DataSources(context.Context) []func() datasource.DataSource {
	return vars.DataSources()
}

// env is a fake API plus provider factories bound to it.
type env struct {
	fake      *fakeapi.Server
	factories map[string]func() (tfprotov6.ProviderServer, error)
}

func newEnv(t *testing.T) env {
	t.Helper()
	fake := fakeapi.New(t)
	c := &client.Client{BaseURL: fake.URL, Tenant: tenant, TokenSource: staticToken(fakeapi.DefaultToken)}
	return env{
		fake: fake,
		factories: map[string]func() (tfprotov6.ProviderServer, error){
			"wayfinder": providerserver.NewProtocol6WithError(&testProvider{c: c}),
		},
	}
}

// writeOnlyVersions: write-only attributes need Terraform >= 1.11.
var writeOnlyVersions = []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)}

// scopeHCL renders the workspace/environment arguments for a scope.
func scopeHCL(s fakeapi.VarScope) string {
	var b strings.Builder
	if s.Workspace != "" {
		fmt.Fprintf(&b, "  workspace   = %q\n", s.Workspace)
	}
	if s.Environment != "" {
		fmt.Fprintf(&b, "  environment = %q\n", s.Environment)
	}
	return b.String()
}

func scopeID(s fakeapi.VarScope, key string) string {
	parts := []string{}
	if s.Workspace != "" {
		parts = append(parts, s.Workspace)
	}
	if s.Environment != "" {
		parts = append(parts, s.Environment)
	}
	return strings.Join(append(parts, key), "/")
}

func varsBase(s fakeapi.VarScope) string {
	p := "/api/v3/general/tenants/" + s.Tenant
	if s.Workspace != "" {
		p += "/workspaces/" + s.Workspace
	}
	if s.Environment != "" {
		p += "/envs/" + s.Environment
	}
	return p + "/vars"
}

var testScopes = map[string]fakeapi.VarScope{
	"tenant":      {Tenant: tenant},
	"workspace":   {Tenant: tenant, Workspace: "ws1"},
	"environment": {Tenant: tenant, Workspace: "ws1", Environment: "dev"},
}

// deleteDirect removes a key through the fake's HTTP API (out-of-band change).
func deleteDirect(t *testing.T, fake *fakeapi.Server, path string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, fake.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+fakeapi.DefaultToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
}

// absentFromState fails when literal appears anywhere in the JSON state.
type absentFromState string

func (a absentFromState) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	b, err := json.Marshal(req.State)
	if err != nil {
		resp.Error = err
		return
	}
	if strings.Contains(string(b), string(a)) {
		resp.Error = fmt.Errorf("secret literal found in state JSON")
	}
}

// absentFromPlan fails when literal appears anywhere in the JSON plan.
type absentFromPlan string

func (a absentFromPlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	b, err := json.Marshal(req.Plan)
	if err != nil {
		resp.Error = err
		return
	}
	if strings.Contains(string(b), string(a)) {
		resp.Error = fmt.Errorf("secret literal found in plan JSON")
	}
}

// jsonEqual compares two JSON documents semantically.
func jsonEqual(got json.RawMessage, want string) bool {
	var a, b any
	if json.Unmarshal(got, &a) != nil || json.Unmarshal([]byte(want), &b) != nil {
		return false
	}
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}
