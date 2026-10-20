// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"strings"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

func (e *emitter) modifiers(class string, immutable, computed bool) string {
	if !immutable && !computed {
		return ""
	}

	e.use("planmodifier", "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier")

	alias := strings.ToLower(class) + "planmodifier"
	e.use(alias, "github.com/hashicorp/terraform-plugin-framework/resource/schema/"+alias)

	var b strings.Builder
	fmt.Fprintf(&b, "PlanModifiers:[]planmodifier.%s{", class)

	if immutable {
		fmt.Fprintf(&b, "%s.RequiresReplace(),", alias)
	}

	if computed {
		fmt.Fprintf(&b, "%s.UseStateForUnknown(),", alias)
	}

	b.WriteString("},")

	return b.String()
}

func (e *emitter) stringValidators(f *manifest.Field) string {
	if len(f.Enum) == 0 && f.Pattern == "" && f.MinLength == nil && f.MaxLength == nil {
		return ""
	}

	e.use("validator", "github.com/hashicorp/terraform-plugin-framework/schema/validator")
	e.use("stringvalidator", "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator")

	var b strings.Builder
	b.WriteString("Validators:[]validator.String{")

	if len(f.Enum) > 0 {
		b.WriteString("stringvalidator.OneOf(")

		for _, value := range f.Enum {
			fmt.Fprintf(&b, "%q,", value)
		}

		b.WriteString("),")
	}

	if f.Pattern != "" {
		e.use("regexp", "regexp")
		fmt.Fprintf(&b, "stringvalidator.RegexMatches(regexp.MustCompile(%q), %q),", f.Pattern, "must match the API field pattern")
	}

	switch {
	case f.MinLength != nil && f.MaxLength != nil:
		fmt.Fprintf(&b, "stringvalidator.LengthBetween(%d,%d),", *f.MinLength, *f.MaxLength)
	case f.MinLength != nil:
		fmt.Fprintf(&b, "stringvalidator.LengthAtLeast(%d),", *f.MinLength)
	case f.MaxLength != nil:
		fmt.Fprintf(&b, "stringvalidator.LengthAtMost(%d),", *f.MaxLength)
	}

	b.WriteString("},")

	return b.String()
}
