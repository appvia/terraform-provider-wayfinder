// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"strings"
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

func TestAuthConfig_whenUseCLIAndProfileSet(t *testing.T) {
	for _, tc := range []struct {
		name   string
		useCLI types.Bool
		want   *bool
	}{
		{name: "null", useCLI: types.BoolNull()},
		{name: "true", useCLI: types.BoolValue(true), want: ptr(true)},
		{name: "false", useCLI: types.BoolValue(false), want: ptr(false)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given a provider block with use_cli and cli_profile.
			m := providerModel{UseCLI: tc.useCLI, CLIProfile: types.StringValue("work")}
			// When mapping it to auth configuration.
			cfg, d := authConfig(m)
			// Then null stays unset and a value is passed through as a pointer.
			if d.HasError() || cfg.CLIProfile != "work" {
				t.Fatalf("cfg = %+v, diagnostics = %v", cfg, d)
			}

			if (cfg.UseCLI == nil) != (tc.want == nil) || (cfg.UseCLI != nil && *cfg.UseCLI != *tc.want) {
				t.Fatalf("UseCLI = %v, want %v", cfg.UseCLI, tc.want)
			}
		})
	}
}

func TestAuthConfig_whenUseCLIUnknown(t *testing.T) {
	// Given a provider block whose use_cli is only known at apply.
	m := providerModel{UseCLI: types.BoolUnknown()}
	// When mapping it to auth configuration.
	_, d := authConfig(m)
	// Then exactly one error names the unknown attribute.
	if d.ErrorsCount() != 1 || d.Errors()[0].Summary() != "Unknown Wayfinder provider attribute" ||
		!strings.Contains(d.Errors()[0].Detail(), "use_cli") {
		t.Fatalf("diagnostics = %v", d)
	}
}

func ptr[T any](v T) *T { return &v }

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
