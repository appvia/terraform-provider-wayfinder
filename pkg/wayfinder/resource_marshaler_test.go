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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func addTestSpec() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MaxItems: 1,
		Required: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"foo": {
					Type:     schema.TypeString,
					Required: true,
				},
				"bar": {
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
				"test": {
					Type:     schema.TypeMap,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
				"selector": {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"matchLabels": {
								Type:     schema.TypeMap,
								Optional: true,
								Elem:     &schema.Schema{Type: schema.TypeString},
							},
						},
					},
				},
				"subresource": {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"foo": {
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
				"user_names": {
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
				"mylist": {
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:     schema.TypeString,
								Required: true,
							},
							"value": {
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
	}
}

func TestPack(t *testing.T) {
	cases := []struct {
		Expected map[string]interface{}
		Raw      map[string]interface{}
		Resource *Resource
		Schema   map[string]*schema.Schema
		TestFunc func(*testing.T, *schema.ResourceData)
	}{
		{
			Resource: &Resource{
				Kind:       "Test",
				APIVersion: "test.appvia.io",
				Namespaced: true,
			},
			Schema: map[string]*schema.Schema{
				"api_version": {Type: schema.TypeString, Computed: true},
				"kind":        {Type: schema.TypeString, Computed: true},
				"metadata":    addMetadataSchema(),
				"spec":        addTestSpec(),
			},
			Raw: map[string]interface{}{
				"apiVersion": "test.appvia.io",
				"kind":       "Test",
				"metadata": map[string]interface{}{
					"name":      "test",
					"namespace": "default",
					"labels": map[string]interface{}{
						"test": "test",
					},
				},
				"spec": map[string]interface{}{
					"foo": "bar",
					"bar": []interface{}{"foo", "bar"},
					"test": map[string]interface{}{
						"foo": "bar",
					},
					"subresource": map[string]interface{}{
						"foo": "bar",
					},
					"userNames": []interface{}{"foo", "bar"},
					"mylist": []interface{}{
						map[string]interface{}{
							"name":  "foo",
							"value": "bar",
						},
						map[string]interface{}{
							"name":  "bar",
							"value": "foo",
						},
					},
				},
			},
			Expected: map[string]interface{}{
				"apiVersion": "test.appvia.io",
				"kind":       "Test",
				"metadata": map[string]interface{}{
					"name":      "test",
					"namespace": "default",
					"labels": map[string]interface{}{
						"test": "test",
					},
				},
				"spec": map[string]interface{}{
					"foo": "bar",
					"bar": []interface{}{"foo", "bar"},
					"test": map[string]interface{}{
						"foo": "bar",
					},
					"subresource": map[string]interface{}{
						"foo": "bar",
					},
					"userNames": []interface{}{"foo", "bar"},
					"mylist": []interface{}{
						map[string]interface{}{
							"name":  "foo",
							"value": "bar",
						},
						map[string]interface{}{
							"name":  "bar",
							"value": "foo",
						},
					},
				},
			},
			TestFunc: func(t *testing.T, d *schema.ResourceData) {
				assert.Equal(t, "test.appvia.io", d.Get("api_version"))
				assert.Equal(t, "Test", d.Get("kind"))
				assert.Equal(t, "test", d.Get("metadata.0.name"))
				assert.Equal(t, "default", d.Get("metadata.0.namespace"))
				assert.Equal(t, "bar", d.Get("spec.0.foo"))
				assert.Equal(t, []interface{}{"foo", "bar"}, d.Get("spec.0.bar"))
				assert.Equal(t, "bar", d.Get("spec.0.test.foo"))
				assert.Equal(t, "bar", d.Get("spec.0.subresource.0.foo"))
				assert.Equal(t, []interface{}{"foo", "bar"}, d.Get("spec.0.user_names"))
				assert.Equal(t, []interface{}{
					map[string]interface{}{
						"name":  "foo",
						"value": "bar",
					},
					map[string]interface{}{
						"name":  "bar",
						"value": "foo",
					},
				}, d.Get("spec.0.mylist"))

			},
		},
	}

	for _, c := range cases {
		raw := &bytes.Buffer{}
		assert.NoError(t, json.NewEncoder(raw).Encode(&c.Raw))
		assert.NotEmpty(t, raw)

		options := PackOptions{
			Data:     schema.TestResourceDataRaw(t, c.Schema, c.Expected),
			Resource: c.Resource,
			Input:    raw.Bytes(),
			Schema:   c.Schema,
		}
		assert.NoError(t, Pack(options))

		unpacked := UnpackOptions{
			Data:     options.Data,
			Resource: c.Resource,
			Schema:   c.Schema,
		}
		resource, err := Unpack(unpacked)
		assert.NoError(t, err)
		assert.Equal(t, c.Expected, resource)

		if c.TestFunc != nil {
			c.TestFunc(t, options.Data)
		}
	}
}

func TestUnpackResourceFromSchema(t *testing.T) {
	cases := []struct {
		DataFunc func(*schema.ResourceData)
		Expected map[string]interface{}
		Resource *Resource
		Schema   map[string]*schema.Schema
	}{
		{
			Resource: &Resource{},
			Schema: map[string]*schema.Schema{
				"metadata": addMetadataSchema(),
			},
			// nolint: errcheck
			DataFunc: func(d *schema.ResourceData) {
				d.SetId("test")
				d.Set("metadata", []map[string]interface{}{
					{
						"name":      "test",
						"namespace": "default",
						"labels": map[string]interface{}{
							"foo": "bar",
						},
					},
				})
			},
			Expected: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "test",
					"namespace": "default",
					"labels": map[string]interface{}{
						"foo": "bar",
					},
				},
			},
		},
		{
			Resource: &Resource{},
			Schema: map[string]*schema.Schema{
				"metadata": addMetadataSchema(),
				"spec":     addTestSpec(),
			},
			// nolint: errcheck
			DataFunc: func(d *schema.ResourceData) {
				d.SetId("test")
				d.Set("metadata", []map[string]interface{}{
					{
						"name":      "test",
						"namespace": "default",
						"labels": map[string]interface{}{
							"foo": "bar",
						},
					},
				})
				d.Set("spec", []map[string]interface{}{
					{
						"foo": "bar",
						"bar": []string{"foo", "bar"},
						"test": map[string]interface{}{
							"foo": "bar",
						},
						"selector": []map[string]interface{}{
							{
								"matchLabels": map[string]interface{}{
									"foo": "bar",
								},
							},
						},
					},
				})
			},
			Expected: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "test",
					"namespace": "default",
					"labels": map[string]interface{}{
						"foo": "bar",
					},
				},
				"spec": map[string]interface{}{
					"foo":  "bar",
					"bar":  []interface{}{"foo", "bar"},
					"test": map[string]interface{}{"foo": "bar"},
					"selector": map[string]interface{}{
						"matchLabels": map[string]interface{}{
							"foo": "bar",
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		d := schema.TestResourceDataRaw(t, c.Schema, nil)
		c.DataFunc(d)

		o := UnpackOptions{
			Data:     d,
			Resource: c.Resource,
			Schema:   c.Schema,
			Output:   new([]byte),
		}

		err := UnpackResourceFromSchema(Paths{}, o)
		assert.NoError(t, err)

		marshaled := make(map[string]interface{})
		assert.NoError(t, json.Unmarshal(*o.Output, &marshaled))

		assert.Equal(t, c.Expected, marshaled)
	}
}
