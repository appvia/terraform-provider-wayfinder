// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

// Package ephemeral implements read-only, sensitive Wayfinder ephemeral resources.
package ephemeral

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

var actionPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// getAction reads one object sub-resource. Error responses can contain credentials,
// so diagnostics deliberately omit server messages, unlike engine.AddAPIError.
func getAction(ctx context.Context, c *client.Client, k manifest.Kind, ref client.Ref, action string) (json.RawMessage, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !actionPattern.MatchString(action) {
		diags.AddError("Invalid action", "Action must match ^[a-z][a-z0-9-]*$.")
		return nil, diags
	}

	if slices.ContainsFunc([]string{ref.Name, ref.Workspace, ref.Environment, ref.ObjectVersion}, invalidSegment) {
		diags.AddError("Invalid resource reference", "Object names, scopes and versions must be single path segments, not . or .. .")
		return nil, diags
	}

	item, err := ref.ItemPath(c.Tenant)
	if err != nil {
		diags.AddError("Invalid resource reference", err.Error())
		return nil, diags
	}

	path := item + "/" + url.PathEscape(action)

	var raw json.RawMessage
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		var api *client.APIError

		switch {
		case errors.Is(err, client.ErrNotFound):
			detail := fmt.Sprintf("Wayfinder returned 404 for %s. Check the object exists at that scope, and that the action is valid.", path)
			if k.Kind == "Webhook" && action == "ingesturl" {
				detail += fmt.Sprintf(" The URL exists only after the webhook is rotated, with a `wayfinder_webhook_rotation` resource or `wf rotate webhook %s`.", ref.Name)
			}

			diags.AddError("Wayfinder action not found", detail)
		case errors.As(err, &api) && api.StatusCode == http.StatusForbidden:
			detail := fmt.Sprintf("Permission denied for action %s on %s.", action, k.Kind)
			if k.Kind == "Webhook" && action == "ingesturl" {
				detail += " The ingesturl action needs the webhooks/ingesturl permission (granted with rotate)."
			}

			diags.AddError("Wayfinder action permission denied", detail)
		case api != nil:
			diags.AddError("Wayfinder request failed", fmt.Sprintf("GET %s returned HTTP %d. Check the action and your Wayfinder permissions.", path, api.StatusCode))
		default:
			diags.AddError("Wayfinder request failed", fmt.Sprintf("Unable to read GET %s. Check the action response, Wayfinder connection and provider credentials.", path))
		}

		return nil, diags
	}

	return raw, diags
}

func invalidSegment(value string) bool {
	return value == "." || value == ".." || strings.ContainsAny(value, `/\`)
}
