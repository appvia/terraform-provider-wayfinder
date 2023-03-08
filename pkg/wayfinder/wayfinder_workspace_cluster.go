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
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// makeWayfinderWorkspaceCluster creates a new wayfinder workspace cluster
func makeWayfinderWorkspaceCluster() *schema.Resource {
	spec := map[string]*schema.Schema{
		"api_version": addAPIVersionSchema(),
		"kind":        addKindSchema(),
		"metadata":    addMetadataSchema(),
		"spec": {
			Description: "The spec of the wayfinder workspace cluster",
			Type:        schema.TypeList,
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"summary": {
						Description: "A short summary describing the usage of this cluster",
						Optional:    true,
						Type:        schema.TypeString,
					},
					"allocation":       addClusterAllocationSchema(),
					"capabilities":     addClusterCapabilitiesSchema(),
					"channel":          addClusterChannelSchema(),
					"networking":       addClusterNetworkSchema(),
					"node_pools":       addClusterNodePoolSchema(),
					"plan":             addClusterPlanNameSchema(),
					"provider":         addClusterPlanNameSchema(),
					"provider_details": addClusterProviderSchema(),
					"region":           addClusterRegionSchema(),
					"security":         addClusterSecuritySchema(),
					"stage":            addClusterStageSchema(),
					"version":          addClusterVersionSchema(),
				},
			},
		},
	}

	resource := &Resource{
		APIVersion: "compute.appvia.io/v1alpha1",
		Kind:       "Cluster",
		Name:       "clusters",
		Namespaced: true,
		Plural:     "clusters",
		Schema:     spec,
	}

	log.Printf("adding resource: %s", resource.Name)

	return &schema.Resource{
		CreateContext: createWorkspaceCluster,
		DeleteContext: deleteResourceHandler(resource),
		ReadContext:   readResourceHandler(resource),
		Schema:        spec,
		UpdateContext: updateResourceHandler(resource),
	}
}

// createWorkspaceCluster creates a new wayfinder workspace cluster
func createWorkspaceCluster(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {

	return diag.Diagnostics{}
}

// readWorkspaceCluster reads a wayfinder workspace cluster
func readWorkspaceCluster(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {

	return diag.Diagnostics{}
}

// updateWorkspaceCluster updates a wayfinder workspace cluster
func updateWorkspaceCluster(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {

	return diag.Diagnostics{}
}
