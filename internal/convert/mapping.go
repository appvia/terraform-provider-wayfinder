// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

// Package convert maps manifest-defined Terraform values to Wayfinder JSON.
package convert

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	ErrMapping = errors.New("invalid manifest mapping")
	ErrValue   = errors.New("invalid conversion value")
)

// Mapping is an immutable, compiled per-kind mapping. Construct it once and reuse.
type Mapping struct {
	kind manifest.Kind
	spec *node
	typ  tftypes.Object
}

type node struct {
	jsonName                      string
	attr                          string
	kind                          string
	required, computed, sensitive bool
	typ                           tftypes.Type
	children                      []*node
	element                       *node
}

// SnakeCase preserves acronym boundaries: HTTPServer becomes http_server.
func SnakeCase(name string) string {
	runes := []rune(name)
	var out strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 && runes[i-1] != '_' &&
			(unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1]) ||
				(i+1 < len(runes) && unicode.IsLower(runes[i+1]) && !pluralAcronym(runes, i))) {
			out.WriteByte('_')
		}
		out.WriteRune(unicode.ToLower(r))
	}
	return out.String()
}

// pluralAcronym reports whether the upper-case rune at i ends an acronym pluralised with a
// lone "s" (ARNs, IDs, VPCs), so the "s" stays in the acronym's word.
func pluralAcronym(runes []rune, i int) bool {
	return unicode.IsUpper(runes[i-1]) && runes[i+1] == 's' &&
		(i+2 == len(runes) || !unicode.IsLower(runes[i+2]))
}

// IsReserved reports collisions with provider identity attributes and Terraform
// resource meta-arguments (framework v1.19 ReservedResourceAttributeNames), plus
// locals reserved by the provider's naming contract.
func IsReserved(name string) bool {
	switch name {
	case "id", "name", "workspace", "environment", "version", "labels", "annotations",
		"status", "status_json", "timeouts", "wait_for_ready",
		"connection", "count", "depends_on", "for_each", "lifecycle", "provider", "provisioner", "locals":
		return true
	default:
		return false
	}
}

// AttrName maps a top-level spec JSON name, before adding _json or _wo suffixes.
// Provider identity and Terraform meta-argument collisions gain a spec_ prefix.
// Nested properties use SnakeCase instead (no resource-level collision prefix).
func AttrName(jsonName string) string {
	name := SnakeCase(jsonName)
	if IsReserved(name) {
		return "spec_" + name
	}
	return name
}

// NewMapping compiles names and types recursively, rejecting ambiguous names,
// unsupported fields and cycles (cycles must be represented as manifest json).
func NewMapping(kind manifest.Kind) (*Mapping, error) {
	spec := kind.Spec
	if spec == nil {
		spec = &manifest.Field{Type: "object"}
	}
	if spec.Type != "object" {
		return nil, fmt.Errorf("spec: expected object: %w", ErrMapping)
	}
	n, err := compile(spec, true, map[*manifest.Field]bool{})
	if err != nil {
		return nil, fmt.Errorf("spec: %w", err)
	}
	attrs := map[string]tftypes.Type{
		"id": tftypes.String, "name": tftypes.String,
		"labels":      tftypes.Map{ElementType: tftypes.String},
		"annotations": tftypes.Map{ElementType: tftypes.String},
		"status":      tftypes.Object{AttributeTypes: map[string]tftypes.Type{"status": tftypes.String, "message": tftypes.String, "detail": tftypes.String}},
		"status_json": tftypes.String,
		"timeouts":    tftypes.Object{AttributeTypes: map[string]tftypes.Type{"create": tftypes.String, "update": tftypes.String, "delete": tftypes.String}},
	}
	if slices.Contains(kind.Scopes, "workspace") || slices.Contains(kind.Scopes, "environment") {
		attrs["workspace"] = tftypes.String
	}
	if slices.Contains(kind.Scopes, "environment") {
		attrs["environment"] = tftypes.String
	}
	if kind.Versioned {
		attrs["version"] = tftypes.String
	}
	if kind.HasStatus {
		attrs["wait_for_ready"] = tftypes.Bool
	}
	for name, typ := range n.typ.(tftypes.Object).AttributeTypes {
		if _, exists := attrs[name]; exists {
			return nil, fmt.Errorf("spec.%s: attribute collision: %w", name, ErrMapping)
		}
		attrs[name] = typ
	}
	kind.Scopes = slices.Clone(kind.Scopes)
	kind.Spec = nil
	return &Mapping{kind: kind, spec: n, typ: tftypes.Object{AttributeTypes: attrs}}, nil
}

