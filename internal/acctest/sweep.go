// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package acctest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

// sweptKinds are the kinds the acceptance tests create, in the order they must be deleted:
// Workspace last, because a workspace refuses deletion while it still holds objects.
var sweptKinds = []string{"ExternalIdentity", "KubeResourcePlan", "RoleBinding", "ServiceAccount", "Group", "Environment", "Webhook", "Integration", "Workspace"}

// sweepVarsName is the sweeper for wayfinder_variable and wayfinder_secret keys.
const sweepVarsName = "wayfinder_vars"

// AddSweepers registers one sweeper per tested kind plus one for variables and secrets.
func AddSweepers() {
	m, err := manifest.LoadEmbedded()
	if err != nil {
		panic(fmt.Sprintf("loading embedded manifest: %v", err))
	}

	byKind := map[string]manifest.Kind{}
	for _, k := range m.Kinds {
		byKind[k.Kind] = k
	}

	resource.AddTestSweepers(sweepVarsName, &resource.Sweeper{Name: sweepVarsName, F: func(string) error {
		c, err := NewClient()
		if err != nil {
			return err
		}

		return sweepVars(context.Background(), c)
	}})

	var before []string

	for _, name := range sweptKinds {
		k := byKind[name]

		deps := append([]string{sweepVarsName}, before...)
		resource.AddTestSweepers(k.TerraformName, &resource.Sweeper{Name: k.TerraformName, Dependencies: deps, F: func(string) error {
			c, err := NewClient()
			if err != nil {
				return err
			}

			return sweepKind(context.Background(), c, k)
		}})
		before = append(before, k.TerraformName)
	}
}

// listItems is the {items: [...]} body of a collection GET.
type listItems struct {
	Items []struct {
		Metadata struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"metadata"`
	} `json:"items"`
}

func list(ctx context.Context, c *client.Client, ref client.Ref) (listItems, error) {
	var out listItems

	p, err := ref.CollectionPath(c.Tenant)
	if err != nil {
		return out, err
	}

	err = c.DoJSON(ctx, http.MethodGet, p, nil, &out)
	if errors.Is(err, client.ErrNotFound) {
		return listItems{}, nil
	}

	return out, err
}

// testWorkspaces lists the names of the tenant's workspaces created by acceptance tests.
func testWorkspaces(ctx context.Context, c *client.Client) ([]string, error) {
	m, err := manifest.LoadEmbedded()
	if err != nil {
		return nil, err
	}

	workspace, ok := m.FindKind("org", "Workspace")
	if !ok {
		return nil, errors.New("embedded manifest has no Workspace kind")
	}

	items, err := list(ctx, c, client.Ref{Group: workspace.Group, Version: workspace.Version, APIPath: workspace.APIPath})
	if err != nil {
		return nil, fmt.Errorf("listing workspaces: %w", err)
	}

	var out []string

	for _, it := range items.Items {
		if strings.HasPrefix(it.Metadata.Name, NamePrefix) {
			out = append(out, it.Metadata.Name)
		}
	}

	return out, nil
}

func hasScope(k manifest.Kind, scope string) bool {
	for _, s := range k.Scopes {
		if s == scope {
			return true
		}
	}

	return false
}

// sweepKind deletes every NamePrefix object of k at tenant scope and inside test workspaces.
func sweepKind(ctx context.Context, c *client.Client, k manifest.Kind) error {
	var workspaces []string
	if hasScope(k, "tenant") {
		workspaces = append(workspaces, "")
	}

	if hasScope(k, "workspace") {
		ws, err := testWorkspaces(ctx, c)
		if err != nil {
			return err
		}

		workspaces = append(workspaces, ws...)
	}

	var errs []error

	for _, ws := range workspaces {
		col := client.Ref{Group: k.Group, Version: k.Version, APIPath: k.APIPath, Workspace: ws, Versioned: k.Versioned}

		items, err := list(ctx, c, col)
		if err != nil {
			errs = append(errs, fmt.Errorf("listing %s in %q: %w", k.Kind, ws, err))
			continue
		}

		for _, it := range items.Items {
			if !strings.HasPrefix(it.Metadata.Name, NamePrefix) {
				continue
			}

			ref := col

			ref.Name, ref.ObjectVersion = it.Metadata.Name, it.Metadata.Version
			if err := c.Delete(ctx, ref); err != nil && !errors.Is(err, client.ErrNotFound) {
				errs = append(errs, fmt.Errorf("deleting %s %s/%s: %w", k.Kind, ws, it.Metadata.Name, err))
			}
		}
	}

	return errors.Join(errs...)
}

// varList is the VarSetSpec body of GET .../vars (secret values are never returned).
type varList struct {
	Secrets   []struct{ Name string } `json:"secrets"`
	Variables []struct{ Name string } `json:"variables"`
}

// sweepVars deletes NamePrefix variables and secrets at tenant, workspace and environment
// scope within test workspaces.
//
// Known limitation: instance-scope stores (.../envs/{e}/instancevars/{i}) cannot be listed
// through the vars API, so keys abandoned there by an aborted run are NOT swept. The acceptance
// tests delete them on destroy and verify that with checkVarsDestroyed.
func sweepVars(ctx context.Context, c *client.Client) error {
	ws, err := testWorkspaces(ctx, c)
	if err != nil {
		return err
	}

	m, err := manifest.LoadEmbedded()
	if err != nil {
		return err
	}

	environment, ok := m.FindKind("org", "Environment")
	if !ok {
		return errors.New("embedded manifest has no Environment kind")
	}

	tenant := "/api/v3/general/tenants/" + url.PathEscape(c.Tenant)
	bases := []string{tenant}

	var errs []error

	for _, w := range ws {
		wsBase := tenant + "/workspaces/" + url.PathEscape(w)
		bases = append(bases, wsBase)

		envs, err := list(ctx, c, client.Ref{Group: environment.Group, Version: environment.Version, APIPath: environment.APIPath, Workspace: w})
		if err != nil {
			errs = append(errs, fmt.Errorf("listing environments in %q: %w", w, err))
			continue
		}

		for _, e := range envs.Items {
			bases = append(bases, wsBase+"/envs/"+url.PathEscape(e.Metadata.Name))
		}
	}

	for _, base := range bases {
		var vl varList
		if err := c.DoJSON(ctx, http.MethodGet, base+"/vars", nil, &vl); err != nil {
			if !errors.Is(err, client.ErrNotFound) {
				errs = append(errs, fmt.Errorf("listing vars at %q: %w", base, err))
			}

			continue
		}

		for kind, entries := range map[string][]struct{ Name string }{"secrets": vl.Secrets, "variables": vl.Variables} {
			for _, e := range entries {
				if !strings.HasPrefix(e.Name, NamePrefix) {
					continue
				}

				err := c.DoJSON(ctx, http.MethodDelete, base+"/vars/"+kind+"/"+url.PathEscape(e.Name), nil, nil)
				if err != nil && !errors.Is(err, client.ErrNotFound) {
					errs = append(errs, fmt.Errorf("deleting %s %s at %q: %w", kind, e.Name, base, err))
				}
			}
		}
	}

	return errors.Join(errs...)
}
