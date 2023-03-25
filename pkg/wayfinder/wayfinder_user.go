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
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// makeUserResource returns a resource schema for a user
func makeUserResource() *schema.Resource {
	spec := map[string]*schema.Schema{
		"api_version": addAPIVersionSchema(),
		"kind":        addKindSchema(),
		"metadata":    addMetadataSchema(),
		"spec": {
			Description: "Definition for the user resource",
			Required:    true,
			Type:        schema.TypeList,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"disabled": {
						Type:        schema.TypeBool,
						Description: "Indicates if the user is disabled",
						Optional:    true,
					},
					"email": {
						Type:        schema.TypeString,
						Description: "The email address associated with the user",
						Required:    true,
					},
					"username": {
						Type:        schema.TypeString,
						Description: "A unique username associated with the user",
						Required:    true,
					},
				},
			},
		},
	}

	resource := &Resource{
		APIVersion:              "org.appvia.io/v1alpha1",
		Kind:                    "User",
		Name:                    "wayfinder_users",
		Namespaced:              false,
		Plural:                  "users",
		Schema:                  spec,
		UnsupportedCreateMethod: true,
	}

	log.Printf("adding resource: %s", resource.Name)

	return &schema.Resource{
		CreateContext: createResourceHandler(resource),
		DeleteContext: deleteResourceHandler(resource),
		ReadContext:   readResourceHandler(resource),
		Schema:        spec,
		UpdateContext: updateResourceHandler(resource),
	}
}