// TFType returns an independent copy of the resource object type. Always present:
// id/name strings, labels/annotations string maps, status object with string
// status/message/detail, status_json string, timeouts object with string
// create/update/delete. workspace is present for workspace OR environment scopes;
// environment for environment scopes; version for versioned kinds; wait_for_ready
// bool only for HasStatus. Flattened spec attributes use AttrName, then _json for
// json fields, then _wo for sensitive fields plus an optional string _wo_version.
// Nested objects apply the same suffix rules but use SnakeCase, not AttrName.
// Scalars: string/bool/Number (both integer and number); arrays: List; maps: Map.
func (m *Mapping) TFType() tftypes.Object { return copyType(m.typ).(tftypes.Object) }

func copyType(typ tftypes.Type) tftypes.Type {
	switch t := typ.(type) {
	case tftypes.Object:
		attrs := make(map[string]tftypes.Type, len(t.AttributeTypes))
		for name, child := range t.AttributeTypes {
			attrs[name] = copyType(child)
		}
		return tftypes.Object{AttributeTypes: attrs}
	case tftypes.List:
		return tftypes.List{ElementType: copyType(t.ElementType)}
	case tftypes.Map:
		return tftypes.Map{ElementType: copyType(t.ElementType)}
	default:
		return typ
	}
}

func compile(f *manifest.Field, top bool, ancestors map[*manifest.Field]bool) (*node, error) {
	if f == nil {
		return nil, fmt.Errorf("missing field: %w", ErrMapping)
	}
	if ancestors[f] {
		return nil, fmt.Errorf("recursive field must use json: %w", ErrMapping)
	}
	ancestors[f] = true
	defer delete(ancestors, f)
	n := &node{kind: f.Type, required: f.Required, computed: f.Computed, sensitive: f.Sensitive}
	switch f.Type {
	case "string", "json":
		n.typ = tftypes.String
	case "boolean":
		n.typ = tftypes.Bool
	case "integer", "number":
		n.typ = tftypes.Number
	case "object":
		attrs := map[string]tftypes.Type{}
		for _, name := range slices.Sorted(maps.Keys(f.Properties)) {
			child, err := compile(f.Properties[name], false, ancestors)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			child.jsonName = name
			child.attr = SnakeCase(name)
			if top {
				child.attr = AttrName(name)
			}
			if child.kind == "json" {
				child.attr += "_json"
			}
			if child.sensitive {
				child.attr += "_wo"
			}
			for _, attr := range child.attrs() {
				if _, exists := attrs[attr]; exists {
					return nil, fmt.Errorf("%s: attribute %s collides: %w", name, attr, ErrMapping)
				}
				attrs[attr] = child.typ
				if attr != child.attr {
					attrs[attr] = tftypes.String
				}
			}
			n.children = append(n.children, child)
		}
		n.typ = tftypes.Object{AttributeTypes: attrs}
	case "array", "map":
		field := f.Items
		if f.Type == "map" {
			field = f.Values
		}
		child, err := compile(field, false, ancestors)
		if err != nil {
			return nil, fmt.Errorf("element: %w", err)
		}
		n.element = child
		child.required = true
		if f.Type == "array" {
			n.typ = tftypes.List{ElementType: child.typ}
		} else {
			n.typ = tftypes.Map{ElementType: child.typ}
		}
	default:
		return nil, fmt.Errorf("unsupported type %q: %w", f.Type, ErrMapping)
	}
	return n, nil
}

func (n *node) attrs() []string {
	if n.sensitive {
		return []string{n.attr, n.attr + "_version"}
	}
	return []string{n.attr}
}
