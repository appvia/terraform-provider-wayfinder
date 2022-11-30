/*
 * Copyright (C) 2022  Appvia Ltd <info@appvia.io>
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License
 * as published by the Free Software Foundation; either version 2
 * of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package swagger

import (
	"strings"

	"github.com/go-openapi/spec"
)

var (
	// DefinitionPrefix is the prefix used for locally defined structures
	// in the swagger spec
	DefinitionPrefix = "#/definitions/"
)

// HasDefinition returns true if the reference has a local prefix
func HasDefinition(reference string) bool {
	return strings.HasPrefix(reference, DefinitionPrefix)
}

// IsParameterDefinition returns true if the parameter is a definition
func IsParameterDefinition(parameter spec.Parameter) bool {
	return HasDefinition(parameter.ParamProps.Schema.Ref.String())
}

// GetReference returns the local definition reference
func GetReference(reference string) string {
	if !HasDefinition(reference) {
		return reference
	}

	return strings.TrimPrefix(reference, DefinitionPrefix)
}

// GetResourceKindFromReference returns the resource kind from the reference
func GetResourceKindFromReference(reference string) string {
	items := strings.Split(reference, ".")

	return items[len(items)-1]
}

// GetResourceAPIGroupFromReference returns the resource apigroup from the reference path
func GetResourceAPIGroupFromReference(reference string) string {
	return strings.TrimPrefix(strings.Split(reference, ".")[0], DefinitionPrefix)
}

// GetBodyParameter returns the body parameter
func GetBodyParameter(parameters []spec.Parameter) (spec.Parameter, bool) {
	for _, x := range parameters {
		if x.Name == "body" && x.Required {
			return x, true
		}
	}

	return spec.Parameter{}, false
}
