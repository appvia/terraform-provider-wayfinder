package fakekinds

import "github.com/appvia/terraform-provider-wayfinder/internal/manifest"

// ServerDefaults returns, for every computed scalar or scalar-list spec field whose parent
// objects are always present (spec itself or Required objects), the value the fake server
// should fill in when the client leaves it unset: the first enum value, "server-default",
// true, 1, or a one-element list of those. Fields under an Optional object, an array or a map
// are left out, because fakeapi.KindSpec.Defaults creates missing parents, which a real server
// does not do for an absent optional block. Assign the result to fakeapi.KindSpec.Defaults.
func ServerDefaults(k manifest.Kind) map[string]any {
	out := map[string]any{}
	if k.Spec != nil {
		collectDefaults(out, "spec", k.Spec)
	}
	return out
}

func collectDefaults(out map[string]any, path string, f *manifest.Field) {
	for name, child := range f.Properties {
		p := path + "." + name
		switch {
		case child.Computed && !child.Sensitive:
			if v, ok := defaultValue(child); ok {
				out[p] = v
			}
		case child.Type == "object" && child.Required:
			collectDefaults(out, p, child)
		}
	}
}

func defaultValue(f *manifest.Field) (any, bool) {
	if len(f.Enum) > 0 {
		return f.Enum[0], true
	}
	switch f.Type {
	case "string":
		return "server-default", true
	case "boolean":
		return true, true
	case "integer", "number":
		return 1, true
	case "array":
		v, ok := defaultValue(f.Items)
		return []any{v}, ok
	default:
		return nil, false
	}
}
