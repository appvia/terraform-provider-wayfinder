// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package convert

import (
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
)

// AttributePath maps dotted API validation fields, including collection indexes,
// to their compiled Terraform names. Unknown fields are not silently renamed.
func (m *Mapping) AttributePath(field string) (path.Path, bool) {
	if p, ok := m.identityPath(field); ok {
		return p, true
	}

	parts := strings.Split(field, ".")
	if len(parts) < 2 {
		return path.Empty(), false
	}

	if parts[0] != "spec" {
		return path.Empty(), false
	}

	p := path.Empty()
	n := m.spec

	for _, part := range parts[1:] {
		name, suffix, _ := strings.Cut(part, "[")
		if n.kind == "map" {
			p, n = p.AtMapKey(name), n.element
		} else {
			var found *node

			for _, child := range n.children {
				if child.jsonName == name {
					found = child
					break
				}
			}

			if found == nil {
				return path.Empty(), false
			}

			p, n = p.AtName(found.attr), found
		}

		for suffix != "" {
			index, rest, ok := strings.Cut(suffix, "]")
			if !ok {
				return path.Empty(), false
			}

			switch n.kind {
			case "array":
				i, err := strconv.Atoi(index)
				if err != nil || i < 0 {
					return path.Empty(), false
				}

				p = p.AtListIndex(i)
			case "map":
				key, err := strconv.Unquote(index)
				if err != nil {
					return path.Empty(), false
				}

				p = p.AtMapKey(key)
			default:
				return path.Empty(), false
			}

			n = n.element
			suffix = strings.TrimPrefix(rest, "[")
		}
	}

	return p, true
}

// identityPath maps bare or metadata.-prefixed identity fields and label/annotation keys.
// Map keys may contain dots and slashes, so everything after the map name is the key.
func (m *Mapping) identityPath(field string) (path.Path, bool) {
	field = strings.TrimPrefix(field, "metadata.")

	name, key, hasKey := strings.Cut(field, "[")
	if hasKey {
		unquoted, err := strconv.Unquote(strings.TrimSuffix(key, "]"))
		if err != nil {
			return path.Empty(), false
		}

		key = unquoted
	} else {
		name, key, hasKey = strings.Cut(field, ".")
	}

	switch name {
	case "name", "workspace", "environment", "version":
		if hasKey {
			return path.Empty(), false
		}
	case "labels", "annotations":
	default:
		return path.Empty(), false
	}

	if _, ok := m.typ.AttributeTypes[name]; !ok {
		return path.Empty(), false
	}

	p := path.Root(name)
	if hasKey && key != "" {
		p = p.AtMapKey(key)
	}

	return p, true
}
