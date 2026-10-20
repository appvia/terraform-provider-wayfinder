// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"strings"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

func (e *emitter) typeExpr(f *manifest.Field, path string) (string, error) {
	e.use("types", "github.com/hashicorp/terraform-plugin-framework/types")

	switch f.Type {
	case "string":
		return "types.StringType", nil
	case "json":
		e.use("jsontypes", "github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes")
		return "jsontypes.NormalizedType{}", nil
	case "boolean":
		return "types.BoolType", nil
	case "integer":
		return "types.Int64Type", nil
	case "number":
		return "types.Float64Type", nil
	case "object":
		if containsSecretChildren(f) {
			return "", fmt.Errorf("%s: write-only children cannot be represented in an unstructured collection element", firstSecret(f, path))
		}

		e.use("attr", "github.com/hashicorp/terraform-plugin-framework/attr")

		var b strings.Builder
		b.WriteString("types.ObjectType{AttrTypes:map[string]attr.Type{")

		for _, name := range propertyNames(f) {
			child := f.Properties[name]
			if e.dataSource && child.Sensitive {
				continue
			}

			typ, err := e.typeExpr(child, path+"."+name)
			if err != nil {
				return "", err
			}

			fmt.Fprintf(&b, "%q:%s,", fieldName(name, child, false), typ)
		}

		b.WriteString("}}")

		return b.String(), nil
	case "array", "map":
		typ, err := e.typeExpr(element(f), path)
		if err != nil {
			return "", err
		}

		class := "List"
		if f.Type == "map" {
			class = "Map"
		}

		return fmt.Sprintf("types.%sType{ElemType:%s}", class, typ), nil
	default:
		return "", fmt.Errorf("%s: unsupported element type %q", path, f.Type)
	}
}
