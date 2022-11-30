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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/iancoleman/strcase"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Pack is called to walk the resource schema and extract the fields from
// the resource data
func Pack(o PackOptions) error {
	switch {
	case o.Resource == nil:
		return errors.New("resource is required")
	case o.Schema == nil:
		return errors.New("schema is required")
	case o.Data == nil:
		return errors.New("data is required")
	case o.Rendered == nil:
		o.Rendered = new([]byte)
	}

	if err := o.Data.Set("api_version", o.Resource.APIVersion); err != nil {
		return err
	}
	if err := o.Data.Set("kind", o.Resource.Kind); err != nil {
		return err
	}

	if err := PackResourceFromSchema(Paths{}, o); err != nil {
		return err
	}

	log.Printf("[DEBUG] rendered resource: %s", string(*o.Rendered))

	// @step: decode in the resource into schema.ResourceData
	for k := range o.Schema {
		if err := o.Data.Set(k, gjson.GetBytes(*o.Rendered, k).Value()); err != nil {
			return err
		}
	}

	// @step: set the resource id
	o.Data.SetId(fmt.Sprintf("%v", o.Data.Get("metadata.0.name")))
	if o.Resource.Namespaced {
		o.Data.SetId(fmt.Sprintf("%s/%s",
			o.Data.Get("metadata.0.namespace"),
			o.Data.Get("metadata.0.name"),
		))
	}

	return nil
}

// PackResourceFromSchema is called to walk the resource schema and extract the fields from
func PackResourceFromSchema(i Paths, o PackOptions) error {

	update := func(src, dest string) error {
		value := gjson.GetBytes(o.Input, src)
		if !value.Exists() {
			return nil
		}

		updated, err := sjson.SetBytes(*o.Rendered, dest, value.Value())
		if err != nil {
			return err
		}
		*o.Rendered = updated

		return nil
	}

	for name, field := range o.Schema {
		src := i.FieldPath(strcase.ToLowerCamel(name))
		dest := i.PointerPath(strcase.ToSnake(name))

		switch field.Type {
		case schema.TypeBool, schema.TypeInt, schema.TypeFloat, schema.TypeString, schema.TypeMap:
			if err := update(src, dest); err != nil {
				return err
			}

		case schema.TypeList, schema.TypeSet:
			switch field.Elem.(type) {
			case *schema.Resource:
				if field.MaxItems == 1 {
					err := PackResourceFromSchema(
						Paths{
							Path:    src,
							Pointer: dest + ".0",
						},
						PackOptions{
							Resource: o.Resource,
							Schema:   field.Elem.(*schema.Resource).Schema,
							Data:     o.Data,
							Input:    o.Input,
							Rendered: o.Rendered,
						})
					if err != nil {
						return err
					}
				} else {
					length := len(gjson.GetBytes(o.Input, src).Array())
					for x := 0; x < length; x++ {
						err := PackResourceFromSchema(
							Paths{
								Path:    src + fmt.Sprintf(".%d", x),
								Pointer: dest + fmt.Sprintf(".%d", x),
							},
							PackOptions{
								Resource: o.Resource,
								Schema:   field.Elem.(*schema.Resource).Schema,
								Data:     o.Data,
								Input:    o.Input,
								Rendered: o.Rendered,
							})
						if err != nil {
							return err
						}
					}
				}

			case *schema.Schema:
				if err := update(src, dest); err != nil {
					return err
				}

			default:
				return fmt.Errorf("unsupported type: %T", field.Elem)
			}

		default:
			return fmt.Errorf("unsupported field type %s", field.Type)
		}
	}

	return nil
}

// Unpack is shorthand for unpacking a resource
func Unpack(o UnpackOptions) (map[string]interface{}, error) {
	switch {
	case o.Resource == nil:
		return nil, errors.New("resource is required")
	case o.Schema == nil:
		return nil, errors.New("schema is required")
	case o.Data == nil:
		return nil, errors.New("data is required")
	case o.Output == nil:
		o.Output = new([]byte)
	}

	if err := o.Data.Set("api_version", o.Resource.APIVersion); err != nil {
		return nil, err
	}
	if err := o.Data.Set("kind", o.Resource.Kind); err != nil {
		return nil, err
	}

	if err := UnpackResourceFromSchema(Paths{}, o); err != nil {
		return nil, err
	}

	resource := make(map[string]interface{})
	if err := json.NewDecoder(bytes.NewReader(o.Bytes())).Decode(&resource); err != nil {
		return nil, err
	}

	return resource, nil
}

// UnpackResourceFromSchema is responsible for taking the schema, iterating the
// properties and unpacking them into a object
// Note: this method SHOULD be called via Unpack() and not directly
func UnpackResourceFromSchema(index Paths, o UnpackOptions) error {

	updateFunc := func(src, dest string) error {
		value, found := o.Data.GetOk(src)
		if !found {
			return nil
		}

		data, err := sjson.SetBytes(o.Bytes(), dest, value)
		if err != nil {
			return err
		}
		o.SetOutput(data)

		return nil
	}

	for name, field := range o.Schema {
		src := index.FieldPath(name)
		dest := index.PointerPath(strcase.ToLowerCamel(name))

		switch field.Type {
		case schema.TypeBool, schema.TypeInt, schema.TypeFloat, schema.TypeString:
			if err := updateFunc(src, dest); err != nil {
				return err
			}

		case schema.TypeList, schema.TypeSet:
			switch field.Elem.(type) {
			case nil:
				return fmt.Errorf("no element type for %s", src)

			case *schema.Schema:
				if err := updateFunc(src, dest); err != nil {
					return err
				}

			case *schema.Resource:
				if field.MaxItems == 1 {
					err := UnpackResourceFromSchema(
						Paths{
							Path:    fmt.Sprintf("%s.0", src),
							Pointer: dest,
						},
						UnpackOptions{
							Data:     o.Data,
							Output:   o.Output,
							Resource: o.Resource,
							Schema:   field.Elem.(*schema.Resource).Schema,
						},
					)
					if err != nil {
						return err
					}

					continue
				}

				var index int
				for {
					_, found := o.Data.GetOk(fmt.Sprintf("%s.%d", src, index))
					if !found {
						break
					}
					err := UnpackResourceFromSchema(
						Paths{
							Path:    fmt.Sprintf("%s.%d", src, index),
							Pointer: fmt.Sprintf("%s.%d", dest, index),
						},
						UnpackOptions{
							Data:     o.Data,
							Output:   o.Output,
							Resource: o.Resource,
							Schema:   field.Elem.(*schema.Resource).Schema,
						},
					)
					if err != nil {
						return err
					}
					index++
				}

			default:
				return fmt.Errorf("unsupported element type for %T", field.Elem)
			}

		case schema.TypeMap:
			if err := updateFunc(src, dest); err != nil {
				return err
			}
		}
	}

	return nil
}
