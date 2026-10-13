package acctest

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

const serverInfoPath = "/api/v3/general/serverinfo"

// serverInfo is the part of types.ServerInfo the tests read.
type serverInfo struct {
	Version struct {
		Release string `json:"release"`
	} `json:"version"`
}

// ServerRelease reads the version release a Wayfinder API reports. The endpoint needs a
// token, so it goes through the authenticated client.
func ServerRelease(ctx context.Context, c *client.Client) (string, error) {
	var info serverInfo
	if err := c.DoJSON(ctx, http.MethodGet, serverInfoPath, nil, &info); err != nil {
		return "", fmt.Errorf("reading server info: %w", err)
	}
	return info.Version.Release, nil
}

// VersionWarning returns a warning when the server release differs from the Wayfinder version
// the provider was generated from (a leading "v" is ignored), or "" when they match.
func VersionWarning(server, generated string) string {
	if strings.TrimPrefix(server, "v") == strings.TrimPrefix(generated, "v") {
		return ""
	}
	return fmt.Sprintf("WARNING: Wayfinder server release %q differs from the provider's generated Wayfinder version %q; failures may come from API differences", server, generated)
}

// logServerVersion logs the server release next to the embedded manifest version, warning
// (never failing) when they differ.
func logServerVersion(t *testing.T) {
	t.Helper()
	m, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatalf("loading embedded manifest: %v", err)
	}
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	release, err := ServerRelease(ctx, c)
	if err != nil {
		t.Fatalf("server info: %v", err)
	}
	t.Logf("Wayfinder server release %q; provider generated from wayfinderVersion %q", release, m.WayfinderVersion)
	if w := VersionWarning(release, m.WayfinderVersion); w != "" {
		t.Log(w)
	}
}
