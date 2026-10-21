// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

func TestDelete_whenDefault(t *testing.T) {
	// Given an enabled feature with the default destroy behavior.
	e := newEnv(t)
	config := featureHCL("azure", "")
	// When destroying through Terraform.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		CheckDestroy:             expectEnabled(e, "azure", false),
		Steps: []resource.TestStep{
			{Config: config},
			{Config: config, Destroy: true},
		},
	})
	// Then the object still exists, disabled, and no DELETE was ever sent.
	if err := expectWrites(e, 1, 1)(nil); err != nil {
		t.Fatal(err)
	}

	t.Logf("default destroy: object retained disabled; request log has one PUT and zero DELETEs (%d requests)", len(e.fake.Requests()))
}

func TestDelete_whenKeepEnabledOnDestroy(t *testing.T) {
	// Given an enabled feature with the default retention flag.
	e := newEnv(t)
	keep := featureHCL("azure", "keep_enabled_on_destroy = true")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		CheckDestroy:             expectEnabled(e, "azure", true),
		Steps: []resource.TestStep{
			{Config: featureHCL("azure", "")},
			{
				// When applying retention before the destroy.
				PreConfig: e.fake.ResetRequests,
				Config:    keep,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "keep_enabled_on_destroy", "true"),
					expectWrites(e, 0, 0),
				),
			},
			{PreConfig: e.fake.ResetRequests, Config: keep, Destroy: true},
		},
	})
	// Then destroy leaves the feature enabled without any write request.
	if err := expectWrites(e, 0, 0)(nil); err != nil {
		t.Fatal(err)
	}
}

func TestDelete_whenDependentsBlock(t *testing.T) {
	// Given a feature with dependents that prevent disabling.
	e := newEnv(t)
	e.fake.RegisterFeature(fakeapi.Feature{Name: "azure", BlockDisable: true})

	config := featureHCL("azure", "")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		CheckDestroy:             expectEnabled(e, "azure", true),
		Steps: []resource.TestStep{
			{Config: config},
			{
				// When destroying normally.
				PreConfig:   e.fake.ResetRequests,
				Config:      config,
				Destroy:     true,
				ExpectError: regexp.MustCompile(`(?s)cannot be disabled while\s+resources depend on it`),
			},
			{
				// Then the server refusal blocks destroy; applying retention permits removal.
				PreConfig: func() {
					if err := expectEnabled(e, "azure", true)(nil); err != nil {
						t.Fatal(err)
					}

					if err := expectWrites(e, 0, 1)(nil); err != nil {
						t.Fatal(err)
					}

					e.fake.ResetRequests()
				},
				Config: featureHCL("azure", "keep_enabled_on_destroy = true"),
				Check:  expectWrites(e, 0, 0),
			},
		},
	})
}
