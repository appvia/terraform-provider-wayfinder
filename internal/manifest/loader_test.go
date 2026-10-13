package manifest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/schema"
)

func TestLoadEmbedded_GoldenHasFourKindsThreeTyped(t *testing.T) {
	m, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	if got := len(m.Kinds); got != 4 {
		t.Fatalf("kinds: got %d, want 4", got)
	}
	if got := len(m.TypedKinds()); got != 3 {
		t.Fatalf("typed kinds: got %d, want 3", got)
	}
	pipeline, ok := m.FindKind("core.appvia.io", "Pipeline")
	if !ok {
		t.Fatal("FindKind(Pipeline): not found")
	}
	if pipeline.Typed {
		t.Fatal("Pipeline must be manifest-only (typed=false)")
	}
}

func TestFindKind_MissingReturnsFalse(t *testing.T) {
	m, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	if _, ok := m.FindKind("core.appvia.io", "NoSuchKind"); ok {
		t.Fatal("FindKind(NoSuchKind): want not found")
	}
}

func TestEmbeddedMatchesGoldenFixture(t *testing.T) {
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if !bytes.Equal(schema.Manifest, golden) {
		t.Fatal("schema/wayfinder-manifest.json drifted from testdata/golden-manifest.json")
	}
}

func TestLoad_RejectsManifestVersion2(t *testing.T) {
	src, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	bad := strings.Replace(string(src), `"manifestVersion": 1`, `"manifestVersion": 2`, 1)
	path := writeTemp(t, bad)
	_, err = Load(path)
	if err == nil || !strings.Contains(err.Error(), "manifestVersion") {
		t.Fatalf("Load: want manifestVersion error, got %v", err)
	}
}

func TestLoad_RejectsBadScope(t *testing.T) {
	src, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	bad := strings.Replace(string(src), `"workspace"`, `"system"`, 1)
	path := writeTemp(t, bad)
	_, err = Load(path)
	if err == nil || !strings.Contains(err.Error(), "system") {
		t.Fatalf("Load: want scope error mentioning system, got %v", err)
	}
}

func TestLoad_MissingFileErrorsWithPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.json")
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("Load: want error containing %q, got %v", path, err)
	}
}

func TestLoad_ValidFileRoundTrips(t *testing.T) {
	m, err := Load(goldenPath)
	if err != nil {
		t.Fatalf("Load golden: %v", err)
	}
	if len(m.Kinds) != 4 {
		t.Fatalf("kinds: got %d, want 4", len(m.Kinds))
	}
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return path
}
