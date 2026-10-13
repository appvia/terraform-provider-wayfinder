package convert

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type readContext struct {
	prior    tftypes.Value
	computed bool
}

// FromObject reads manifest fields with prior-aware drift filtering. Secrets are
// always null; version companions and framework-only inputs are retained from
// prior. Optional noncomputed values are read only where prior configured them.
func FromObject(m *Mapping, obj map[string]any, prior tftypes.Value) (tftypes.Value, error) {
	attrs, err := resourceAttrs(m, prior)
	if err != nil {
		return tftypes.Value{}, fmt.Errorf("prior: %w", err)
	}
	out := map[string]tftypes.Value{}
	for _, name := range slices.Sorted(maps.Keys(m.typ.AttributeTypes)) {
		out[name] = tftypes.NewValue(m.typ.AttributeTypes[name], nil)
	}
	meta := jsonMap(obj["metadata"])
	for _, name := range []string{"name", "workspace", "environment", "version"} {
		typ, exists := m.typ.AttributeTypes[name]
		if !exists {
			continue
		}
		v, err := readScalar(typ, meta[name])
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("metadata.%s: %w", name, err)
		}
		out[name] = v
	}
	for _, name := range []string{"labels", "annotations"} {
		keys, err := valueMap(attrs[name])
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("%s: %w", name, err)
		}
		if attrs[name].IsNull() || !attrs[name].IsKnown() {
			continue
		}
		filtered := map[string]any{}
		for _, key := range slices.Sorted(maps.Keys(keys)) {
			if value, ok := jsonMap(meta[name])[key]; ok {
				filtered[key] = value
			}
		}
		v, err := readScalar(m.typ.AttributeTypes[name], filtered)
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("metadata.%s: %w", name, err)
		}
		out[name] = v
	}
	spec, err := m.spec.readFields(jsonMap(obj["spec"]), readContext{prior: prior})
	if err != nil {
		return tftypes.Value{}, fmt.Errorf("spec: %w", err)
	}
	for name, v := range spec {
		out[name] = v
	}
	for _, name := range []string{"timeouts", "wait_for_ready"} {
		if v, exists := attrs[name]; exists {
			out[name] = v.Copy()
		}
	}
	identity := Identity{}
	identity.Name, _ = meta["name"].(string)
	identity.Workspace, _ = meta["workspace"].(string)
	identity.Environment, _ = meta["environment"].(string)
	identity.Version, _ = meta["version"].(string)
	if identity.Name != "" {
		out["id"] = tftypes.NewValue(tftypes.String, identity.ID())
	}
	if status, ok := StatusJSON(obj); ok {
		out["status_json"] = tftypes.NewValue(tftypes.String, status)
		s, message, detail := StatusSummary(obj)
		out["status"] = tftypes.NewValue(m.typ.AttributeTypes["status"], map[string]tftypes.Value{"status": tftypes.NewValue(tftypes.String, s), "message": tftypes.NewValue(tftypes.String, message), "detail": tftypes.NewValue(tftypes.String, detail)})
	}
	return newValue(m.typ, out)
}

func (n *node) readFields(obj map[string]any, ctx readContext) (map[string]tftypes.Value, error) {
	prior, err := valueMap(ctx.prior)
	if err != nil {
		return nil, fmt.Errorf("prior object: %w", err)
	}
	out := map[string]tftypes.Value{}
	for _, child := range n.children {
		v := tftypes.NewValue(child.typ, nil)
		if !child.sensitive {
			v, err = child.read(obj[child.jsonName], readContext{prior: prior[child.attr], computed: ctx.computed})
			if err != nil {
				return nil, fmt.Errorf("%s: %w", child.attr, err)
			}
		} else {
			version := tftypes.NewValue(tftypes.String, nil)
			if p, ok := prior[child.attr+"_version"]; ok {
				version = p.Copy()
			}
			out[child.attr+"_version"] = version
		}
		out[child.attr] = v
	}
	return out, nil
}

func (n *node) read(obj any, ctx readContext) (tftypes.Value, error) {
	configured := ctx.prior.Type() != nil && !ctx.prior.IsNull()
	if obj == nil || (!configured && !n.required && !n.computed && !ctx.computed) {
		return tftypes.NewValue(n.typ, nil), nil
	}
	switch n.kind {
	case "json":
		b, err := json.Marshal(obj)
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("encode JSON: %w", err)
		}
		return tftypes.NewValue(tftypes.String, string(b)), nil
	case "string", "boolean", "integer", "number":
		v, err := readScalar(n.typ, obj)
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("decode %s: %w", n.kind, err)
		}
		if n.kind == "integer" {
			if _, err := n.write(v, nil, writeContext{}); err != nil {
				return tftypes.Value{}, fmt.Errorf("decode integer: %w", err)
			}
		}
		return v, nil
	case "object":
		m, ok := obj.(map[string]any)
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected JSON object: %w", ErrValue)
		}
		attrs, err := n.readFields(m, readContext{prior: ctx.prior, computed: ctx.computed || n.computed})
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("decode object: %w", err)
		}
		return newValue(n.typ, attrs)
	case "map":
		m, ok := obj.(map[string]any)
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected JSON map: %w", ErrValue)
		}
		prior, err := valueMap(ctx.prior)
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("prior map: %w", err)
		}
		attrs := map[string]tftypes.Value{}
		for _, key := range slices.Sorted(maps.Keys(m)) {
			v, err := n.element.read(m[key], readContext{prior: prior[key], computed: ctx.computed || n.computed})
			if err != nil {
				return tftypes.Value{}, fmt.Errorf("[%q]: %w", key, err)
			}
			attrs[key] = v
		}
		return newValue(n.typ, attrs)
	case "array":
		list, ok := obj.([]any)
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected JSON list: %w", ErrValue)
		}
		var old []tftypes.Value
		if configured && ctx.prior.IsKnown() {
			if err := ctx.prior.As(&old); err != nil {
				return tftypes.Value{}, fmt.Errorf("prior list: %w", err)
			}
		}
		out := make([]tftypes.Value, len(list))
		for i, obj := range list {
			var p tftypes.Value
			if i < len(old) {
				p = old[i]
			}
			v, err := n.element.read(obj, readContext{prior: p, computed: ctx.computed || n.computed || (configured && n.element.kind != "object")})
			if err != nil {
				return tftypes.Value{}, fmt.Errorf("[%d]: %w", i, err)
			}
			out[i] = v
		}
		return newValue(n.typ, out)
	default:
		return tftypes.Value{}, fmt.Errorf("unsupported type %s: %w", n.kind, ErrMapping)
	}
}
