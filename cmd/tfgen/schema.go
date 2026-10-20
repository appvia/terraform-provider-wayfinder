// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

type emitter struct {
	imports    map[string]string
	dataSource bool
	// inComputed is set while emitting the descendants of a computed container. Terraform
	// treats a non-computed value inside a server-filled computed container as configuration
	// and plans it away, so every descendant is Optional+Computed.
	inComputed bool
}

func (e *emitter) use(alias, path string) string { e.imports[alias] = path; return alias }
func (e *emitter) pkg() string {
	if e.dataSource {
		return e.use("dschema", "github.com/hashicorp/terraform-plugin-framework/datasource/schema")
	}
	return e.use("rschema", "github.com/hashicorp/terraform-plugin-framework/resource/schema")
}

func (e *emitter) kindSchema(k manifest.Kind) (string, error) {
	description, err := reword(k.Description)
	if err != nil {
		return "", fmt.Errorf("%s: %w", k.Kind, err)
	}
	if err := safeText(k.Description); err != nil {
		return "", err
	}
	attrs, err := e.properties(k.Spec, true, "spec")
	if err != nil {
		return "", err
	}
	pkg := e.pkg()
	var b strings.Builder
	b.WriteString(attrs)
	fmt.Fprintf(&b, "%q: %s.StringAttribute{Computed:true,", "id", pkg)
	if !e.dataSource {
		b.WriteString(e.modifiers("String", false, true))
	}
	b.WriteString("},\n")
	identity := func(name string, required bool) {
		fmt.Fprintf(&b, "%q: %s.StringAttribute{Required:%t, Optional:%t,", name, pkg, required, !required)
		if !e.dataSource {
			b.WriteString(e.modifiers("String", true, false))
		}
		if name == "version" {
			b.WriteString(e.stringValidators(&manifest.Field{Type: "string", Pattern: semverPattern}))
		}
		b.WriteString("},\n")
	}
	identity("name", true)
	if slices.Contains(k.Scopes, "workspace") || slices.Contains(k.Scopes, "environment") {
		identity("workspace", !slices.Contains(k.Scopes, "tenant"))
	}
	if slices.Contains(k.Scopes, "environment") {
		identity("environment", len(k.Scopes) == 1)
	}
	if k.Versioned {
		identity("version", true)
	}
	e.use("types", "github.com/hashicorp/terraform-plugin-framework/types")
	for _, name := range []string{"labels", "annotations"} {
		mode := "Optional:true"
		if e.dataSource {
			mode = "Computed:true"
		}
		fmt.Fprintf(&b, "%q: %s.MapAttribute{%s, ElementType:types.StringType},\n", name, pkg, mode)
	}
	fmt.Fprintf(&b, "%q: %s.SingleNestedAttribute{Computed:true, Attributes:map[string]%s.Attribute{", "status", pkg, pkg)
	for _, name := range []string{"status", "message", "detail"} {
		fmt.Fprintf(&b, "%q: %s.StringAttribute{Computed:true},", name, pkg)
	}
	b.WriteString("}},\n")
	e.use("jsontypes", "github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes")
	fmt.Fprintf(&b, "%q: %s.StringAttribute{Computed:true, CustomType:jsontypes.NormalizedType{}},\n", "status_json", pkg)
	if !e.dataSource {
		e.use("timeouts", "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts")
		b.WriteString("\"timeouts\": timeouts.Attributes(ctx, timeouts.Opts{Create:true, Update:true, Delete:true}),\n")
		if k.HasStatus {
			e.use("booldefault", "github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault")
			b.WriteString("\"wait_for_ready\": rschema.BoolAttribute{Optional:true, Computed:true, Default:booldefault.StaticBool(true)},\n")
		}
	}
	return fmt.Sprintf("%s.Schema{MarkdownDescription:%q, Attributes:map[string]%s.Attribute{%s}}", pkg, description, pkg, b.String()), nil
}

func propertyNames(f *manifest.Field) []string {
	if f == nil {
		return nil
	}
	return slices.Sorted(maps.Keys(f.Properties))
}

// Strict SemVer 2.0: numeric prerelease identifiers cannot have leading zeroes.
const semverPattern = `^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(\.(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`
