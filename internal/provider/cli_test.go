package provider

import (
	"bytes"
	"encoding/json"
	"encoding/pem"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

func TestProvider_configure_whenCLIProfileSession(t *testing.T) {
	// Given a fake API served over TLS whose certificate is trusted only through the wf CLI
	// profile's caCertificate, and a profiles file holding a refresh token for it.
	e := newTestEnv(t)
	e.fake.AddRefreshToken("cli-refresh")
	api := httptest.NewTLSServer(e.fake)
	t.Cleanup(api.Close)
	ca := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: api.Certificate().Raw})
	body, err := json.Marshal(map[string]any{
		"current-profile": "other",
		"users":           map[string]any{"alice": map[string]any{"identity": map[string]string{"refresh-token": "cli-refresh"}}},
		"profiles": map[string]any{
			"test":  map[string]string{"user": "alice", "server": "api", "tenant": testTenant},
			"other": map[string]string{"user": "alice", "server": "unused", "tenant": "other"},
		},
		"servers": map[string]any{
			"api":    map[string]string{"server": api.URL, "caCertificate": string(ca)},
			"unused": map[string]string{"server": "http://127.0.0.1:9"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	profiles := filepath.Join(t.TempDir(), "profiles")
	if err := os.WriteFile(profiles, body, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WAYFINDER_CONFIG", profiles)
	t.Setenv("WAYFINDER_USE_CLI", "")
	t.Setenv(CAFileEnv, "")
	// When Terraform applies a workspace and reads it back through a data source, configuring
	// the provider only from the CLI profile.
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{{
		Config: `provider "wayfinder" {
 use_cli = true
 cli_profile = "test"
}
resource "wayfinder_workspace" "example" {
 name = "example"
 description = "Platform"
}
data "wayfinder_workspace" "read" {
 name = wayfinder_workspace.example.name
}
`,
		Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.wayfinder_workspace.read", "id", "example"),
			resource.TestCheckResourceAttr("data.wayfinder_workspace.read", "description", "Platform"),
		),
	}}})
	// Then tokens were minted from the profile's refresh token, API requests reached the TLS
	// server (only trusted via the profile CA), and the profiles file is untouched.
	if n := e.fake.Mints(fakeapi.GrantRefreshToken); n < 1 {
		t.Fatalf("refresh-token mints = %d, want >= 1", n)
	}
	if n := e.fake.CountRequests("GET", "/api/v3/resources/tenants/"+testTenant+"/"); n < 1 {
		t.Fatalf("GET requests to tenant %s = %d, want >= 1", testTenant, n)
	}
	after, err := os.ReadFile(profiles)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, body) {
		t.Fatal("profiles file was modified")
	}
}

func TestProvider_configure_whenNoCredentialsAndCLIDisabled(t *testing.T) {
	for _, tc := range []struct{ name, block string }{
		{name: "attribute", block: "use_cli = false"},
		{name: "environment"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given a provider block with a server and tenant but no credentials.
			e := newTestEnv(t)
			cfg := "provider \"wayfinder\" {\n server = \"" + e.fake.URL + "\"\n tenant = \"" + testTenant + "\"\n " + tc.block + "\n}\n"
			// When Terraform plans, then configuration fails with the original credentials
			// error, without the wf CLI hint.
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{{
				Config:      cfg + `data "wayfinder_workspace" "read" { name = "example" }`,
				ExpectError: regexp.MustCompile(`(?s)set token, service_account.*WAYFINDER_CLIENT_SECRET(\s|$)`),
			}}})
			if n := len(e.fake.Requests()); n != 0 {
				t.Fatalf("requests = %d, want 0", n)
			}
		})
	}
}
