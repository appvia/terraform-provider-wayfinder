// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/engine"
)

func integrationRef(name string) client.Ref {
	return client.Ref{Group: "integrations", Version: "v3", APIPath: "integrations", Name: "wayfinder." + name}
}

func enablePath(tenant, name string) string {
	return "/api/v3/general/tenants/" + url.PathEscape(tenant) + "/integrations/" + url.PathEscape(name) + "/enable"
}

func enable(ctx context.Context, c *client.Client, name string, params map[string]json.RawMessage) (map[string]any, error) {
	body := struct {
		Params map[string]json.RawMessage `json:"params,omitempty"`
	}{Params: params}

	var out map[string]any
	if err := c.DoJSON(ctx, http.MethodPost, enablePath(c.Tenant, name), body, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func addAPIError(d *diag.Diagnostics, err error, name string) {
	if errors.Is(err, client.ErrNotFound) {
		detail := err.Error()
		if !strings.HasSuffix(detail, ".") && !strings.HasSuffix(detail, "!") && !strings.HasSuffix(detail, "?") {
			detail += "."
		}

		d.AddAttributeError(path.Root("name"), "Unknown feature integration",
			detail+" Feature integrations available to enable are listed by `wf enable integration` with no name.")

		return
	}

	engine.AddAPIError(d, err, engine.ErrorTarget{
		Kind: "feature integration", ID: "wayfinder." + name, Address: "wayfinder_feature_integration.<name>",
		AttributePath: fieldPath,
	})
}

func fieldPath(field string) (path.Path, bool) {
	switch {
	case field == "name":
		return path.Root("name"), true
	case field == "spec.params":
		return path.Root("params"), true
	case strings.HasPrefix(field, "spec.params."):
		return path.Root("params").AtMapKey(strings.TrimPrefix(field, "spec.params.")), true
	default:
		return path.Path{}, false
	}
}

func projectParams(prior types.Map, obj map[string]any, keys []string) (types.Map, diag.Diagnostics) {
	if prior.IsNull() && len(keys) == 0 {
		return types.MapNull(jsontypes.NormalizedType{}), nil
	}

	spec, _ := obj["spec"].(map[string]any)
	params, _ := spec["params"].(map[string]any)
	values := make(map[string]attr.Value, len(keys))

	var diags diag.Diagnostics

	for _, key := range keys {
		value, ok := params[key]
		if !ok {
			continue
		}

		raw, err := json.Marshal(value)
		if err != nil {
			diags.AddAttributeError(path.Root("params").AtMapKey(key), "Unable to read feature parameter", err.Error())
			continue
		}

		values[key] = jsontypes.NewNormalizedValue(string(raw))
	}

	result, mapDiags := types.MapValue(jsontypes.NormalizedType{}, values)
	diags.Append(mapDiags...)

	return result, diags
}

func (m *featureModel) setObject(obj map[string]any) {
	spec, _ := obj["spec"].(map[string]any)
	enabled, _ := spec["enabled"].(bool)
	typ, _ := spec["type"].(string)
	m.ID = types.StringValue("wayfinder." + m.Name.ValueString())
	m.Enabled = types.BoolValue(enabled)
	m.Type = types.StringValue(typ)
}
