package manifestresource

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

// secretValue is a non-empty value configured at a sensitive manifest path.
type secretValue struct {
	// path addresses the value in the manifest, e.g. spec.bearerToken.token or spec.rules[0].token.
	path string
	// attr is the write-only attribute of the kind's dedicated resource, e.g. bearer_token.token_wo.
	attr string
}

// secretValues lists non-empty values set at sensitive paths of the kind, sorted by path.
// Blank values are allowed: Wayfinder keeps the stored secret when a blank one is sent.
func secretValues(k manifest.Kind, obj map[string]any) []secretValue {
	var out []secretValue
	if k.Spec != nil {
		walkSecrets(k.Spec, obj["spec"], walkPos{path: "spec", top: true}, &out)
	}
	slices.SortFunc(out, func(a, b secretValue) int { return cmp.Compare(a.path, b.path) })
	return out
}

type walkPos struct {
	path, attr string
	top        bool
}

func walkSecrets(f *manifest.Field, v any, at walkPos, out *[]secretValue) {
	switch f.Type {
	case "object":
		m, _ := v.(map[string]any)
		for name, child := range m {
			field, known := f.Properties[name]
			if !known {
				continue
			}
			next := walkPos{path: at.path + "." + name, attr: childAttr(at, name, field)}
			if field.Sensitive {
				if isSet(child) {
					*out = append(*out, secretValue{path: next.path, attr: next.attr + "_wo"})
				}
				continue
			}
			walkSecrets(field, child, next, out)
		}
	case "array":
		l, _ := v.([]any)
		for i, child := range l {
			walkSecrets(f.Items, child, walkPos{path: at.path + "[" + strconv.Itoa(i) + "]", attr: at.attr}, out)
		}
	case "map":
		m, _ := v.(map[string]any)
		for key, child := range m {
			walkSecrets(f.Values, child, walkPos{path: at.path + "[" + strconv.Quote(key) + "]", attr: at.attr}, out)
		}
	}
}

// childAttr follows the generated resources' naming (D-MAP/D-WO): top-level spec fields use
// convert.AttrName, nested ones snake_case; json fields get a _json suffix.
func childAttr(at walkPos, name string, f *manifest.Field) string {
	attr := convert.SnakeCase(name)
	if at.top {
		attr = convert.AttrName(name)
	}
	if f.Type == "json" {
		attr += "_json"
	}
	if at.attr == "" {
		return attr
	}
	return at.attr + "." + attr
}

func isSet(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case map[string]any:
		for _, child := range t {
			if isSet(child) {
				return true
			}
		}
		return false
	case []any:
		return slices.ContainsFunc(t, isSet)
	default:
		return true
	}
}

// secretMessage tells the user where the secret belongs instead.
func secretMessage(k manifest.Kind, s secretValue) string {
	if k.Typed && k.TerraformName != "" {
		return fmt.Sprintf("%s is a secret. Set it with the wayfinder_%s.%s attribute, or in manifest_wo, so it is never stored in state.", s.path, k.TerraformName, s.attr)
	}
	return fmt.Sprintf("%s is a secret, and wayfinder_manifest never stores secrets in state. Set it in manifest_wo instead, leaving it out of manifest or blank (\"\").", s.path)
}
