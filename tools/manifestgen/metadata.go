//go:build wfmanifestgen

package main

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/appvia/wayfinder/hack/tfprovider-manifestgen/manifest"
	corev3 "github.com/appvia/wayfinder/pkg/apis/core/v3"
)

type swaggerDocProvider interface{ SwaggerDoc() map[string]string }
type swaggerValidationProvider interface {
	SwaggerValidation() map[string]corev3.FieldValidation
}
type encryptedFieldsProvider interface{ EncryptedFields() map[string]bool }

type typeMetadata struct {
	docs       map[string]string
	validation map[string]corev3.FieldValidation
	encrypted  map[string]bool
}

func metadataFor(t reflect.Type) typeMetadata {
	v := reflect.New(indirect(t)).Interface()
	var meta typeMetadata
	if p, ok := v.(swaggerDocProvider); ok {
		meta.docs = p.SwaggerDoc()
	}
	if p, ok := v.(swaggerValidationProvider); ok {
		meta.validation = p.SwaggerValidation()
	}
	if p, ok := v.(encryptedFieldsProvider); ok {
		meta.encrypted = p.EncryptedFields()
	}
	return meta
}

func applyValidation(f *manifest.Field, v corev3.FieldValidation) error {
	f.Required, f.Pattern = v.Required, v.Pattern
	f.MinLength, f.MaxLength, f.Minimum, f.Maximum = v.MinLength, v.MaxLength, v.Minimum, v.Maximum
	if v.Format != "" {
		f.Format = v.Format
	}
	if v.Enum != nil {
		if len(v.Enum) == 0 {
			return fmt.Errorf("enum with no values")
		}
		f.Enum = slices.Clone(v.Enum)
		f.EnumDocs = slices.DeleteFunc(f.EnumDocs, func(doc manifest.EnumDoc) bool { return !slices.Contains(f.Enum, doc.Value) })
	}
	if v.JSONType != "" {
		switch v.JSONType {
		case "string", "boolean", "integer", "number":
			f.Type, f.Properties, f.PropertyOrder, f.Items, f.Values = v.JSONType, nil, nil, nil, nil
		case "object":
			if f.Type != "object" && f.Type != "map" {
				f.Type, f.Items, f.Values = "json", nil, nil
			}
		case "array":
			if f.Type != "array" {
				f.Type, f.Properties, f.PropertyOrder, f.Values = "array", nil, nil, nil
				f.Items = &manifest.Field{Type: "json"}
			}
		default:
			return fmt.Errorf("unsupported JSONType %q", v.JSONType)
		}
	}
	return nil
}

func hasEncryptionMetadata(t reflect.Type, seen map[reflect.Type]bool) bool {
	t = indirect(t)
	if seen[t] {
		return false
	}
	seen[t] = true
	if _, ok := reflect.New(t).Interface().(encryptedFieldsProvider); ok {
		return true
	}
	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if hasEncryptionMetadata(t.Field(i).Type, seen) {
				return true
			}
		}
	case reflect.Map, reflect.Slice, reflect.Array:
		return hasEncryptionMetadata(t.Elem(), seen)
	}
	return false
}
