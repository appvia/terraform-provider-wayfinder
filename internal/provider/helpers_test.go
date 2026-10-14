package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi/fakekinds"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

const testTenant = "acme"

// authEnv lists every environment variable the provider's authentication reads.
var authEnv = []string{
	"WAYFINDER_SERVER", "WAYFINDER_TOKEN", "WAYFINDER_TENANT", "WAYFINDER_CLIENT_ID", "WAYFINDER_CLIENT_SECRET",
	"WAYFINDER_SERVICE_ACCOUNT", "WAYFINDER_FEDERATED_TOKEN_FILE", "WAYFINDER_FEDERATED_TOKEN_REQUEST_URL",
	"WAYFINDER_FEDERATED_TOKEN_REQUEST_TOKEN", "WAYFINDER_FEDERATED_TOKEN_REQUEST_AUDIENCE",
	"ACTIONS_ID_TOKEN_REQUEST_URL", "ACTIONS_ID_TOKEN_REQUEST_TOKEN",
	"WAYFINDER_USE_CLI", "WAYFINDER_CLI_PROFILE", "WAYFINDER_CONFIG",
}

// testEnv is the real provider served in-process against a fake API that knows every kind of
// the embedded manifest.
type testEnv struct {
	fake      *fakeapi.Server
	m         *manifest.Manifest
	factories map[string]func() (tfprotov6.ProviderServer, error)
}

func newTestEnv(t *testing.T) testEnv {
	t.Helper()
	for _, k := range authEnv {
		t.Setenv(k, "")
	}
	// Never fall back to the developer's real wf CLI session.
	t.Setenv("WAYFINDER_USE_CLI", "false")
	m, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	f := fakeapi.New(t)
	fakekinds.RegisterAll(f, m)
	return testEnv{fake: f, m: m, factories: map[string]func() (tfprotov6.ProviderServer, error){
		"wayfinder": providerserver.NewProtocol6WithError(New("test")()),
	}}
}

// providerBlock configures the provider against the fake with a static token.
func (e testEnv) providerBlock() string {
	return fmt.Sprintf("provider \"wayfinder\" {\n server = %q\n tenant = %q\n token = %q\n}\n", e.fake.URL, testTenant, fakeapi.DefaultToken)
}

// kind returns the embedded kind whose Terraform type is wayfinder_<suffix>.
func (e testEnv) kind(t *testing.T, suffix string) manifest.Kind {
	t.Helper()
	for _, k := range e.m.Kinds {
		if k.Typed && strings.TrimPrefix(k.TerraformName, "wayfinder_") == suffix {
			return k
		}
	}
	t.Fatalf("embedded manifest has no typed kind wayfinder_%s", suffix)
	return manifest.Kind{}
}

func (e testEnv) ref(k manifest.Kind, workspace, name string) fakeapi.Ref {
	return fakeapi.Ref{Tenant: testTenant, Workspace: workspace, Group: k.Group, Version: k.Version, APIPath: k.APIPath, Name: name}
}

func nestedMap(obj map[string]any, keys ...string) (map[string]any, error) {
	cur := obj
	for i, k := range keys {
		v := cur[k]
		next, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: want map, got %T", strings.Join(keys[:i+1], "."), v)
		}
		cur = next
	}
	return cur, nil
}

func nestedSlice(obj map[string]any, keys ...string) ([]any, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("nestedSlice: no keys given")
	}
	parent, err := nestedMap(obj, keys[:len(keys)-1]...)
	if err != nil {
		return nil, err
	}
	last := keys[len(keys)-1]
	v := parent[last]
	s, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%s: want slice, got %T", strings.Join(keys, "."), v)
	}
	return s, nil
}

func TestNestedMap_whenShapeWrong(t *testing.T) {
	obj := map[string]any{
		"metadata": map[string]any{"labels": "not-a-map"},
		"spec":     map[string]any{"helm": map[string]any{"chart": "x"}},
	}
	for _, tc := range []struct {
		name     string
		keys     []string
		wantErr  string
		wantKeys string
	}{
		{name: "missing key", keys: []string{"spec", "absent"}, wantErr: "spec.absent: want map, got <nil>"},
		{name: "non-map value", keys: []string{"metadata", "labels"}, wantErr: "metadata.labels: want map, got string"},
		{name: "success", keys: []string{"spec", "helm"}, wantKeys: "chart"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given: an object with a missing key, a non-map value, and a valid nested map.
			// When: nestedMap walks the key path.
			got, err := nestedMap(obj, tc.keys...)
			// Then: wrong shapes return the path in the error; the right shape returns the map.
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("nestedMap(%v) error = %v; want %q", tc.keys, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("nestedMap(%v) unexpected error: %v", tc.keys, err)
			}
			if _, ok := got[tc.wantKeys]; !ok {
				t.Fatalf("nestedMap(%v) = %#v; want key %q", tc.keys, got, tc.wantKeys)
			}
		})
	}
}
