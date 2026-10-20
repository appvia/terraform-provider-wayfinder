// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package manifestresource

import (
	"fmt"
	"slices"
	"strings"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

// kindError reports a kind the provider cannot manage. Its text is shown to users verbatim.
type kindError struct {
	kind     string
	excluded bool
}

func (e *kindError) Error() string {
	if e.excluded {
		return "Kind " + e.kind + " cannot be managed with Terraform."
	}
	return "Kind " + e.kind + " is not supported by this provider version; upgrade the provider."
}

// splitAPIVersion splits "<group>/<version>". Wayfinder objects carry a short group token
// (apiVersion: org/v3); a fully-qualified group (org.appvia.io/v3) is accepted too.
func splitAPIVersion(apiVersion string) (group, version string, err error) {
	group, version, ok := strings.Cut(apiVersion, "/")
	if !ok || group == "" || version == "" || strings.Contains(version, "/") {
		return "", "", fmt.Errorf("apiVersion %q must have the form <group>/<version>, for example org/v3", apiVersion)
	}
	return group, version, nil
}

// groupMatches compares API groups by exact value or by their first DNS label, so that
// "org", "org.appvia.io" and a manifest group of either form identify the same group.
func groupMatches(manifestGroup, group string) bool {
	first := func(g string) string { s, _, _ := strings.Cut(g, "."); return s }
	return manifestGroup == group || first(manifestGroup) == first(group)
}

// resolveKind finds kind in group among every manifest kind, typed or manifest-only. version
// is checked only when non-empty.
func resolveKind(m *manifest.Manifest, group, version, kind string) (manifest.Kind, error) {
	for _, k := range m.Kinds {
		if k.Kind != kind || !groupMatches(k.Group, group) {
			continue
		}
		if version != "" && version != k.Version {
			return manifest.Kind{}, fmt.Errorf("apiVersion %s/%s is not served for kind %s; use %s/%s", group, version, kind, k.Group, k.Version)
		}
		return k, nil
	}
	return manifest.Kind{}, &kindError{kind: kind, excluded: slices.Contains(m.Excluded, kind)}
}
