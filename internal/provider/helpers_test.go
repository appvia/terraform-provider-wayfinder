package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi/fakekinds"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

const testTenant = "acme"

// authEnv lists every environment variable the provider's authentication reads.
var authEnv = []string{
	"WAYFINDER_SERVER", "WAYFINDER_TOKEN", "WAYFINDER_TENANT", "WAYFINDER_CLIENT_ID", "WAYFINDER_CLIENT_SECRET",
	"WAYFINDER_SERVICE_ACCOUNT", "WAYFINDER_FEDERATED_TOKEN_FILE", "WAYFINDER_FEDERATED_TOKEN_REQUEST_URL",
	"WAYFINDER_FEDERATED_TOKEN_REQUEST_TOKEN", "WAYFINDER_FEDERATED_TOKEN_REQUEST_AUDIENCE",
	"ACTIONS_ID_TOKEN_REQUEST_URL", "ACTIONS_ID_TOKEN_REQUEST_TOKEN",
}

// testEnv is the real provider served in-process against a fake API that knows every kind of
// the embedded manifest.
type testEnv struct {
	fake      *fakeapi.Server
	m         *manifest.Manifest
	factories map[string]func() (tfprotov6.ProviderServer, error)
}

func newTestEnv(t *testing.T) testEnv {
	t.Helper()
	for _, k := range authEnv {
		t.Setenv(k, "")
	}
	m, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	f := fakeapi.New(t)
	fakekinds.RegisterAll(f, m)
	return testEnv{fake: f, m: m, factories: map[string]func() (tfprotov6.ProviderServer, error){
		"wayfinder": providerserver.NewProtocol6WithError(New("test")()),
	}}
}

// providerBlock configures the provider against the fake with a static token.
func (e testEnv) providerBlock() string {
	return fmt.Sprintf("provider \"wayfinder\" {\n server = %q\n tenant = %q\n token = %q\n}\n", e.fake.URL, testTenant, fakeapi.DefaultToken)
}

// kind returns the embedded kind whose Terraform type is wayfinder_<suffix>.
func (e testEnv) kind(t *testing.T, suffix string) manifest.Kind {
	t.Helper()
	for _, k := range e.m.Kinds {
		if k.Typed && strings.TrimPrefix(k.TerraformName, "wayfinder_") == suffix {
			return k
		}
	}
	t.Fatalf("embedded manifest has no typed kind wayfinder_%s", suffix)
	return manifest.Kind{}
}

func (e testEnv) ref(k manifest.Kind, workspace, name string) fakeapi.Ref {
	return fakeapi.Ref{Tenant: testTenant, Workspace: workspace, Group: k.Group, Version: k.Version, APIPath: k.APIPath, Name: name}
}
