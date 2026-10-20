// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package generated

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var embeddedManifest = sync.OnceValues(manifest.LoadEmbedded)

func generatedKind(group, name string) (manifest.Kind, error) {
	m, err := embeddedManifest()
	if err != nil {
		return manifest.Kind{}, err
	}

	k, ok := m.FindKind(group, name)
	if !ok {
		return manifest.Kind{}, fmt.Errorf("generated kind %s/%s absent from embedded manifest; regenerate the provider", group, name)
	}

	copy := *k
	// The engine consumes a suffix, while real manifests carry the full type name.
	copy.TerraformName = strings.TrimPrefix(copy.TerraformName, "wayfinder_")

	return copy, nil
}

type resourceManifestError struct {
	resource.Resource
	err error
}

func resourceWithManifestError(r resource.Resource, err error) resource.Resource {
	if err == nil {
		return r
	}

	return resourceManifestError{r, err}
}

func (r resourceManifestError) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	r.Resource.Schema(ctx, req, resp)
	resp.Diagnostics.AddError("Invalid embedded manifest", r.err.Error())
}

type dataSourceManifestError struct {
	datasource.DataSource
	err error
}

func dataSourceWithManifestError(s datasource.DataSource, err error) datasource.DataSource {
	if err == nil {
		return s
	}

	return dataSourceManifestError{s, err}
}

func (s dataSourceManifestError) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	s.DataSource.Schema(ctx, req, resp)
	resp.Diagnostics.AddError("Invalid embedded manifest", s.err.Error())
}
