// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type paramValuesValidator struct{}

func (paramValuesValidator) Description(context.Context) string {
	return "Parameter values must be JSON-encoded strings. Use jsonencode(null) for a JSON null instead of a Terraform null element."
}

func (v paramValuesValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v paramValuesValidator) ValidateMap(ctx context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	for key, value := range req.ConfigValue.Elements() {
		if value.IsNull() {
			resp.Diagnostics.AddAttributeError(req.Path.AtMapKey(key), "Invalid feature parameter", v.Description(ctx))
		}
	}
}
