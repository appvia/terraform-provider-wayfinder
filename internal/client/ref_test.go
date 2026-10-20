// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package client_test

import (
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
)

func TestRef_paths_match_R8(t *testing.T) {
	cases := []struct {
		name             string
		ref              client.Ref
		collection, item string
	}{
		{
			name:       "workspace",
			ref:        client.Ref{Group: "org", Version: "v3", APIPath: "environments", Workspace: "plat", Name: "dev"},
			collection: "/api/v3/resources/tenants/acme/workspaces/plat/org/v3/environments",
			item:       "/api/v3/resources/tenants/acme/workspaces/plat/org/v3/environments/dev",
		},
		{
			name:       "tenant",
			ref:        client.Ref{Group: "org", Version: "v3", APIPath: "workspaces", Name: "plat"},
			collection: "/api/v3/resources/tenants/acme/org/v3/workspaces",
			item:       "/api/v3/resources/tenants/acme/org/v3/workspaces/plat",
		},
		{
			name:       "environment",
			ref:        client.Ref{Group: "app.appvia.io", Version: "v3", APIPath: "appenvs", Workspace: "plat", Environment: "dev", Name: "web"},
			collection: "/api/v3/resources/tenants/acme/workspaces/plat/envs/dev/app.appvia.io/v3/appenvs",
			item:       "/api/v3/resources/tenants/acme/workspaces/plat/envs/dev/app.appvia.io/v3/appenvs/web",
		},
		{
			name:       "versioned",
			ref:        client.Ref{Group: "kubeplans.appvia.io", Version: "v3", APIPath: "kuberesourceplans", Name: "redis", ObjectVersion: "1.0.0", Versioned: true},
			collection: "/api/v3/resources/tenants/acme/kubeplans.appvia.io/v3/kuberesourceplans",
			item:       "/api/v3/resources/tenants/acme/kubeplans.appvia.io/v3/kuberesourceplans/redis/versions/1.0.0",
		},
		{
			name:       "escaped name",
			ref:        client.Ref{Group: "org", Version: "v3", APIPath: "workspaces", Name: "a b/c"},
			collection: "/api/v3/resources/tenants/acme/org/v3/workspaces",
			item:       "/api/v3/resources/tenants/acme/org/v3/workspaces/a%20b%2Fc",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			col, err := tc.ref.CollectionPath(tenant)
			if err != nil || col != tc.collection {
				t.Errorf("CollectionPath = %q, %v; want %q", col, err, tc.collection)
			}

			item, err := tc.ref.ItemPath(tenant)
			if err != nil || item != tc.item {
				t.Errorf("ItemPath = %q, %v; want %q", item, err, tc.item)
			}
		})
	}
}

func TestRef_invalid_refs_are_rejected(t *testing.T) {
	cases := map[string]client.Ref{
		"missing group":              {Version: "v3", APIPath: "x", Name: "n"},
		"environment w/o workspace":  {Group: "g", Version: "v3", APIPath: "x", Environment: "dev", Name: "n"},
		"missing name":               {Group: "g", Version: "v3", APIPath: "x"},
		"versioned without version":  {Group: "g", Version: "v3", APIPath: "x", Name: "n", Versioned: true},
		"version on unversioned ref": {Group: "g", Version: "v3", APIPath: "x", Name: "n", ObjectVersion: "1.0.0"},
	}
	for name, ref := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ref.ItemPath(tenant); err == nil {
				t.Errorf("ItemPath(%+v) succeeded; want error", ref)
			}
		})
	}

	if _, err := (client.Ref{Group: "g", Version: "v3", APIPath: "x"}).CollectionPath(""); err == nil {
		t.Error("CollectionPath with empty tenant succeeded; want error")
	}
}
