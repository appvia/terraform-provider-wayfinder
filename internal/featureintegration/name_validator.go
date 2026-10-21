// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type bareNameValidator struct{}

func (bareNameValidator) Description(context.Context) string {
	return `Use the bare feature name, e.g. "azure" rather than "wayfinder.azure".`
}

func (v bareNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v bareNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	name := req.ConfigValue.ValueString()
	if name == "" || strings.HasPrefix(name, "wayfinder.") {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid feature name", v.Description(ctx))
	}
}
