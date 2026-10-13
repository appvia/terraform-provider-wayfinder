package vars

import (
	"testing"
)

func TestScopeID_round_trips_for_every_scope(t *testing.T) {
	cases := map[string]scopedKey{
		"region":         {Key: "region"},
		"ws1/region":     {Workspace: "ws1", Key: "region"},
		"ws1/dev/region": {Workspace: "ws1", Environment: "dev", Key: "region"},
	}
	for id, want := range cases {
		t.Run(id, func(t *testing.T) {
			got, err := parseID(id)
			if err != nil {
				t.Fatalf("parseID(%q) error: %v", id, err)
			}
			if got != want {
				t.Errorf("parseID(%q) = %+v, want %+v", id, got, want)
			}
			if got.ID() != id {
				t.Errorf("ID() = %q, want %q", got.ID(), id)
			}
		})
	}
}

func TestParseID_rejects_malformed_ids(t *testing.T) {
	for _, id := range []string{"", "a/b/c/d", "ws1//key", "/key", "ws1/", "1key", "ws1/dev/bad.key"} {
		if _, err := parseID(id); err == nil {
			t.Errorf("parseID(%q) succeeded, want error", id)
		}
	}
}

func TestItemPath_escapes_segments_per_scope(t *testing.T) {
	cases := []struct {
		k    scopedKey
		kind string
		want string
	}{
		{scopedKey{Key: "region"}, kindVariables, "/api/v3/general/tenants/acme/vars/variables/region"},
		{scopedKey{Workspace: "ws1", Key: "db"}, kindSecrets, "/api/v3/general/tenants/acme/workspaces/ws1/vars/secrets/db"},
		{scopedKey{Workspace: "ws 1", Environment: "dev", Key: "k"}, kindVariables, "/api/v3/general/tenants/acme/workspaces/ws%201/envs/dev/vars/variables/k"},
	}
	for _, c := range cases {
		if got := c.k.itemPath("acme", c.kind); got != c.want {
			t.Errorf("itemPath(%+v, %s) = %q, want %q", c.k, c.kind, got, c.want)
		}
	}
}
