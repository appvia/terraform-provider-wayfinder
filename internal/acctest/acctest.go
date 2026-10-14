// Package acctest runs the provider's acceptance tests (TF_ACC=1) against a real Wayfinder API
// and sweeps the objects they leave behind. Every object a test creates is named with
// NamePrefix so the sweepers can find it.
package acctest

import (
	"fmt"
	"os"
	"strings"
	"testing"

	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/appvia/terraform-provider-wayfinder/internal/auth"
	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/provider"
)

// NamePrefix starts the name of every object an acceptance test creates.
const NamePrefix = "tfacc-"

// ProtoV6ProviderFactories serves the real provider in-process.
var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"wayfinder": func() (tfprotov6.ProviderServer, error) {
		return providerserver.NewProtocol6WithError(provider.New("acc")())()
	},
}

// Env is the acceptance-test connection, read from the provider's WAYFINDER_* variables.
type Env struct {
	Server, Tenant, Token, ClientID, ClientSecret string
}

// LoadEnv reads the acceptance environment through getenv (os.Getenv in tests).
func LoadEnv(getenv func(string) string) Env {
	return Env{
		Server:       getenv("WAYFINDER_SERVER"),
		Tenant:       getenv("WAYFINDER_TENANT"),
		Token:        getenv("WAYFINDER_TOKEN"),
		ClientID:     getenv("WAYFINDER_CLIENT_ID"),
		ClientSecret: getenv("WAYFINDER_CLIENT_SECRET"),
	}
}

// Missing lists the variables that must still be set. Client credentials derive the tenant
// from the client ID, so WAYFINDER_TENANT is only required with a token.
func (e Env) Missing() []string {
	var out []string
	if e.Server == "" {
		out = append(out, "WAYFINDER_SERVER")
	}
	clientCredentials := e.ClientID != "" && e.ClientSecret != ""
	switch {
	case e.Token == "" && !clientCredentials:
		out = append(out, "WAYFINDER_TOKEN or WAYFINDER_CLIENT_ID and WAYFINDER_CLIENT_SECRET")
	case e.Token != "" && e.Tenant == "":
		out = append(out, "WAYFINDER_TENANT")
	}
	return out
}

// PreCheck fails the test when the acceptance environment is incomplete, then logs the server
// release next to the Wayfinder version the provider was generated from.
func PreCheck(t *testing.T) {
	t.Helper()
	if missing := LoadEnv(os.Getenv).Missing(); len(missing) > 0 {
		t.Fatalf("acceptance tests need %s (and %s for a custom certificate authority bundle)",
			strings.Join(missing, ", "), provider.CAFileEnv)
	}
	logServerVersion(t)
}

// RandomName returns NamePrefix followed by eight random lowercase letters.
func RandomName() string {
	return NamePrefix + sdkacctest.RandStringFromCharSet(8, sdkacctest.CharSetAlpha)
}

// NewClient authenticates exactly as the provider does from the WAYFINDER_* environment.
func NewClient() (*client.Client, error) {
	httpClient, err := provider.NewHTTPClient(os.Getenv(provider.CAFileEnv))
	if err != nil {
		return nil, err
	}
	resolved, err := auth.Resolve(auth.Config{HTTP: httpClient, CLIHTTP: provider.CLIHTTPClient})
	if err != nil {
		return nil, fmt.Errorf("resolving Wayfinder credentials: %w", err)
	}
	return &client.Client{
		BaseURL: resolved.Server, Tenant: resolved.Tenant, TokenSource: resolved.TokenSource,
		HTTP: resolved.HTTP, UserAgent: client.UserAgent("acc"),
	}, nil
}
