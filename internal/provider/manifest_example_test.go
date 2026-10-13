package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestManifestExample_yamlencode(t *testing.T) {
	// Given: the documented wayfinder_manifest example, read from disk so docs and tests cannot drift.
	example, err := os.ReadFile("../../examples/resources/wayfinder_manifest/resource.tf")
	if err != nil {
		t.Fatal(err)
	}
	e := newTestEnv(t)

	// When: applying it against the fake API.
	// Then: it applies, the object exists, and the follow-up plan is empty.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: e.factories,
		Steps: []resource.TestStep{{
			Config:           e.providerBlock() + string(example),
			ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			Check:            resource.TestCheckResourceAttrSet("wayfinder_manifest.developers", "id"),
		}},
	})
}
