// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

// Package vars implements the wayfinder_variable and wayfinder_secret resources and the
// wayfinder_variable data source over Wayfinder's per-key variables API (plan R16).
//
// Variables and secrets live in Wayfinder's variable and secret hierarchy: tenant, workspace,
// environment or instance scope. Each Terraform object manages exactly one key at one scope; nothing in
// this package reads or writes a scope's full collection.
//
// ProviderData contract: the provider's Configure must set ResourceData and DataSourceData to a
// *client.Client whose Tenant is the provider tenant. A nil ProviderData (early validation calls)
// is tolerated; any other type is reported as an error.
package vars

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
)

// Resources returns the resource factories of this package for provider registration.
func Resources() []func() resource.Resource {
	return []func() resource.Resource{NewVariableResource, NewSecretResource}
}

// DataSources returns the data source factories of this package for provider registration.
func DataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{NewVariableDataSource}
}

// keyPattern is the server's key rule (server/org/handlers/internal/varset/validate.go:29-30).
var keyPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

// supportedScopes lists the variable and secret scopes this package implements. It must equal the
// manifest's varScopes and the keys of the tests' testScopes.
var supportedScopes = []string{"environment", "instance", "tenant", "workspace"}

// API collection names in the per-key routes.
const (
	kindVariables = "variables"
	kindSecrets   = "secrets"
)

// scopedKey is one key at one scope; empty Workspace means tenant scope, empty Environment
// means workspace scope and empty Instance means environment scope.
type scopedKey struct {
	Workspace, Environment, Instance, Key string
}

// ID renders the import ID `[workspace/[environment/[instance/]]]key`.
func (k scopedKey) ID() string {
	parts := make([]string, 0, 4)
	if k.Workspace != "" {
		parts = append(parts, k.Workspace)
	}
	if k.Environment != "" {
		parts = append(parts, k.Environment)
	}
	if k.Instance != "" {
		parts = append(parts, k.Instance)
	}
	return strings.Join(append(parts, k.Key), "/")
}

var errInvalidID = errors.New("expected an ID of the form key, workspace/key, workspace/environment/key or workspace/environment/instance/key")

// parseID parses `[workspace/[environment/[instance/]]]key`; the number of `/` decides the scope.
func parseID(id string) (scopedKey, error) {
	parts := strings.Split(id, "/")
	for _, p := range parts {
		if p == "" {
			return scopedKey{}, fmt.Errorf("%q: %w", id, errInvalidID)
		}
	}
	var k scopedKey
	switch len(parts) {
	case 1:
		k = scopedKey{Key: parts[0]}
	case 2:
		k = scopedKey{Workspace: parts[0], Key: parts[1]}
	case 3:
		k = scopedKey{Workspace: parts[0], Environment: parts[1], Key: parts[2]}
	case 4:
		k = scopedKey{Workspace: parts[0], Environment: parts[1], Instance: parts[2], Key: parts[3]}
	default:
		return scopedKey{}, fmt.Errorf("%q: %w", id, errInvalidID)
	}
	if !keyPattern.MatchString(k.Key) {
		return scopedKey{}, fmt.Errorf("%q: key %q must start with a letter or underscore and contain only letters, digits, underscores and dashes", id, k.Key)
	}
	return k, nil
}

// itemPath is the escaped per-key route for kind (variables or secrets) under tenant. Instance
// scope is served under instancevars with no /vars segment, as upstream registers it.
func (k scopedKey) itemPath(tenant, kind string) string {
	p := "/api/v3/general/tenants/" + url.PathEscape(tenant)
	if k.Workspace != "" {
		p += "/workspaces/" + url.PathEscape(k.Workspace)
		if k.Environment != "" {
			p += "/envs/" + url.PathEscape(k.Environment)
			if k.Instance != "" {
				return p + "/instancevars/" + url.PathEscape(k.Instance) + "/" + kind + "/" + url.PathEscape(k.Key)
			}
		}
	}
	return p + "/vars/" + kind + "/" + url.PathEscape(k.Key)
}

// rawItem is types.VariableItem {key, value}; secrets come back with a null value.
type rawItem struct {
	Value json.RawMessage `json:"value"`
}

// setValueRequest is types.SetValueRequest {value}.
type setValueRequest struct {
	Value json.RawMessage `json:"value"`
}

// get reads one key. A missing key yields an error matching client.ErrNotFound.
func get(ctx context.Context, c *client.Client, k scopedKey, kind string) (rawItem, error) {
	var out rawItem
	if err := c.DoJSON(ctx, http.MethodGet, k.itemPath(c.Tenant, kind), nil, &out); err != nil {
		return out, err
	}
	// The server indents its responses; keep the compact form jsonencode produces so an
	// imported value matches a created one. Secrets come back without a value.
	if len(out.Value) == 0 {
		return out, nil
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, out.Value); err != nil {
		return out, fmt.Errorf("decoding %s %s value: %w", kind, k.ID(), err)
	}
	out.Value = compact.Bytes()
	return out, nil
}

// put upserts one key with a JSON value.
func put(ctx context.Context, c *client.Client, k scopedKey, kind string, value json.RawMessage) error {
	return c.DoJSON(ctx, http.MethodPut, k.itemPath(c.Tenant, kind), setValueRequest{Value: value}, nil)
}

// remove deletes one key; a missing key is not an error.
func remove(ctx context.Context, c *client.Client, k scopedKey, kind string) error {
	err := c.DoJSON(ctx, http.MethodDelete, k.itemPath(c.Tenant, kind), nil, nil)
	if errors.Is(err, client.ErrNotFound) {
		return nil
	}
	return err
}

// ensureAbsent refuses to create over an existing key, because the per-key PUT is an upsert
// and would silently overwrite a value managed elsewhere (UI, CLI or another configuration).
func ensureAbsent(ctx context.Context, c *client.Client, k scopedKey, kind string) diag.Diagnostics {
	var diags diag.Diagnostics
	_, err := get(ctx, c, k, kind)
	switch {
	case errors.Is(err, client.ErrNotFound):
		return diags
	case err != nil:
		diags.AddError("Unable to check whether "+label(kind)+" exists", err.Error())
	default:
		diags.AddError(label(kind)+" already exists",
			fmt.Sprintf("%s %s already exists. Import it with: terraform import wayfinder_%s.<name> %s",
				label(kind), k.ID(), resourceSuffix(kind), k.ID()))
	}
	return diags
}

func label(kind string) string {
	if kind == kindSecrets {
		return "Secret"
	}
	return "Variable"
}

func resourceSuffix(kind string) string {
	if kind == kindSecrets {
		return "secret"
	}
	return "variable"
}

// clientFrom applies the ProviderData contract from the package documentation.
func clientFrom(data any) (*client.Client, diag.Diagnostics) {
	var diags diag.Diagnostics
	if data == nil {
		return nil, diags
	}
	c, ok := data.(*client.Client)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", data))
	}
	return c, diags
}
