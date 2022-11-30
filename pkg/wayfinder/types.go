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

// Methods is a collection of method the resource supports
type Methods struct {
	// Create is the create method
	Create bool
	// Delete is the delete method
	Delete bool
	// Get is the get method
	Get bool
	// Update is the update method
	Update bool
}

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
	// Methods is a collection of methods the resource supports
	Methods Methods
	// UnsupportedCreateMethod indicates if the resource supports a create method
	UnsupportedCreateMethod bool
}

// UnpackOptions are the options for unpacking
type UnpackOptions struct {
	// Resource is the resource we are unpacking
	Resource *Resource
	// Schema is the schema we are unpacking
	Schema map[string]*schema.Schema
	// Data is the data we are unpacking
	Data *schema.ResourceData
	// Output is the output we are unpacking
	Output *[]byte
}

// String returns the string representation of the output
func (o *UnpackOptions) String() string {
	if o.Output == nil {
		return ""
	}

	return string(*o.Output)
}

// SetOutput sets the output of the options
func (o *UnpackOptions) SetOutput(data []byte) {
	*o.Output = data
}

// Bytes returns the bytes representation of the output
func (o *UnpackOptions) Bytes() []byte {
	if o.Output == nil {
		return nil
	}

	return *o.Output
}

// Paths is the index of the resource
type Paths struct {
	// Path is the path of the current field
	Path string
	// Pointer is the path to store the value
	Pointer string
}

// PackOptions are the options for parsing and extracting date
type PackOptions struct {
	// Data is the data we are unpacking
	Data *schema.ResourceData
	// Input is the underlying raw data
	Input []byte
	// Resource is the resource we are unpacking
	Resource *Resource
	// Rendered is the output we are unpacking
	Rendered *[]byte
	// Schema is the schema we are unpacking
	Schema map[string]*schema.Schema
}

// IsValid checks if the options are valid
func (r *PackOptions) IsValid() error {
	switch {
	case r.Resource == nil:
		return errors.New("resource is required")
	case r.Schema == nil:
		return errors.New("schema is required")
	case r.Data == nil:
		return errors.New("data is required")
	case r.Rendered == nil:
		r.Rendered = new([]byte)
	}

	return nil
}

// FieldPath returns the field name plus path if there
func (u *Paths) FieldPath(name string) string {
	if u.Path == "" {
		return name
	}

	return fmt.Sprintf("%s.%s", u.Path, name)
}

// PointerPath returns the field name plus path if there
func (u *Paths) PointerPath(name string) string {
	if u.Pointer == "" {
		return name
	}

	return fmt.Sprintf("%s.%s", u.Pointer, name)
}
