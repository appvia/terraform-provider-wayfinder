package manifestresource

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"sigs.k8s.io/yaml"
)

type woProblem struct {
	attr, summary, detail string
}

var (
	yamlLinePattern = regexp.MustCompile(`line (\d+)`)
	woForbiddenKeys = []string{"apiVersion", "kind", "metadata", "status"}
)

// The YAML parser's error is never copied into a problem: it can echo the secret value.
func parseWriteOnly(text string) (map[string]any, []woProblem) {
	docs := 0
	for _, doc := range documentSep.Split(text, -1) {
		if strings.TrimSpace(doc) != "" {
			docs++
		}
	}
	if docs > 1 {
		return nil, []woProblem{{
			attr:    "manifest_wo",
			summary: "Invalid manifest_wo",
			detail:  fmt.Sprintf("manifest_wo holds %d YAML documents; it must be one object.", docs),
		}}
	}
	raw, err := yaml.YAMLToJSON([]byte(text))
	if err != nil {
		detail := "manifest_wo is not valid YAML or JSON"
		if m := yamlLinePattern.FindStringSubmatch(err.Error()); m != nil {
			detail += " (line " + m[1] + ")"
		}
		return nil, []woProblem{{attr: "manifest_wo", summary: "Invalid manifest_wo", detail: detail}}
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return nil, []woProblem{{
			attr:    "manifest_wo",
			summary: "Invalid manifest_wo",
			detail:  "manifest_wo must be one YAML or JSON object.",
		}}
	}
	obj, _ = withoutNulls(obj).(map[string]any)
	var problems []woProblem
	for _, key := range woForbiddenKeys {
		if _, set := obj[key]; set {
			problems = append(problems, woProblem{
				attr:    "manifest_wo",
				summary: "Invalid manifest_wo",
				detail:  fmt.Sprintf("manifest_wo must not set %s; set it in manifest.", key),
			})
		}
	}
	return obj, problems
}

// mergeWriteOnly applies wo to a private copy, allowing blank placeholders only at secret paths.
func mergeWriteOnly(k manifest.Kind, base, wo map[string]any) (map[string]any, []woProblem) {
	var problems []woProblem
	var merge func(any, any, bool, woPath) any
	merge = func(base, value any, exists bool, p woPath) any {
		if value == nil {
			return base
		}
		if !exists {
			return deepCopy(value)
		}
		switch b := base.(type) {
		case map[string]any:
			if w, ok := value.(map[string]any); ok {
				keys := make([]string, 0, len(w))
				for key := range w {
					keys = append(keys, key)
				}
				slices.Sort(keys)
				for _, key := range keys {
					child, present := b[key]
					if w[key] != nil {
						b[key] = merge(child, w[key], present, append(p, segment{key: key}))
					}
				}
				return b
			}
		case []any:
			if w, ok := value.([]any); ok {
				if len(b) != len(w) {
					problems = append(problems, woProblem{
						attr:    "manifest_wo",
						summary: "Invalid manifest_wo",
						detail:  fmt.Sprintf("%s has %d items in manifest_wo but %d in manifest; give both the same length, using null for items with nothing to add.", p, len(w), len(b)),
					})
					return b
				}
				for i := range w {
					b[i] = merge(b[i], w[i], true, append(p, segment{index: i, isIndex: true}))
				}
				return b
			}
		}
		if !isSet(base) && isSecretPath(k, p) {
			return deepCopy(value)
		}
		problems = append(problems, woProblem{
			attr:    "manifest_wo",
			summary: "Conflicting manifest_wo value",
			detail:  fmt.Sprintf("%s is set in both manifest and manifest_wo. Set it in one place; manifest may keep a blank placeholder only for secret fields.", p),
		})
		return base
	}
	out := merge(deepCopy(base), wo, true, nil).(map[string]any)
	return out, problems
}

// isSecretPath includes sensitive ancestors; a kind without a spec cannot classify paths.
func isSecretPath(k manifest.Kind, p woPath) bool {
	if k.Spec == nil {
		return true
	}
	if len(p) == 0 || p[0].isIndex || p[0].key != "spec" {
		return false
	}
	field := k.Spec
	for _, seg := range p[1:] {
		if field.Sensitive {
			return true
		}
		switch field.Type {
		case "object":
			if seg.isIndex {
				return false
			}
			field = field.Properties[seg.key]
		case "array":
			if !seg.isIndex {
				return false
			}
			field = field.Items
		case "map":
			if seg.isIndex {
				return false
			}
			field = field.Values
		default:
			return false
		}
		if field == nil {
			return false
		}
	}
	return field.Sensitive
}
