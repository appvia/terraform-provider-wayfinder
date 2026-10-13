package main

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

// synth renders HCL attribute values for a kind's spec. In example mode it emits only Required
// fields; in fixture mode it also emits every write-only field (and the containers leading to
// it), so the generated plan-stability tests exercise each secret. Every write-only value is a
// deterministic literal unique per Terraform type and attribute path.
type synth struct {
	typ     string
	fixture bool
	// secrets are the write-only literals emitted; writeOnly are their flatmap state paths.
	secrets, writeOnly []string
	// problems are required fields no generated validator-compliant value was found for.
	problems []string
}

// stringCandidates are tried in order; the first one passing the field's pattern and length
// constraints is used.
var stringCandidates = []string{"example", "https://example.com", "example.tf", "example-1", "example_1", "1.0.0", "a"}

func (s *synth) include(f *manifest.Field) bool {
	if f.Required && (!f.Computed || f.Sensitive) {
		return true
	}
	return s.fixture && !f.Computed && (f.Sensitive || containsSecretChildren(f))
}

// body renders the included properties of an object, one per line at indent.
func (s *synth) body(f *manifest.Field, top bool, indent, path string) string {
	var b strings.Builder
	for _, jsonName := range propertyNames(f) {
		child := f.Properties[jsonName]
		if !s.include(child) {
			continue
		}
		name := fieldName(jsonName, child, top)
		childPath := joinPath(path, name)
		if child.Sensitive {
			if !s.fixture {
				fmt.Fprintf(&b, "%s# Replace this write-only placeholder with an ephemeral secret.\n", indent)
			}
			fmt.Fprintf(&b, "%s%s = %s\n%s%s_version = \"1\"\n", indent, name, s.secret(child, childPath), indent, name)
			continue
		}
		fmt.Fprintf(&b, "%s%s = %s\n", indent, name, s.value(child, indent, childPath))
	}
	return b.String()
}

func (s *synth) secret(f *manifest.Field, path string) string {
	literal := strings.NewReplacer(".", "-", "_", "-").Replace("wo-" + strings.TrimPrefix(s.typ, "wayfinder_") + "-" + path)
	s.secrets = append(s.secrets, literal)
	s.writeOnly = append(s.writeOnly, path)
	if f.Type == "map" {
		return fmt.Sprintf("{ example = %q }", literal)
	}
	if !matchesConstraints(f, literal) {
		s.problems = append(s.problems, fmt.Sprintf("%s: write-only literal does not satisfy the field constraints", path))
	}
	return strconv.Quote(literal)
}

func (s *synth) value(f *manifest.Field, indent, path string) string {
	if len(f.Enum) > 0 {
		return strconv.Quote(f.Enum[0])
	}
	switch f.Type {
	case "string":
		return strconv.Quote(s.str(f, path))
	case "json":
		return "jsonencode({})"
	case "boolean":
		return "true"
	case "integer", "number":
		return s.number(f, path)
	case "array":
		return "[" + s.value(f.Items, indent, path+".0") + "]"
	case "map":
		inner := indent + "  "
		return "{\n" + inner + "example = " + s.value(f.Values, inner, path+".example") + "\n" + indent + "}"
	case "object":
		return "{\n" + s.body(f, false, indent+"  ", path) + indent + "}"
	default:
		s.problems = append(s.problems, fmt.Sprintf("%s: unsupported type %q", path, f.Type))
		return "null"
	}
}

func (s *synth) str(f *manifest.Field, path string) string {
	candidates := stringCandidates
	switch f.Format {
	case "date-time":
		candidates = append([]string{"2030-01-01T00:00:00Z"}, candidates...)
	case "uri", "url":
		candidates = append([]string{"https://example.com"}, candidates...)
	}
	for _, c := range candidates {
		if matchesConstraints(f, c) {
			return c
		}
	}
	s.problems = append(s.problems, fmt.Sprintf("%s: no synthesized literal satisfies pattern %q and length limits", path, f.Pattern))
	return candidates[0]
}

func matchesConstraints(f *manifest.Field, v string) bool {
	if (f.MinLength != nil && len(v) < *f.MinLength) || (f.MaxLength != nil && len(v) > *f.MaxLength) {
		return false
	}
	if f.Pattern == "" {
		return true
	}
	re, err := regexp.Compile(f.Pattern)
	return err == nil && re.MatchString(v)
}

func (s *synth) number(f *manifest.Field, path string) string {
	v := 1.0
	if f.Minimum != nil && *f.Minimum > v {
		v = math.Ceil(*f.Minimum)
	}
	if f.Maximum != nil && *f.Maximum < v {
		s.problems = append(s.problems, fmt.Sprintf("%s: no integer at or above the minimum is within maximum %v", path, *f.Maximum))
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func joinPath(parent, name string) string {
	if parent == "" {
		return name
	}
	return parent + "." + name
}
