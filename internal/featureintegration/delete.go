// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
)

func (r *featureResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m featureModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)

	if resp.Diagnostics.HasError() || m.KeepEnabledOnDestroy.ValueBool() {
		return
	}

	name := m.Name.ValueString()
	effectivePath := "/api/v3/general/tenants/" + url.PathEscape(r.client.Tenant) +
		"/integrations/" + url.PathEscape(name) + "/effective"

	var effective struct {
		AlwaysEnabled bool `json:"alwaysEnabled"`
	}

	err := r.client.DoJSON(ctx, http.MethodGet, effectivePath, nil, &effective)
	if errors.Is(err, client.ErrNotFound) {
		return
	}

	if err != nil {
		addAPIError(&resp.Diagnostics, err, name)
		return
	}

	if effective.AlwaysEnabled {
		resp.Diagnostics.AddWarning("Feature integration left enabled",
			fmt.Sprintf("%q is always enabled in Wayfinder and cannot be disabled; it was removed from Terraform state only.", name))

		return
	}

	ref := integrationRef(name)
	obj, err := r.client.Get(ctx, ref)

	if errors.Is(err, client.ErrNotFound) {
		return
	}

	if err != nil {
		addAPIError(&resp.Diagnostics, err, name)
		return
	}

	spec := obj["spec"].(map[string]any)
	if spec["enabled"] == false {
		return
	}

	spec["enabled"] = false

	if _, err := r.client.Update(ctx, ref, obj); err != nil {
		addAPIError(&resp.Diagnostics, err, name)
	}
}
