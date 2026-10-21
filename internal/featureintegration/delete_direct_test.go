// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDelete_whenRetainedStateSkipsAPI(t *testing.T) {
	// Given retained state for an enabled feature.
	e := newEnv(t)
	e.fake.Seed(featureRef("azure"), featureObject("azure", true, nil))

	r, state := lifecycleResource(t, e, "azure")
	if diags := state.SetAttribute(t.Context(), path.Root("keep_enabled_on_destroy"), types.BoolValue(true)); diags.HasError() {
		t.Fatal(diags)
	}

	var resp resource.DeleteResponse
	// When deleting from that state.
	r.Delete(t.Context(), resource.DeleteRequest{State: state}, &resp)
	// Then no API call is made and the enabled object is untouched.
	if resp.Diagnostics.HasError() || len(e.fake.Requests()) != 0 {
		t.Fatalf("diagnostics = %v, requests = %+v", resp.Diagnostics, e.fake.Requests())
	}

	if err := expectEnabled(e, "azure", true)(nil); err != nil {
		t.Fatal(err)
	}
}

func TestDelete_whenAlwaysEnabled(t *testing.T) {
	t.Run("warning", func(t *testing.T) {
		// Given an always-enabled feature in Terraform state.
		e := newEnv(t)
		e.fake.Seed(featureRef("dns"), featureObject("dns", true, nil))
		r, state := lifecycleResource(t, e, "dns")

		var resp resource.DeleteResponse
		// When requesting deletion with the default retention flag.
		r.Delete(t.Context(), resource.DeleteRequest{State: state}, &resp)
		// Then it succeeds with a warning and never writes the object.
		warnings := resp.Diagnostics.Warnings()
		if resp.Diagnostics.HasError() || len(warnings) != 1 {
			t.Fatalf("diagnostics = %v, want one warning", resp.Diagnostics)
		}

		if !strings.Contains(warnings[0].Detail(), `"dns"`) || !strings.Contains(warnings[0].Detail(), "always enabled") {
			t.Fatalf("warning = %v", warnings[0])
		}

		if err := expectEnabled(e, "dns", true)(nil); err != nil {
			t.Fatal(err)
		}

		if err := expectWrites(e, 0, 0)(nil); err != nil {
			t.Fatal(err)
		}

		if got := e.fake.CountRequests(http.MethodGet, ""); got != 1 {
			t.Fatalf("GET requests = %d, want only the effective-feature lookup", got)
		}
	})

	t.Run("terraform_destroy", func(t *testing.T) {
		// Given an always-enabled feature managed by Terraform.
		e := newEnv(t)
		// When applying and destroying it through the CLI.
		testresource.UnitTest(t, testresource.TestCase{
			ProtoV6ProviderFactories: e.factories,
			CheckDestroy:             expectEnabled(e, "dns", true),
			Steps:                    []testresource.TestStep{{Config: featureHCL("dns", "")}},
		})
		// Then Terraform drops state while the object remains enabled without any PUT.
		if err := expectWrites(e, 1, 0)(nil); err != nil {
			t.Fatal(err)
		}
	})
}

func TestDelete_whenAlreadyDisabled(t *testing.T) {
	// Given stale Terraform state for a feature that is already disabled.
	e := newEnv(t)
	e.fake.Seed(featureRef("azure"), featureObject("azure", false, nil))
	r, state := lifecycleResource(t, e, "azure")

	var resp resource.DeleteResponse
	// When destroying.
	r.Delete(t.Context(), resource.DeleteRequest{State: state}, &resp)
	// Then the disabled object is retained without a redundant PUT.
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	if err := expectEnabled(e, "azure", false)(nil); err != nil {
		t.Fatal(err)
	}

	if err := expectWrites(e, 0, 0)(nil); err != nil {
		t.Fatal(err)
	}

	if got := e.fake.CountRequests(http.MethodGet, ""); got != 2 {
		t.Fatalf("GET requests = %d, want effective lookup and current object", got)
	}
}

func TestDelete_whenFeatureMissing(t *testing.T) {
	for _, name := range []string{"unknown", "azure"} {
		t.Run(name, func(t *testing.T) {
			// Given a missing catalogue entry or a known feature without an object.
			e := newEnv(t)
			r, state := lifecycleResource(t, e, name)

			var resp resource.DeleteResponse
			// When destroying.
			r.Delete(t.Context(), resource.DeleteRequest{State: state}, &resp)
			// Then a 404 from either lookup is treated as already gone.
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}

			if err := expectWrites(e, 0, 0)(nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}
