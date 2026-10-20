// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package acctest

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi/fakekinds"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

const fakeTenant = "acme"

func envOf(vars map[string]string) Env {
	return LoadEnv(func(k string) string { return vars[k] })
}

func TestEnvMissing_whenServerAndCredentialsUnset(t *testing.T) {
	// Given no acceptance environment.
	e := envOf(nil)
	// When checking what is missing.
	got := strings.Join(e.Missing(), "; ")
	// Then the server and a credential are reported.
	for _, want := range []string{"WAYFINDER_SERVER", "WAYFINDER_TOKEN or WAYFINDER_CLIENT_ID and WAYFINDER_CLIENT_SECRET"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing = %q, want it to mention %q", got, want)
		}
	}
}

func TestEnvMissing_whenTokenWithoutTenant(t *testing.T) {
	// Given a token but no tenant.
	e := envOf(map[string]string{"WAYFINDER_SERVER": "https://x", "WAYFINDER_TOKEN": "t"})
	// Then the tenant is reported missing.
	if got := e.Missing(); len(got) != 1 || got[0] != "WAYFINDER_TENANT" {
		t.Fatalf("missing = %v, want [WAYFINDER_TENANT]", got)
	}
}

func TestEnvMissing_whenClientCredentialsDeriveTenant(t *testing.T) {
	// Given client credentials whose ID names the tenant.
	e := envOf(map[string]string{"WAYFINDER_SERVER": "https://x", "WAYFINDER_CLIENT_ID": "acme:ci", "WAYFINDER_CLIENT_SECRET": "s"})
	// Then nothing is missing.
	if got := e.Missing(); len(got) != 0 {
		t.Fatalf("missing = %v, want none", got)
	}
}

func TestVersionWarning_whenReleasesMatchIgnoringPrefix(t *testing.T) {
	if got := VersionWarning("3.3.2", "v3.3.2"); got != "" {
		t.Fatalf("warning = %q, want none", got)
	}
}

func TestVersionWarning_whenReleasesDiffer(t *testing.T) {
	got := VersionWarning("v3.4.0-dev", "v3.3.2")
	if !strings.HasPrefix(got, "WARNING:") || !strings.Contains(got, "v3.4.0-dev") || !strings.Contains(got, "v3.3.2") {
		t.Fatalf("warning = %q", got)
	}
}

func TestServerRelease_whenServerReportsVersion(t *testing.T) {
	// Given a server reporting a release.
	f := fakeapi.New(t)
	f.SetServerVersion("v9.8.7")
	// When reading it.
	c := &client.Client{BaseURL: f.URL, Tenant: fakeTenant, TokenSource: staticToken(fakeapi.DefaultToken)}
	got, err := ServerRelease(context.Background(), c)
	// Then the release is returned.
	if err != nil || got != "v9.8.7" {
		t.Fatalf("release = %q, err = %v", got, err)
	}
}

func TestRandomName_whenGenerated(t *testing.T) {
	a, b := RandomName(), RandomName()
	if !strings.HasPrefix(a, NamePrefix) || len(a) != len(NamePrefix)+8 || a == b {
		t.Fatalf("names = %q, %q", a, b)
	}
}

// sweepEnv is a fake API with every embedded kind plus a client for it.
func sweepEnv(t *testing.T) (*fakeapi.Server, *client.Client, *manifest.Manifest) {
	t.Helper()
	m, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	f := fakeapi.New(t)
	fakekinds.RegisterAll(f, m)
	c := &client.Client{BaseURL: f.URL, Tenant: fakeTenant, TokenSource: staticToken(fakeapi.DefaultToken)}
	return f, c, m
}

type staticToken string

func (s staticToken) Token(context.Context) (string, error) { return string(s), nil }

func kindRef(t *testing.T, m *manifest.Manifest, kind, workspace, name string) fakeapi.Ref {
	t.Helper()
	k, ok := m.FindKind(kindGroup(t, m, kind), kind)
	if !ok {
		t.Fatalf("kind %s not found", kind)
	}
	return fakeapi.Ref{Tenant: fakeTenant, Workspace: workspace, Group: k.Group, Version: k.Version, APIPath: k.APIPath, Name: name}
}

func kindGroup(t *testing.T, m *manifest.Manifest, kind string) string {
	t.Helper()
	for _, k := range m.Kinds {
		if k.Kind == kind {
			return k.Group
		}
	}
	t.Fatalf("kind %s not found", kind)
	return ""
}

