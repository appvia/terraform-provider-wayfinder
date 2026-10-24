// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package webhookrotation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

// newRotationID is a variable so tests can make it fail.
var newRotationID = func() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func (r *rotationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m rotationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ref, item, ok := r.webhook(m.Webhook.ValueString(), m.Workspace, &resp.Diagnostics)
	if !ok {
		return
	}

	// The response carries the new ingest URL, a credential: it is decoded only as JSON and
	// then dropped, never read, logged, stored or put in a diagnostic.
	var discard json.RawMessage

	path := item + "/rotate"
	if err := r.client.DoJSON(ctx, http.MethodPost, path, struct{}{}, &discard); err != nil {
		addRotateError(&resp.Diagnostics, err, path, ref)
		return
	}

	rotatedAt := time.Now().UTC().Format(time.RFC3339Nano)

	rotationID, err := newRotationID()
	if err != nil {
		resp.Diagnostics.AddError("Unable to generate rotation ID",
			fmt.Sprintf("Wayfinder rotated the webhook URL, but no random rotation ID could be generated: %v. Apply again to rotate once more.", err))

		return
	}

	m.ID = types.StringValue(objectID(m.Workspace.ValueString(), m.Webhook.ValueString()))
	m.Name = m.Webhook
	m.RotationID = types.StringValue(rotationID)
	m.RotatedAt = types.StringValue(rotatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *rotationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m rotationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ref, item, ok := r.webhook(m.Webhook.ValueString(), m.Workspace, &resp.Diagnostics)
	if !ok {
		return
	}

	_, err := r.client.Get(ctx, ref)
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		addRequestError(&resp.Diagnostics, err, http.MethodGet, item)
	}
}

// Update only carries the plan into state: every configurable attribute requires replacement.
func (r *rotationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m rotationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)

	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
	}
}

// Delete only forgets the rotation. The current URL keeps working, because rotating here would
// break senders that already use it.
func (r *rotationResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

// webhook resolves the webhook object and its item path. Names are checked again here because
// values unknown at plan time skip the schema validators.
func (r *rotationResource) webhook(name string, workspace types.String, d *diag.Diagnostics) (client.Ref, string, bool) {
	if invalidSegment(name) || (!workspace.IsNull() && invalidSegment(workspace.ValueString())) {
		d.AddError("Invalid webhook reference", "webhook and workspace must be single non-empty names, not . or .., without / or \\.")
		return client.Ref{}, "", false
	}

	embedded, err := manifest.LoadEmbedded()
	if err != nil {
		d.AddError("Unable to load Wayfinder manifest", err.Error())
		return client.Ref{}, "", false
	}

	k, ok := embedded.FindKind("integrations", "Webhook")
	if !ok {
		d.AddError("Webhook type unavailable", "The embedded Wayfinder manifest does not contain integrations/Webhook.")
		return client.Ref{}, "", false
	}

	ref := client.Ref{
		Group: k.Group, Version: k.Version, APIPath: k.APIPath, Versioned: k.Versioned,
		Name: name, Workspace: workspace.ValueString(),
	}

	item, err := ref.ItemPath(r.client.Tenant)
	if err != nil {
		d.AddError("Invalid webhook reference", err.Error())
		return client.Ref{}, "", false
	}

	return ref, item, true
}

func objectID(workspace, webhook string) string {
	if workspace == "" {
		return webhook
	}

	return workspace + "/" + webhook
}

func addRotateError(d *diag.Diagnostics, err error, path string, ref client.Ref) {
	scope := "tenant scope"
	if ref.Workspace != "" {
		scope = fmt.Sprintf("workspace %q", ref.Workspace)
	}

	var api *client.APIError

	switch {
	case errors.Is(err, client.ErrNotFound):
		d.AddError("Webhook not found",
			fmt.Sprintf("Wayfinder returned 404 for POST %s. Check webhook %q exists at %s.", path, ref.Name, scope))
	case errors.As(err, &api) && api.StatusCode == http.StatusForbidden:
		d.AddError("Permission denied",
			fmt.Sprintf("Wayfinder returned 403 for POST %s. Rotating webhook %q at %s requires the webhooks/rotate permission.", path, ref.Name, scope))
	default:
		addRequestError(d, err, http.MethodPost, path)
	}
}

// addRequestError reports a failed request by method, path and status only. Unlike
// engine.AddAPIError it never includes the server message: rotate responses carry the ingest
// URL, and error bodies may echo credentials.
func addRequestError(d *diag.Diagnostics, err error, method, path string) {
	var (
		api        *client.APIError
		validation *client.ValidationError
	)

	switch {
	case errors.As(err, &api):
		d.AddError("Wayfinder request failed", fmt.Sprintf("%s %s returned HTTP %d.", method, path, api.StatusCode))
	case errors.As(err, &validation):
		d.AddError("Wayfinder request failed", fmt.Sprintf("%s %s returned HTTP %d.", method, path, http.StatusBadRequest))
	default:
		d.AddError("Wayfinder request failed",
			fmt.Sprintf("Unable to complete %s %s. Check the Wayfinder connection and provider credentials.", method, path))
	}
}
