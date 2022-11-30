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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	corev1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	"github.com/appvia/wayfinder/pkg/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// makeRequest is responsible for creating the request based off the resource
func makeRequest(resource *Resource, rd *schema.ResourceData, cc client.Interface) client.RestInterface {
	request := cc.Request().Resource(resource.Plural)
	if resource.Namespaced {
		namespace := rd.Get("metadata.0.namespace")
		request = request.Workspace(corev1.ToWorkspace(fmt.Sprintf("%v", namespace)))
	}

	return request
}

// readResourceHandler is called when creating a resource via terraform
func readResourceHandler(resource *Resource) schema.ReadContextFunc {

	return func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
		result := &unstructured.Unstructured{}
		request := makeRequest(resource, rd, i.(client.Interface))
		name := rd.Get("metadata.0.name").(string)

		log.Printf("attempting reading resource: %s/%s", resource.Plural, name)

		// @step: retrieve the resource from the endpoint
		if _, err := request.Result(result).Name(name).Get().Do(); err != nil {
			return diag.FromErr(err)
		}

		// @step: parse the schema and extract the fields
		encoded := &bytes.Buffer{}
		if err := json.NewEncoder(encoded).Encode(result.Object); err != nil {
			return diag.FromErr(err)
		}
		if err := Pack(PackOptions{
			Data:     rd,
			Input:    encoded.Bytes(),
			Resource: resource,
			Schema:   resource.Schema,
		}); err != nil {
			return diag.FromErr(err)
		}

		log.Printf("successfully read resource: %s/%s", resource.Plural, name)

		return diag.Diagnostics{}
	}
}

// createResourceHandler is called when creating a resource via terraform
func createResourceHandler(resource *Resource) schema.CreateContextFunc {

	return func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
		request := makeRequest(resource, rd, i.(client.Interface))

		// @step: we need to unpack the schema into the resource
		object, err := Unpack(UnpackOptions{
			Data:     rd,
			Resource: resource,
			Schema:   resource.Schema,
		})
		if err != nil {
			return diag.FromErr(err)
		}

		// @step: we need to create the resource
		result := &unstructured.Unstructured{}
		if resource.UnsupportedCreateMethod {
			_, err = request.Payload(&object).
				Result(result).
				Name(fmt.Sprintf("%v", rd.Get("metadata.0.name"))).
				Update().
				Do()
		} else {
			_, err = request.Payload(&object).
				Result(result).
				Create().
				Do()
		}
		if err != nil {
			return diag.FromErr(err)
		}

		// @step: we need to set the id of the resource
		switch resource.Namespaced {
		case true:
			rd.SetId(fmt.Sprintf("%s/%s", result.GetNamespace(), result.GetName()))
		default:
			rd.SetId(result.GetName())
		}

		// @step: unpack the resource back into the schema
		encoded := &bytes.Buffer{}
		if err := json.NewEncoder(encoded).Encode(&result.Object); err != nil {
			return diag.FromErr(err)
		}

		if err := Pack(PackOptions{
			Data:     rd,
			Input:    encoded.Bytes(),
			Resource: resource,
			Schema:   resource.Schema,
		}); err != nil {
			return diag.FromErr(err)
		}

		return diag.Diagnostics{}
	}
}

// deleteResourceHandler is called to delete a resource
func deleteResourceHandler(resource *Resource) schema.DeleteContextFunc {

	return func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
		request := makeRequest(resource, rd, i.(client.Interface))
		name := rd.Get("metadata.0.name").(string)

		log.Printf("attempting delete resource: %s/%s", resource.Plural, name)
		if _, err := request.Name(name).Delete().Do(); err != nil {
			return diag.FromErr(err)
		}
		log.Printf("successfully deleted resource: %s/%s", resource.Plural, name)

		return diag.Diagnostics{}
	}
}

// updateResourceHandler is called to update a resource
func updateResourceHandler(resource *Resource) schema.UpdateContextFunc {

	return func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
		current := &unstructured.Unstructured{}
		request := makeRequest(resource, rd, i.(client.Interface))
		name := rd.Get("metadata.0.name").(string)

		// @step: first we need to get the resource
		resp, err := request.Result(current).Name(name).Get().Do()
		if err != nil {
			return diag.FromErr(err)
		}

		// @step: marshal the current resource into json
		response, err := io.ReadAll(resp.Body())
		if err != nil {
			return diag.FromErr(err)
		}

		// @step: apply the changes to the resource
		changed, err := Unpack(UnpackOptions{
			Data:     rd,
			Resource: resource,
			Schema:   resource.Schema,
			Output:   &response,
		})
		if err != nil {
			return diag.FromErr(err)
		}

		// @step: apply changes to the resource
		u := &unstructured.Unstructured{}
		if _, err := request.
			Result(u).
			Payload(changed).
			Update().
			Do(); err != nil {
			return diag.FromErr(err)
		}

		// @step: update the id of the resource
		switch resource.Namespaced {
		case true:
			rd.SetId(fmt.Sprintf("%s/%s", u.GetNamespace(), u.GetName()))
		default:
			rd.SetId(u.GetName())
		}

		return diag.Diagnostics{}
	}
}
