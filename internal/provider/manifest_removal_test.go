// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestProvider_manifestUpdate_whenOnlyManagedKeysRemoved(t *testing.T) {
	const (
		initial      = "apiVersion: org/v3\nkind: Workspace\nmetadata:\n  name: example\n  labels: {a: '1', b: '2'}\nspec: {description: first}\n"
		labelRemoved = "apiVersion: org/v3\nkind: Workspace\nmetadata:\n  name: example\n  labels: {a: '1'}\nspec: {description: first}\n"
		bothRemoved  = "apiVersion: org/v3\nkind: Workspace\nmetadata:\n  name: example\n  labels: {a: '1'}\nspec: {}\n"
		specRemoved  = "apiVersion: org/v3\nkind: Workspace\nmetadata:\n  name: example\n  labels: {a: '1', b: '2'}\nspec: {}\n"
	)
	for _, tc := range []struct {
		name      string
		manifests []string
	}{
		{"label then spec", []string{labelRemoved, bothRemoved}},
		{"spec only", []string{specRemoved}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given: an applied manifest and a server-added label never managed by it.
			e := newTestEnv(t)
			ref := e.ref(e.kind(t, "workspace"), "", "example")
			cfg := func(text string) string {
				return e.providerBlock() + fmt.Sprintf("resource \"wayfinder_manifest\" \"example\" {\n manifest = %q\n}\n", text)
			}
			steps := []resource.TestStep{{Config: cfg(initial)}}
			for _, text := range tc.manifests {
				// When: applying an edit that only removes a managed key.
				steps = append(steps, resource.TestStep{
					PreConfig: e.fake.ResetRequests,
					Config:    cfg(text),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction("wayfinder_manifest.example", plancheck.ResourceActionUpdate)},
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
					Check: func(_ *terraform.State) error {
						// Then: the server actually deleted the key, with one PUT and no drift.
						obj, exists := e.fake.Object(ref)
						if !exists {
							return fmt.Errorf("updated Workspace missing")
						}
						labels, err := nestedMap(obj, "metadata", "labels")
						if err != nil {
							return err
						}
						spec, err := nestedMap(obj, "spec")
						if err != nil {
							return err
						}
						_, hasB := labels["b"]
						_, hasDescription := spec["description"]
						puts := e.fake.CountRequests(http.MethodPut, "")
						if hasB != (text == specRemoved) || hasDescription != (text == labelRemoved) || puts != 1 {
							return fmt.Errorf("removal-only edit: label b present=%t, description present=%t, PUT count=%d; want label b=%t, description=%t, PUT=1", hasB, hasDescription, puts, text == specRemoved, text == labelRemoved)
						}
						if labels["a"] != "1" || labels[fakeapi.ManagedByLabel] != "true" {
							return fmt.Errorf("retained or server-added label lost: %#v", labels)
						}
						t.Logf("removal-only edit reached server: PUT=1, removed keys absent, server-added label preserved")
						return nil
					},
				})
			}
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: steps})
		})
	}
}
