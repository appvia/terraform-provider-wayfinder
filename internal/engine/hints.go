// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"fmt"
	"strings"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

// featurePrefix is the name prefix Wayfinder gives stored feature integrations.
const featurePrefix = "wayfinder."

// notFoundHint explains a missing tenant-scoped Integration whose name lacks
// the "wayfinder." prefix: feature integrations are stored under the prefixed
// name, so a bare feature name never matches. It returns "" for every other
// kind or name. Features are tenant-scope only, so a workspace-scoped identity
// never gets the hint.
func notFoundHint(k manifest.Kind, i convert.Identity) string {
	if k.Group != "integrations" || k.Kind != "Integration" || i.Workspace != "" || strings.HasPrefix(i.Name, featurePrefix) {
		return ""
	}

	return fmt.Sprintf("Feature integrations such as %q are stored as %q. To read or import an enabled feature, use %q; to enable one, use the wayfinder_feature_integration resource.", i.Name, featurePrefix+i.Name, featurePrefix+i.Name)
}
