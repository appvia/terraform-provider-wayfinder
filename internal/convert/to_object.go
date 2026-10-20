// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package convert

import (
	"encoding/json"
	"fmt"
	"maps"
	"math/big"
	"slices"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ToOpts selects secrets and supplies the fresh GET object for update overlays.
// IncludeWriteOnly keys are Terraform attribute paths: api_key_wo at the root,
// security.token_wo in objects, rules[0].token_wo in lists, and entries["key"].token_wo
// in maps. Keys name the _wo attribute, never its _wo_version companion.
// Base is never mutated. Unselected secrets retain Base verbatim.
// Prior enables deletion of removed managed values on Update; null/unset keeps
// overlay-only behavior. Blank write-only values keep stored secrets: removing
// a secret is not supported by Update and requires replacement.
type ToOpts struct {
	IncludeWriteOnly map[string]bool
	Base             map[string]any
	Prior            tftypes.Value
}

type writeContext struct {
	include map[string]bool
	path    string
	prior   tftypes.Value
}

// ToObject overlays known configured attributes onto Base, omitting null/unknown
// attributes and never sending framework-only or write-only version attributes.
func ToObject(m *Mapping, cfg tftypes.Value, opts ToOpts) (map[string]any, error) {
	attrs, err := resourceAttrs(m, cfg)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	base, err := cloneJSON(opts.Base)
	if err != nil {
		return nil, fmt.Errorf("base: %w", err)
	}
	out := jsonMap(base)
	meta := jsonMap(out["metadata"])
	prior, err := valueMap(opts.Prior)
	if err != nil {
		return nil, fmt.Errorf("prior: %w", err)
	}
	for _, name := range []string{"labels", "annotations"} {
		if err := removeMapKeys(jsonMap(meta[name]), attrs[name], prior[name]); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
	}
	for _, name := range []string{"name", "workspace", "environment", "version", "labels", "annotations"} {
		v, ok := attrs[name]
		if !ok || v.IsNull() || !v.IsKnown() {
			continue
		}
		n := &node{kind: "string", typ: tftypes.String}
		if name == "labels" || name == "annotations" {
			n.kind = "map"
			n.typ = v.Type()
			n.element = &node{kind: "string", typ: tftypes.String}
		}
		value, err := n.write(v, nil, writeContext{path: name})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if name == "labels" || name == "annotations" {
			merged := jsonMap(meta[name])
			for _, key := range slices.Sorted(maps.Keys(jsonMap(value))) {
				merged[key] = jsonMap(value)[key]
			}
			meta[name] = merged
		} else {
			meta[name] = value
		}
	}
	spec, err := m.spec.writeFields(attrs, jsonMap(out["spec"]), writeContext{include: opts.IncludeWriteOnly, prior: opts.Prior})
	if err != nil {
		return nil, fmt.Errorf("spec: %w", err)
	}
	out["apiVersion"] = m.kind.Group + "/" + m.kind.Version
	out["kind"] = m.kind.Kind
	out["metadata"] = meta
	out["spec"] = spec
	return out, nil
}

func (n *node) writeFields(attrs map[string]tftypes.Value, base map[string]any, ctx writeContext) (map[string]any, error) {
	out := maps.Clone(base)
	prior, err := valueMap(ctx.prior)
	if err != nil {
		return nil, fmt.Errorf("prior fields: %w", err)
	}
	for _, child := range n.children {
		path := child.attr
		if ctx.path != "" {
			path = ctx.path + "." + path
		}
		if child.sensitive && !ctx.include[path] {
			continue
		}
		v, present := attrs[child.attr]
		if !child.required && !child.computed && !child.sensitive && removedValue(v, present, prior[child.attr]) {
			delete(out, child.jsonName)
		}
		if v.IsNull() || !v.IsKnown() {
			continue
		}
		value, err := child.write(v, out[child.jsonName], writeContext{include: ctx.include, path: path, prior: prior[child.attr]})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out[child.jsonName] = value
	}
	return out, nil
}

func (n *node) write(v tftypes.Value, base any, ctx writeContext) (any, error) {
	switch n.kind {
	case "string", "json":
		var text string
		if err := v.As(&text); err != nil {
			return nil, fmt.Errorf("decode string: %w", err)
		}
		if n.kind == "string" {
			return text, nil
		}
		var obj any
		if err := json.Unmarshal([]byte(text), &obj); err != nil {
			return nil, fmt.Errorf("decode JSON: %w", err)
		}
		return obj, nil
	case "boolean":
		var b bool
		if err := v.As(&b); err != nil {
			return nil, fmt.Errorf("decode bool: %w", err)
		}
		return b, nil
	case "integer", "number":
		var b big.Float
		if err := v.As(&b); err != nil {
			return nil, fmt.Errorf("decode number: %w", err)
		}
		if n.kind == "integer" {
			i, accuracy := b.Int64()
			if accuracy != big.Exact {
				return nil, fmt.Errorf("expected int64: %w", ErrValue)
			}
			return i, nil
		}
		return json.Number(b.Text('f', -1)), nil
	case "object", "map":
		var attrs map[string]tftypes.Value
		if err := v.As(&attrs); err != nil {
			return nil, fmt.Errorf("decode object/map: %w", err)
		}
		if n.kind == "object" {
			return n.writeFields(attrs, jsonMap(base), ctx)
		}
		out := map[string]any{}
		prior, err := valueMap(ctx.prior)
		if err != nil {
			return nil, fmt.Errorf("prior map: %w", err)
		}
		if !n.sensitive && ctx.prior.Type() != nil && ctx.prior.IsKnown() {
			out = maps.Clone(jsonMap(base))
			if !n.computed && !n.sensitive {
				if err := removeMapKeys(out, v, ctx.prior); err != nil {
					return nil, err
				}
			}
		}
		for _, key := range slices.Sorted(maps.Keys(attrs)) {
			if attrs[key].IsNull() || !attrs[key].IsKnown() {
				continue
			}
			value, err := n.element.write(attrs[key], jsonMap(base)[key], writeContext{include: ctx.include, path: fmt.Sprintf("%s[%q]", ctx.path, key), prior: prior[key]})
			if err != nil {
				return nil, fmt.Errorf("%s[%q]: %w", ctx.path, key, err)
			}
			out[key] = value
		}
		return out, nil
	case "array":
		var elems []tftypes.Value
		if err := v.As(&elems); err != nil {
			return nil, fmt.Errorf("decode list: %w", err)
		}
		old, _ := base.([]any)
		out := make([]any, 0, len(elems))
		for i, elem := range elems {
			if elem.IsNull() || !elem.IsKnown() {
				continue
			}
			var prior any
			if i < len(old) {
				prior = old[i]
				if ctx.prior.Type() != nil && ctx.prior.IsKnown() {
					prior = n.element.retainedBase(prior)
				}
			}
			value, err := n.element.write(elem, prior, writeContext{include: ctx.include, path: fmt.Sprintf("%s[%d]", ctx.path, i)})
			if err != nil {
				return nil, fmt.Errorf("%s[%d]: %w", ctx.path, i, err)
			}
			out = append(out, value)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported type %s: %w", n.kind, ErrMapping)
	}
}
