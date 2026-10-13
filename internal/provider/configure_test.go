package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAuthConfig_whenAttributeUnknown(t *testing.T) {
	// Given a provider block whose token is only known at apply.
	m := providerModel{Server: types.StringValue("https://example.com"), Token: types.StringUnknown()}
	// When mapping it to auth configuration.
	_, d := authConfig(m)
	// Then exactly one error names the unknown attribute.
	if d.ErrorsCount() != 1 || d.Errors()[0].Summary() != "Unknown Wayfinder provider attribute" {
		t.Fatalf("diagnostics = %v", d)
	}
}

func TestAuthConfig_whenNullFallsBackToEnvironment(t *testing.T) {
	// Given a provider block that sets only the tenant.
	m := providerModel{Tenant: types.StringValue("acme"), Token: types.StringNull()}
	// When mapping it to auth configuration.
	cfg, d := authConfig(m)
	// Then null attributes are empty, so auth falls back to WAYFINDER_* variables.
	if d.HasError() || cfg.Tenant != "acme" || cfg.Token != "" || cfg.Server != "" {
		t.Fatalf("cfg = %+v, diagnostics = %v", cfg, d)
	}
}
