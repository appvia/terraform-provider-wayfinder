package acctest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

// defaultIgnore lists attributes that import cannot or need not reproduce: provider-only
// settings, server-owned status, which a reconcile may change between two reads, and the
// annotations Wayfinder stamps on every write (appvia.io/createdBy, lastUpdated, ...), which
// import adopts because it populates every field.
var defaultIgnore = []string{"wait_for_ready", "timeouts", "status", "annotations"}

// importStep is one resource to import after the update step.
type importStep struct {
	addr, id string
	ignore   []string
}

// lifecycle is create -> empty plan -> update -> empty plan -> import verify -> destroy.
type lifecycle struct {
	create, update string
	// updated is the resource the update step must change in place.
	updated string
	imports []importStep
	checks  []resource.TestCheckFunc
	// secrets are literals that must never reach the plan, state or the state file.
	secrets []string
}

func emptyAfterApply(pre ...plancheck.PlanCheck) resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{PreApply: pre, PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}
}

func run(t *testing.T, lc lifecycle) {
	t.Helper()
	stateDir := t.TempDir()
	t.Setenv("TF_ACC_TEMP_DIR", stateDir)
	secretPlan := []plancheck.PlanCheck{absentFromPlan(lc.secrets)}
	steps := []resource.TestStep{
		{
			Config: lc.create, ConfigPlanChecks: emptyAfterApply(secretPlan...),
			ConfigStateChecks: []statecheck.StateCheck{absentFromState(lc.secrets)},
			Check:             resource.ComposeTestCheckFunc(append(lc.checks, absentFromStateFile(stateDir, lc.secrets))...),
		},
		{
			Config: lc.update,
			ConfigPlanChecks: emptyAfterApply(append(secretPlan,
				plancheck.ExpectResourceAction(lc.updated, plancheck.ResourceActionUpdate))...),
			ConfigStateChecks: []statecheck.StateCheck{absentFromState(lc.secrets)},
			Check:             absentFromStateFile(stateDir, lc.secrets),
		},
	}
	for _, i := range lc.imports {
		steps = append(steps, resource.TestStep{
			Config: lc.update, ResourceName: i.addr, ImportState: true, ImportStateId: i.id,
			ImportStateVerify: true, ImportStateVerifyIgnore: append(append([]string{}, defaultIgnore...), i.ignore...),
		})
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { PreCheck(t) },
		ProtoV6ProviderFactories: ProtoV6ProviderFactories,
		// Write-only attributes need Terraform 1.11.
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		CheckDestroy:           checkDestroyed,
		Steps:                  steps,
	})
}

// checkDestroyed confirms through the API that every typed object in the final state is gone.
func checkDestroyed(s *terraform.State) error {
	m, err := manifest.LoadEmbedded()
	if err != nil {
		return err
	}
	byType := map[string]manifest.Kind{}
	for _, k := range m.TypedKinds() {
		byType[k.TerraformName] = k
	}
	c, err := NewClient()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		k, ok := byType[rs.Type]
		if !ok || rs.Primary == nil {
			continue
		}
		a := rs.Primary.Attributes
		ref := client.Ref{Group: k.Group, Version: k.Version, APIPath: k.APIPath, Workspace: a["workspace"],
			Environment: a["environment"], Name: a["name"], ObjectVersion: a["version"], Versioned: k.Versioned}
		if _, err := c.Get(context.Background(), ref); !errors.Is(err, client.ErrNotFound) {
			return fmt.Errorf("%s %s still exists after destroy (err=%v)", k.Kind, rs.Primary.ID, err)
		}
	}
	return nil
}

func containsAny(b []byte, secrets []string) error {
	for _, s := range secrets {
		if strings.Contains(string(b), s) {
			return fmt.Errorf("secret literal of length %d found", len(s))
		}
	}
	return nil
}

type secretCheck []string

func absentFromPlan(secrets []string) plancheck.PlanCheck { return secretCheck(secrets) }

func absentFromState(secrets []string) statecheck.StateCheck { return secretCheck(secrets) }

// CheckPlan marshals the values of the plan read via `terraform show -json`: planned values,
// prior state and resource changes. The configuration section is skipped because Terraform
// itself echoes literal configuration expressions there.
func (s secretCheck) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	b, err := json.Marshal([]any{req.Plan.PlannedValues, req.Plan.PriorState, req.Plan.ResourceChanges, req.Plan.OutputChanges})
	if err == nil {
		err = containsAny(b, s)
	}
	resp.Error = err
}

// CheckState marshals the state read via `terraform show -json`.
func (s secretCheck) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	b, err := json.Marshal(req.State)
	if err == nil {
		err = containsAny(b, s)
	}
	resp.Error = err
}

// absentFromStateFile greps every raw terraform.tfstate file under dir (TF_ACC_TEMP_DIR).
func absentFromStateFile(dir string, secrets []string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if len(secrets) == 0 {
			return nil
		}
		found := 0
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasPrefix(d.Name(), "terraform.tfstate") {
				return err
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			found++
			if err := containsAny(b, secrets); err != nil {
				return fmt.Errorf("%s: %w", p, err)
			}
			return nil
		})
		if err == nil && found == 0 {
			err = fmt.Errorf("no terraform.tfstate file under %s", dir)
		}
		return err
	}
}
