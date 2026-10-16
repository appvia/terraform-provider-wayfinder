package vars_test

import (
	"slices"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/appvia/terraform-provider-wayfinder/internal/vars"
)

func diffScopes(got, want []string) (missing, extra []string) {
	for _, s := range want {
		if !slices.Contains(got, s) {
			missing = append(missing, s)
		}
	}
	for _, s := range got {
		if !slices.Contains(want, s) {
			extra = append(extra, s)
		}
	}
	return missing, extra
}

func TestSupportedScopes_matchManifest(t *testing.T) {
	m, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	missing, extra := diffScopes(vars.SupportedScopes, m.VarScopes)
	if len(missing) > 0 || len(extra) > 0 {
		t.Fatalf("vars scopes differ from the Wayfinder vars API (manifest varScopes): missing support for %v, supported but no longer served %v; implement the scope in internal/vars or classify it exclude in tools/manifestgen/kinds.yaml", missing, extra)
	}
}

func TestSupportedScopes_haveLifecycleCoverage(t *testing.T) {
	var keys []string
	for k := range testScopes {
		keys = append(keys, k)
	}
	missing, extra := diffScopes(keys, vars.SupportedScopes)
	if len(missing) > 0 || len(extra) > 0 {
		t.Fatalf("testScopes must cover every supported scope: missing %v, unexpected %v", missing, extra)
	}
}
