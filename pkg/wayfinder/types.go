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
	"io"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Config is the configuration options for the provider
type Config struct {
	// IncludeResources includes specific resources
	IncludeResources []string
	// SkippedResources is a collection of resources we should not handle
	SkippedResources []string
	// SwaggerSpec is the reader containing the swagger configuration
	SwaggerSpec io.Reader
}

// Resources is a map of name to resource definition
type Resources map[string]*Resource

// Resource defines a managed wayfinder resource
type Resource struct {
	// APIVersion is the api group of the resource
	APIVersion string
	// Kind is the resource type
	Kind string
	// Plural is the plural form of the resource
	Plural string
	// Name is the terraform name for this resource
	Name string
	// Description is a description for the resource
	Description string
	// Namespaced indicates if the resource is namespaced (i.e. in a workspace)
	Namespaced bool
	// Schema is the terraform schema for this resource
	Schema map[string]*schema.Schema
	// UnsupportedCreateMethod indicates if the resource supports a create method
	UnsupportedCreateMethod bool
}
