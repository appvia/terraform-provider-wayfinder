package main

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// attrChange names one resource attribute in Terraform terms (generator naming).
type attrChange struct {
	Resource, Attribute, Detail string
}

// diffReport is the schema-level change set between two manifests.
type diffReport struct {
	From, To                 string
	AddedKinds, RemovedKinds []string
	Added, Removed           []attrChange
	NowRequired, TypeChanged []attrChange
}

func (r diffReport) breaking() bool {
	return len(r.RemovedKinds)+len(r.Removed)+len(r.NowRequired)+len(r.TypeChanged) > 0
}

func (r diffReport) changed() bool {
	return r.breaking() || len(r.AddedKinds)+len(r.Added) > 0
}

// resourceSchema maps flattened attribute paths to type strings and required flags.
type resourceSchema struct {
	types    map[string]string
	required map[string]bool
}

func diffFiles(oldPath, newPath string) (diffReport, error) {
	oldM, err := manifest.Load(oldPath)
	if err != nil {
		return diffReport{}, err
	}
	newM, err := manifest.Load(newPath)
	if err != nil {
		return diffReport{}, err
	}
	return diffManifests(oldM, newM)
}

func diffManifests(oldM, newM *manifest.Manifest) (diffReport, error) {
	r := diffReport{From: oldM.WayfinderVersion, To: newM.WayfinderVersion}
	oldS, err := resourceSchemas(oldM)
	if err != nil {
		return r, fmt.Errorf("old manifest: %w", err)
	}
	newS, err := resourceSchemas(newM)
	if err != nil {
		return r, fmt.Errorf("new manifest: %w", err)
	}
	for _, res := range slices.Sorted(maps.Keys(oldS)) {
		if _, ok := newS[res]; !ok {
			r.RemovedKinds = append(r.RemovedKinds, res)
		}
	}
	for _, res := range slices.Sorted(maps.Keys(newS)) {
		before, ok := oldS[res]
		if !ok {
			r.AddedKinds = append(r.AddedKinds, res)
			continue
		}
		r.compare(res, before, newS[res])
	}
	return r, nil
}

func (r *diffReport) compare(res string, before, after resourceSchema) {
	for _, attr := range slices.Sorted(maps.Keys(before.types)) {
		if _, ok := after.types[attr]; !ok {
			r.Removed = append(r.Removed, attrChange{Resource: res, Attribute: attr})
		}
	}
	for _, attr := range slices.Sorted(maps.Keys(after.types)) {
		oldType, existed := before.types[attr]
		switch {
		case !existed:
			r.Added = append(r.Added, attrChange{Resource: res, Attribute: attr, Detail: after.types[attr]})
		case oldType != after.types[attr]:
			r.TypeChanged = append(r.TypeChanged, attrChange{Resource: res, Attribute: attr, Detail: oldType + " -> " + after.types[attr]})
		}
		if after.required[attr] && !before.required[attr] {
			r.NowRequired = append(r.NowRequired, attrChange{Resource: res, Attribute: attr})
		}
	}
}

func resourceSchemas(m *manifest.Manifest) (map[string]resourceSchema, error) {
	out := map[string]resourceSchema{}
	for _, k := range m.TypedKinds() {
		mapping, err := convert.NewMapping(k)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k.Kind, err)
		}
		s := resourceSchema{types: map[string]string{}, required: map[string]bool{}}
		flattenType(mapping.TFType(), "", s.types)
		if k.Spec != nil {
			requiredPaths(k.Spec, true, "", s.required)
		}
		out[typeName(k)] = s
	}
	return out, nil
}

func flattenType(obj tftypes.Object, prefix string, out map[string]string) {
	for name, typ := range obj.AttributeTypes {
		path := prefix + name
		out[path] = typeString(typ)
		if nested, ok := nestedObject(typ); ok {
			flattenType(nested, path+".", out)
		}
	}
}

func nestedObject(typ tftypes.Type) (tftypes.Object, bool) {
	switch t := typ.(type) {
	case tftypes.Object:
		return t, true
	case tftypes.List:
		return nestedObject(t.ElementType)
	case tftypes.Map:
		return nestedObject(t.ElementType)
	default:
		return tftypes.Object{}, false
	}
}

func typeString(typ tftypes.Type) string {
	switch t := typ.(type) {
	case tftypes.Object:
		return "object"
	case tftypes.List:
		return "list(" + typeString(t.ElementType) + ")"
	case tftypes.Map:
		return "map(" + typeString(t.ElementType) + ")"
	default:
		switch {
		case typ.Is(tftypes.String):
			return "string"
		case typ.Is(tftypes.Bool):
			return "bool"
		case typ.Is(tftypes.Number):
			return "number"
		default:
			return typ.String()
		}
	}
}

// requiredPaths mirrors the generator's naming (fieldName) over the manifest tree.
func requiredPaths(f *manifest.Field, top bool, prefix string, out map[string]bool) {
	for _, jsonName := range propertyNames(f) {
		child := f.Properties[jsonName]
		path := prefix + fieldName(jsonName, child, top)
		if child.Required {
			out[path] = true
		}
		switch child.Type {
		case "object":
			requiredPaths(child, false, path+".", out)
		case "array", "map":
			if el := element(child); el.Type == "object" {
				requiredPaths(el, false, path+".", out)
			}
		}
	}
}

// write renders the report as Markdown; the first line is BREAKING when any
// attribute or resource was removed, became required, or changed type.
func (r diffReport) write(w io.Writer) {
	var b strings.Builder
	if r.breaking() {
		b.WriteString("BREAKING\n\n")
	}
	fmt.Fprintf(&b, "## Wayfinder %s -> %s: Terraform schema changes\n", r.From, r.To)
	if !r.changed() {
		b.WriteString("\nNo resource schema changes.\n")
	}
	section := func(title string, items []string) {
		if len(items) == 0 {
			return
		}
		fmt.Fprintf(&b, "\n### %s\n\n", title)
		for _, item := range items {
			fmt.Fprintf(&b, "- %s\n", item)
		}
	}
	attrs := func(changes []attrChange) []string {
		items := make([]string, 0, len(changes))
		for _, c := range changes {
			item := fmt.Sprintf("`%s`: `%s`", c.Resource, c.Attribute)
			if c.Detail != "" {
				item += " (" + c.Detail + ")"
			}
			items = append(items, item)
		}
		return items
	}
	section("Removed resources (breaking)", r.RemovedKinds)
	section("Removed attributes (breaking)", attrs(r.Removed))
	section("Attributes that became required (breaking)", attrs(r.NowRequired))
	section("Attribute type changes (breaking)", attrs(r.TypeChanged))
	section("Added resources", r.AddedKinds)
	section("Added attributes", attrs(r.Added))
	_, _ = io.WriteString(w, b.String())
}
