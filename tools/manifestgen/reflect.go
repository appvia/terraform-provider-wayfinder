//go:build wfmanifestgen

package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/appvia/wayfinder/hack/tfprovider-manifestgen/manifest"
	corev3 "github.com/appvia/wayfinder/pkg/apis/core/v3"
)

func indirect(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func snake(name string) string {
	runes := []rune(name)
	var b strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 && runes[i-1] != '_' &&
			(unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1]) ||
				(i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func reflectField(t reflect.Type, visiting map[reflect.Type]bool) (*manifest.Field, error) {
	t = indirect(t)
	f := &manifest.Field{}
	if t == reflect.TypeFor[corev3.Time]() {
		f.Type, f.Format = "string", "date-time"
		return f, nil
	}
	if t == reflect.TypeFor[json.RawMessage]() || visiting[t] {
		f.Type = "json"
		return f, nil
	}
	visiting[t] = true
	defer delete(visiting, t)
	switch t.Kind() {
	case reflect.String:
		f.Type = "string"
	case reflect.Bool:
		f.Type = "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		f.Type = "integer"
	case reflect.Float32, reflect.Float64:
		f.Type = "number"
	case reflect.Slice, reflect.Array:
		item, err := reflectField(t.Elem(), visiting)
		if err != nil {
			return nil, fmt.Errorf("%s items: %w", t, err)
		}
		f.Type, f.Items = "array", item
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			f.Type = "json"
			break
		}
		value, err := reflectField(t.Elem(), visiting)
		if err != nil {
			return nil, fmt.Errorf("%s values: %w", t, err)
		}
		f.Type, f.Values = "map", value
	case reflect.Struct:
		if err := reflectProperties(f, t, visiting); err != nil {
			return nil, err
		}
		if len(f.Properties) == 0 {
			f.Type = "json"
		}
	default:
		f.Type = "json"
	}
	if p, ok := reflect.New(t).Interface().(corev3.EnumValuesProvider); ok {
		values := p.EnumValues()
		if len(values) == 0 {
			return nil, fmt.Errorf("%s: enum with no values", t)
		}
		for _, value := range values {
			f.Enum = append(f.Enum, value.Value)
			f.EnumDocs = append(f.EnumDocs, manifest.EnumDoc{Value: value.Value, Description: value.Description})
		}
	}
	return f, nil
}

func reflectProperties(f *manifest.Field, t reflect.Type, visiting map[reflect.Type]bool) error {
	f.Type, f.Properties = "object", make(map[string]*manifest.Field)
	meta := metadataFor(t)
	f.Description = meta.docs[""]
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if field.PkgPath != "" || name == "-" {
			continue
		}
		child, err := reflectField(field.Type, visiting)
		if err != nil {
			return fmt.Errorf("%s.%s: %w", t, field.Name, err)
		}
		if name == "" && field.Anonymous {
			for _, embedded := range child.PropertyOrder {
				if err := addProperty(f, embedded, child.Properties[embedded]); err != nil {
					return err
				}
			}
			continue
		}
		if name == "" {
			name = field.Name
		}
		desc, ok := meta.docs[name]
		if !ok {
			desc, ok = meta.docs[strings.ToLower(field.Name[:1])+field.Name[1:]]
		}
		if ok {
			child.Description = desc
		}
		child.Sensitive = meta.encrypted[name]
		if validation, ok := meta.validation[name]; ok {
			if err := applyValidation(child, validation); err != nil {
				return fmt.Errorf("%s.%s: %w", t, name, err)
			}
		}
		if err := addProperty(f, name, child); err != nil {
			return fmt.Errorf("%s: %w", t, err)
		}
	}
	return nil
}

func addProperty(parent *manifest.Field, name string, child *manifest.Field) error {
	for _, existing := range parent.PropertyOrder {
		if snake(existing) == snake(name) {
			return fmt.Errorf("spec fields %q and %q collide after snake_case", existing, name)
		}
	}
	parent.Properties[name] = child
	parent.PropertyOrder = append(parent.PropertyOrder, name)
	return nil
}
