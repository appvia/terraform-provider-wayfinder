// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package webhookrotation

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// segmentValidator requires a value that is safe as one request path segment, because webhook
// and workspace names become part of the rotate request path.
type segmentValidator struct{}

func (segmentValidator) Description(context.Context) string {
	return `Must be a single non-empty name, not "." or "..", without "/" or "\".`
}

func (v segmentValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v segmentValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if invalidSegment(req.ConfigValue.ValueString()) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid name", v.Description(ctx))
	}
}

func invalidSegment(value string) bool {
	return value == "" || value == "." || value == ".." || strings.ContainsAny(value, `/\`)
}
