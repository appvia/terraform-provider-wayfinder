package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestProvider_update_whenManagedValuesRemoved(t *testing.T) {
	for _, tc := range []struct {
		kind, identity, before, after string
		check                         func(map[string]any) error
	}{
		{
			kind: "workspace", before: `description = "first"
 labels = {a = "1", b = "2"}
 annotations = {a = "1", b = "2"}`, after: `labels = {a = "1"}
 annotations = {a = "1"}`,
			check: func(obj map[string]any) error {
				spec, err := nestedMap(obj, "spec")
				if err != nil {
					return err
				}
				if _, exists := spec["description"]; exists {
					return fmt.Errorf("removed description still stored")
				}
				for _, name := range []string{"labels", "annotations"} {
					keys, err := nestedMap(obj, "metadata", name)
					if err != nil {
						return err
					}
					if _, exists := keys["b"]; exists {
						return fmt.Errorf("removed %s key b still stored", name)
					}
					if keys["a"] != "1" {
						return fmt.Errorf("retained %s key a lost", name)
					}
				}
				labels, err := nestedMap(obj, "metadata", "labels")
				if err != nil {
					return err
				}
				if labels["wayfinder.appvia.io/managed-by-fake"] != "true" {
					return fmt.Errorf("server-added label lost")
				}
				return nil
			},
		},
		{
			kind: "kube_resource_plan", identity: `version = "1.0.0"`,
			before: `helm = {chart_name = "chart", release_name = "removed"}
 inputs = [{name = "input", ref_type = {kind = "Workspace", filter = {a = "1", b = "2"}}}]`,
			after: `helm = {chart_name = "chart"}
 inputs = [{name = "input", ref_type = {kind = "Workspace", filter = {a = "1"}}}]`,
			check: func(obj map[string]any) error {
				helm, err := nestedMap(obj, "spec", "helm")
				if err != nil {
					return err
				}
				if _, exists := helm["releaseName"]; exists {
					return fmt.Errorf("removed nested releaseName still stored")
				}
				inputs, err := nestedSlice(obj, "spec", "inputs")
				if err != nil {
					return err
				}
				if len(inputs) == 0 {
					return fmt.Errorf("spec.inputs is empty")
				}
				input, ok := inputs[0].(map[string]any)
				if !ok {
					return fmt.Errorf("spec.inputs[0]: want map, got %T", inputs[0])
				}
				filter, err := nestedMap(input, "refType", "filter")
				if err != nil {
					return err
				}
				if len(filter) != 1 || filter["a"] != "1" {
					return fmt.Errorf("removed nested map entry still stored: %#v", filter)
				}
				return nil
			},
		},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			// Given: a real provider against the HTTP fake, with configured fields.
			e := newTestEnv(t)
			k := e.kind(t, tc.kind)
			ref := e.ref(k, "", "example")
			if k.Versioned {
				ref.ObjectVersion = "1.0.0"
			}
			cfg := func(fields string) string {
				return e.providerBlock() + fmt.Sprintf("resource \"wayfinder_%s\" \"example\" {\n name = \"example\"\n %s\n %s\n}\n", tc.kind, tc.identity, fields)
			}
			addr := "wayfinder_" + tc.kind + ".example"
			// When: applying a configuration that removes previously managed values.
			resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: e.factories, Steps: []resource.TestStep{
				{Config: cfg(tc.before)},
				{Config: cfg(tc.after), ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				}, Check: func(_ *terraform.State) error {
					// Then: inspect stored API data, not just drift-filtered Terraform state.
					obj, exists := e.fake.Object(ref)
					if !exists {
						return fmt.Errorf("updated object missing")
					}
					return tc.check(obj)
				}},
				{Config: cfg(tc.after), PlanOnly: true},
			}})
		})
	}
}
