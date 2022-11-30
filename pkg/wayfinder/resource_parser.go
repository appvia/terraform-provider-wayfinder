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

package wayfinder

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/iancoleman/strcase"
	"github.com/thoas/go-funk"

	"github.com/appvia/terraform-provider-wayfinder/pkg/utils/swagger"
)

// ParserOptions are the options for parsing the specification
type ParserOptions struct {
	// Document is the swagger specification
	Document *loads.Document
	// IncludeResources is used to includes specifics resource types
	IncludeResources []string
	// PathPrefixes is a collection of prefixes to filter on
	PathPrefixes []string
	// SkippedResources is a collection of resources we should skip if found
	SkippedResources []string
}

// ParseForResources is responsible to for parsing the swagger definition and
// and extracting any supportable terraform resources from it
func ParseForResources(options ParserOptions) (Resources, error) {
	var err error

	switch {
	case len(options.Document.Spec().Definitions) == 0:
		return nil, errors.New("no definitions defined")
	case options.Document.Spec() == nil:
		return nil, errors.New("no spec defined")
	case options.Document.Spec().Paths == nil:
		return nil, errors.New("no paths defined in spec")
	case len(options.Document.Spec().Paths.Paths) == 0:
		return nil, nil
	}

	isPathPrefix := func(path string) bool {
		if len(options.PathPrefixes) == 0 {
			return true
		}

		for _, x := range options.PathPrefixes {
			if strings.HasPrefix(path, x) {
				return true
			}
		}

		return false
	}

	resources := make(Resources)
	var parameters []spec.Parameter

	// @note: we only support paths which start with
	// - /api/version/<resource>
	// - /api/version/workspaces/<workspace>/<resource>/<name>
	for path, x := range options.Document.Spec().Paths.Paths {

		items := strings.Split(strings.TrimPrefix(path, "/"), "/")

		// @step: first we filter out the basics
		switch {
		case !isPathPrefix(path):
			continue
		case len(items) != 3 && len(items) != 6:
			continue
		case len(items) == 3 && (x.Post == nil || x.Get == nil):
			continue
		case len(items) == 6 && (x.Put == nil || x.Delete == nil):
			continue
		}

		switch {
		case x.Put != nil:
			parameters = x.Put.Parameters
		case x.Post != nil:
			parameters = x.Post.Parameters
		}

		parameter, found := swagger.GetBodyParameter(parameters)
		switch {
		case !found, !swagger.IsParameterDefinition(parameter):
			continue
		}

		// @step: extract the definition reference, kind and api version
		reference := swagger.GetReference(parameter.ParamProps.Schema.Ref.String())
		kind := swagger.GetResourceKindFromReference(reference)
		group := swagger.GetResourceAPIGroupFromReference(reference)

		resource := &Resource{
			APIVersion: GetGroupVersion(group),
			Kind:       kind,
		}

		// is this a workspace or globally scopes resource
		switch len(items) {
		case 6:
			resource.Plural = items[3]
			resource.Namespaced = true
			resource.Name = fmt.Sprintf("wayfinder_workspace_%s", strings.ToLower(resource.Kind))

		default:
			resource.Plural = items[2]
			resource.Name = fmt.Sprintf("wayfinder_%s", strings.ToLower(resource.Plural))
		}

		// @step: check for included and skipped resources
		switch {
		case len(options.IncludeResources) > 0 && !funk.Contains(options.IncludeResources, resource.Name):
			continue
		case funk.Contains(options.SkippedResources, resource.Name):
			continue
		}

		// @step: we need to lookup the resource definition from the swagger spec
		definition, found := options.Document.Spec().Definitions[reference]
		if !found {
			continue
		}
		resource.Description = definition.Description

		// @step: parse and generate a terraform schema from the definition
		resource.Schema, err = convertToTerraformSchema(definition, options.Document.Spec().Definitions)
		if err != nil {
			return nil, err
		}

		// @step: verify everything went ok
		if err := resource.IsValid(); err != nil {
			return nil, fmt.Errorf("resource %s is invalid: %s", resource.Name, err)
		}

		resources[resource.Name] = resource
	}

	return resources, nil
}

