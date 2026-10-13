package convert

import (
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
)

// AttributePath maps dotted API validation fields, including collection indexes,
// to their compiled Terraform names. Unknown fields are not silently renamed.
func (m *Mapping) AttributePath(field string) (path.Path, bool) {
	parts := strings.Split(field, ".")
	if len(parts) < 2 {
		return path.Empty(), false
	}
	if parts[0] == "metadata" {
		if _, ok := m.typ.AttributeTypes[parts[1]]; !ok {
			return path.Empty(), false
		}
		p := path.Root(parts[1])
		for _, key := range parts[2:] {
			p = p.AtMapKey(key)
		}
		return p, true
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