func TestSweepKind_whenTestAndOtherObjectsExist(t *testing.T) {
	// Given tfacc and non-tfacc service accounts at tenant scope and inside a tfacc workspace.
	f, c, m := sweepEnv(t)
	f.Seed(kindRef(t, m, "Workspace", "", "tfacc-ws"), map[string]any{"spec": map[string]any{}})
	f.Seed(kindRef(t, m, "Workspace", "", "keep"), map[string]any{"spec": map[string]any{}})
	for _, ws := range []string{"", "tfacc-ws", "keep"} {
		for _, name := range []string{"tfacc-sa", "prod-sa"} {
			f.Seed(kindRef(t, m, "ServiceAccount", ws, name), map[string]any{"spec": map[string]any{}})
		}
	}
	// When sweeping service accounts.
	if err := sweepKind(context.Background(), c, mustKind(t, m, "ServiceAccount")); err != nil {
		t.Fatal(err)
	}
	// Then only tfacc objects in tenant and tfacc workspaces are gone.
	for _, tc := range []struct {
		ws, name string
		exists   bool
	}{{"", "tfacc-sa", false}, {"tfacc-ws", "tfacc-sa", false}, {"", "prod-sa", true}, {"tfacc-ws", "prod-sa", true}, {"keep", "tfacc-sa", true}} {
		if _, ok := f.Object(kindRef(t, m, "ServiceAccount", tc.ws, tc.name)); ok != tc.exists {
			t.Fatalf("%s/%s exists = %v, want %v", tc.ws, tc.name, ok, tc.exists)
		}
	}
}

func TestSweepVars_whenTestAndOtherKeysExist(t *testing.T) {
	// Given tfacc and other keys at tenant, tfacc-workspace and tfacc-environment scope,
	// plus an environment-scope tfacc key in a workspace the tests did not create.
	f, c, m := sweepEnv(t)
	f.Seed(kindRef(t, m, "Workspace", "", "tfacc-ws"), map[string]any{"spec": map[string]any{}})
	f.Seed(kindRef(t, m, "Environment", "tfacc-ws", "tfacc-env"), map[string]any{"spec": map[string]any{}})
	f.Seed(kindRef(t, m, "Workspace", "", "keep-ws"), map[string]any{"spec": map[string]any{}})
	f.Seed(kindRef(t, m, "Environment", "keep-ws", "tfacc-env"), map[string]any{"spec": map[string]any{}})
	swept := []fakeapi.VarScope{
		{Tenant: fakeTenant}, {Tenant: fakeTenant, Workspace: "tfacc-ws"},
		{Tenant: fakeTenant, Workspace: "tfacc-ws", Environment: "tfacc-env"},
	}
	for _, scope := range swept {
		f.SetVariable(scope, "tfacc-v", json.RawMessage(`"x"`))
		f.SetSecret(scope, "tfacc-s", json.RawMessage(`"x"`))
		f.SetVariable(scope, "keep", json.RawMessage(`"x"`))
	}
	other := fakeapi.VarScope{Tenant: fakeTenant, Workspace: "keep-ws", Environment: "tfacc-env"}
	f.SetVariable(other, "tfacc-v", json.RawMessage(`"x"`))
	// When sweeping variables and secrets.
	if err := sweepVars(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	// Then only the tfacc keys in tenant and tfacc workspaces are gone.
	for _, scope := range swept {
		if _, ok := f.Variable(scope, "tfacc-v"); ok {
			t.Fatalf("%+v tfacc-v survived", scope)
		}
		if _, ok := f.Secret(scope, "tfacc-s"); ok {
			t.Fatalf("%+v tfacc-s survived", scope)
		}
		if _, ok := f.Variable(scope, "keep"); !ok {
			t.Fatalf("%+v keep was deleted", scope)
		}
	}
	if _, ok := f.Variable(other, "tfacc-v"); !ok {
		t.Fatalf("%+v tfacc-v in a non-test workspace was deleted", other)
	}
}

func mustKind(t *testing.T, m *manifest.Manifest, kind string) manifest.Kind {
	t.Helper()
	k, ok := m.FindKind(kindGroup(t, m, kind), kind)
	if !ok {
		t.Fatalf("kind %s not found", kind)
	}
	return *k
}