// convertToTerraformSchema is responsible for converting a openapi spec to terraform schema
func convertToTerraformSchema(
	resource spec.Schema,
	definitions spec.Definitions) (map[string]*schema.Schema, error) {

	properties := make(map[string]*schema.Schema)

	// @step: we iterate the properties of the resource and convert to a
	// terraform schema. Where the resource is point to another definition, we
	// move recursive
	for name, property := range resource.Properties {
		name = strcase.ToSnake(name)

		switch name {
		case "status":
			continue
		case "api_version":
			properties["api_version"] = addAPIVersionSchema()
			continue
		case "kind":
			properties["kind"] = addKindSchema()
			continue
		case "metadata":
			properties["metadata"] = addMetadataSchema()
			continue
		}

		properties[name] = &schema.Schema{}
		properties[name].Default = property.Default
		properties[name].Description = property.Description
		properties[name].Optional = !funk.Contains(resource.Required, name)
		properties[name].Required = !properties[name].Optional
		properties[name].Computed = false

		// @else we parse the property
		switch {
		case property.Type.Contains("boolean"):
			properties[name].Type = schema.TypeBool
		case property.Type.Contains("integer"):
			properties[name].Type = schema.TypeInt
		case property.Type.Contains("object"):
			properties[name].Type = schema.TypeMap
		case property.Type.Contains("string"):
			properties[name].Type = schema.TypeString

		case property.Type.Contains("array"):
			properties[name].Type = schema.TypeList
			switch {
			case property.Items.Schema == nil:
				return nil, fmt.Errorf("properties %s contains no schema", name)

			case property.Items.ContainsType("integer"):
				properties[name].Elem = &schema.Schema{Type: schema.TypeInt}
			case property.Items.ContainsType("object"):
				properties[name].Elem = &schema.Schema{Type: schema.TypeMap}
			case property.Items.ContainsType("string"):
				properties[name].Elem = &schema.Schema{Type: schema.TypeString}
			case swagger.GetReference(property.Items.Schema.Ref.String()) != "":
				reference := swagger.GetReference(property.Items.Schema.Ref.String())

				definition, found := definitions[reference]
				if !found {
					return nil, fmt.Errorf("missing definition for %q", reference)
				}
				props, err := convertToTerraformSchema(definition, definitions)
				if err != nil {
					return nil, err
				}

				properties[name].Elem = &schema.Resource{Schema: props}
			}

		case property.Ref.Ref.String() != "":
			reference := swagger.GetReference(property.Ref.Ref.String())

			// @step: we lookup the definition and we go recursive
			definition, found := definitions[reference]
			if !found {
				return nil, fmt.Errorf("missing definition for %q", reference)
			}
			properties[name].Type = schema.TypeList
			properties[name].MaxItems = 1

			props, err := convertToTerraformSchema(definition, definitions)
			if err != nil {
				return nil, err
			}
			properties[name].Elem = &schema.Resource{Schema: props}

		default:
			return nil, fmt.Errorf("unknown property type: %q", property.Type)
		}
	}

	return properties, nil
}

// addAPIVersionSchema is responsible for adding the api version schema
func addAPIVersionSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Description: "The API version for the resource",
		Optional:    true,
		Computed:    true,
	}
}

// addKindSchema is responsible for adding the kind schema
func addKindSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Description: "The resource kind of the object",
		Optional:    true,
		Computed:    true,
	}
}

// addMetadataSchema returns a schema for the metadata
func addMetadataSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Required:    true,
		Description: "Metadata associated to the resource",
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:        schema.TypeString,
					Description: "Name of the resource being created",
					ForceNew:    true,
					Optional:    true,
					Computed:    true,
				},
				"namespace": {
					Type:        schema.TypeString,
					Description: "Namespace defines the space within which name of the resource must be unique.",
					ForceNew:    true,
					Optional:    true,
					Computed:    true,
				},
				"generation": {
					Type:        schema.TypeInt,
					Description: "A sequence number representing a specific generation of the desired state.",
					Computed:    true,
				},
				"annotations": {
					Type:        schema.TypeMap,
					Description: "A collection of key values annotations associated to the resource.",
					Elem:        &schema.Schema{Type: schema.TypeString},
					Optional:    true,
					Computed:    true,
				},
				"labels": {
					Type:        schema.TypeMap,
					Description: "A collection of key values labels associated to the resource.",
					Elem:        &schema.Schema{Type: schema.TypeString},
					Optional:    true,
					Computed:    true,
				},
				"resource_version": {
					Type:        schema.TypeString,
					Description: "An opaque value that represents the internal version of this resource.",
					Computed:    true,
				},
				"uid": {
					Type:        schema.TypeString,
					Description: "The unique in time and space value for this resource.",
					Computed:    true,
				},
			},
		},
	}
}
