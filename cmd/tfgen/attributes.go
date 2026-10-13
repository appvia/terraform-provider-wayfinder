package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

func fieldName(name string, f *manifest.Field, top bool) string {
	n := convert.SnakeCase(name)
	if top {
		n = convert.AttrName(name)
	}
	if f.Type == "json" {
		n += "_json"
	}
	if f.Sensitive {
		n += "_wo"
	}
	return n
}

func (e *emitter) properties(f *manifest.Field, top bool, path string) (string, error) {
	var b strings.Builder
	for _, jsonName := range propertyNames(f) {
		child := f.Properties[jsonName]
		if e.dataSource && child.Sensitive {
			continue
		}
		name := fieldName(jsonName, child, top)
		if err := safeText(name); err != nil {
			return "", fmt.Errorf("%s.%s: %w", path, jsonName, err)
		}
		value, err := e.attribute(child, path+"."+jsonName)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "%q: %s,\n", name, value)
		if child.Sensitive {
			fmt.Fprintf(&b, "%q: %s.StringAttribute{Optional:true, MarkdownDescription:%q},\n", name+"_version", e.pkg(), "Change this value to send a new "+name+" to Wayfinder.")
		}
	}
	return b.String(), nil
}

func (e *emitter) attribute(f *manifest.Field, path string) (string, error) {
	if f.Pattern != "" {
		if _, err := regexp.Compile(f.Pattern); err != nil {
			return "", fmt.Errorf("%s: invalid pattern: %w", path, err)
		}
	}
	for _, value := range f.Enum {
		if err := safeText(value); err != nil {
			return "", fmt.Errorf("%s: %w", path, err)
		}
	}
	if err := safeText(f.Description); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	if !e.dataSource && (f.Computed || f.Sensitive) && containsSecretChildren(f) {
		return "", fmt.Errorf("%s: write-only field %s cannot be below a computed or write-only container", path, firstSecret(f, path))
	}
	if f.Sensitive && (f.Type == "object" || ((f.Type == "array" || f.Type == "map") && element(f).Type == "object")) {
		return "", fmt.Errorf("%s: write-only object containers cannot preserve stored companion attributes", path)
	}
	class := attributeClass(f)
	pkg := e.pkg()
	var b strings.Builder
	fmt.Fprintf(&b, "%s.%sAttribute{", pkg, class)
	switch {
	case e.dataSource:
		b.WriteString("Computed:true,")
	case f.Sensitive:
		fmt.Fprintf(&b, "Required:%t,Optional:%t,Sensitive:true,WriteOnly:true,", f.Required, !f.Required)
	case f.Computed:
		b.WriteString("Optional:true,Computed:true,")
	default:
		fmt.Fprintf(&b, "Required:%t,Optional:%t,", f.Required, !f.Required)
	}
	description := f.Description
	for _, doc := range f.EnumDocs {
		if err := safeText(doc.Value + doc.Description); err != nil {
			return "", err
		}
		description += "\n\n- `" + doc.Value + "`"
		if doc.Description != "" {
			description += ": " + doc.Description
		}
	}
	if f.Sensitive {
		description += "\n\nWrite-only: never stored in state. Requires Terraform or OpenTofu 1.11 or later."
	}
	fmt.Fprintf(&b, "MarkdownDescription:%q,", description)
	if !e.dataSource && !f.Sensitive {
		b.WriteString(e.modifiers(modifierClass(class), f.Immutable, f.Computed))
	}
	switch f.Type {
	case "string", "json":
		b.WriteString(e.stringValidators(f))
		if f.Type == "json" {
			e.use("jsontypes", "github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes")
			b.WriteString("CustomType:jsontypes.NormalizedType{},")
		}
	case "object":
		children, err := e.properties(f, false, path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "Attributes:map[string]%s.Attribute{%s},", pkg, children)
	case "array", "map":
		child := element(f)
		if child.Sensitive {
			return "", fmt.Errorf("%s: sensitive collection elements need a named write-only attribute", path)
		}
		if child.Type == "object" {
			children, err := e.properties(child, false, path)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&b, "NestedObject:%s.NestedAttributeObject{Attributes:map[string]%s.Attribute{%s}},", pkg, pkg, children)
		} else {
			typ, err := e.typeExpr(child, path)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&b, "ElementType:%s,", typ)
		}
	case "boolean", "integer", "number":
	default:
		return "", fmt.Errorf("%s: unsupported field type %q", path, f.Type)
	}
	b.WriteString("}")
	return b.String(), nil
}

func element(f *manifest.Field) *manifest.Field {
	if f.Type == "map" {
		return f.Values
	}
	return f.Items
}

func attributeClass(f *manifest.Field) string {
	switch f.Type {
	case "string", "json":
		return "String"
	case "boolean":
		return "Bool"
	case "integer":
		return "Int64"
	case "number":
		return "Float64"
	case "object":
		return "SingleNested"
	case "array":
		if f.Items.Type == "object" {
			return "ListNested"
		}
		return "List"
	case "map":
		if f.Values.Type == "object" {
			return "MapNested"
		}
		return "Map"
	default:
		return "Unsupported"
	}
}

func modifierClass(class string) string {
	switch class {
	case "SingleNested":
		return "Object"
	case "ListNested":
		return "List"
	case "MapNested":
		return "Map"
	default:
		return class
	}
}

func containsSecretChildren(f *manifest.Field) bool {
	for _, child := range f.Properties {
		if child.Sensitive || containsSecretChildren(child) {
			return true
		}
	}
	for _, child := range []*manifest.Field{f.Items, f.Values} {
		if child != nil && (child.Sensitive || containsSecretChildren(child)) {
			return true
		}
	}
	return false
}

func firstSecret(f *manifest.Field, path string) string {
	for _, name := range propertyNames(f) {
		c := f.Properties[name]
		if c.Sensitive {
			return path + "." + name
		}
		if containsSecretChildren(c) {
			return firstSecret(c, path+"."+name)
		}
	}
	for _, child := range []*manifest.Field{f.Items, f.Values} {
		if child != nil {
			return firstSecret(child, path)
		}
	}
	return path
}
