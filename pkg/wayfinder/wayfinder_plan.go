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
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/appvia/wayfinder/pkg/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tidwall/gjson"
)

// makePlanResource returns a resource schema for a user
func makePlanResource() *schema.Resource {
	spec := map[string]*schema.Schema{
		"api_version": addAPIVersionSchema(),
		"kind":        addKindSchema(),
		"metadata":    addMetadataSchema(),
		"spec": {
			Description: "Definition for the cluster plan resource",
			Required:    true,
			Type:        schema.TypeList,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"summary": {
						Description: "A summary describing the use case for this plan",
						Optional:    true,
						Type:        schema.TypeString,
					},
					"allocation": addClusterAllocationSchema(),
					"template": {
						Description: "The template which the cluster uses to provision from",
						Required:    true,
						Type:        schema.TypeList,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
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
				},
			},
		},
	}

	resource := &Resource{
		APIVersion: "compute.appvia.io/v1alpha1",
		Kind:       "Plan",
		Name:       "plans",
		Namespaced: false,
		Plural:     "plans",
		Schema:     spec,
	}

	log.Printf("adding resource: %s", resource.Name)

	return &schema.Resource{
		CreateContext: createResourceHandler(resource),
		DeleteContext: deleteResourceHandler(resource),
		ReadContext:   readWayfinderPlan,
		Schema:        spec,
		UpdateContext: updateResourceHandler(resource),
	}
}

func readWayfinderPlan(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
	cc := i.(client.Interface)
	name := rd.Get("metadata.0.name").(string)

	resp, err := cc.Request().Resource("plans").Name(name).Get().Do()
	if err != nil {
		return diag.FromErr(err)
	}

	raw, err := io.ReadAll(resp.Body())
	if err != nil {
		return diag.FromErr(err)
	}

	o := gjson.ParseBytes(raw)
	rd.Set("api_version", o.Get("apiVersion").String())
	rd.Set("kind", o.Get("kind").String())
	rd.SetId(o.Get("metadata.0.uid").String())

	rd.Set("metadata.0.name", o.Get("metadata.name").String())
	rd.Set("metadata.0.namespace", o.Get("metadata.namespace").String())
	rd.Set("metadata.0.uid", o.Get("metadata.uid").String())
	rd.Set("metadata.0.resource_version", o.Get("metadata.resourceVersion").String())
	rd.Set("metadata.0.creation_timestamp", o.Get("metadata.creationTimestamp").String())
	rd.Set("metadata.0.labels", o.Get("metadata.labels"))
	rd.Set("metadata.0.annotations", o.Get("metadata.annotations"))
	rd.Set("spec.0.summary", o.Get("spec.summary").String())
	rd.Set("spec.0.allocation.0.type", o.Get("spec.allocation.type").String())
	rd.Set("spec.0.allocation.0.workspaces", o.Get("spec.allocation.workspaces"))

	capabilities := []string{
		"cert_manager",
		"external_dns_aws",
		"ingress_nginx",
		"psa_operator",
		"terranetes_controller",
	}

	for _, x := range capabilities {
		dest := fmt.Sprintf("spec.0.template.0.capabilities.%s", x)
		source := fmt.Sprintf("spec.template.capabilities.%s", strings.ReplaceAll(x, "_", "-"))

		rd.Set(dest+".description", o.Get(source+".description").String())
		rd.Set(dest+".description", o.Get(source+".enabled").Bool())
	}

	rd.Set("spec.0.template.0.channel", o.Get("spec.template.channel").String())
	rd.Set("spec.0.template.0.networking.0.authorized_master_networks", o.Get("spec.template.networking.authorizedMasterNetworks"))
	rd.Set("spec.0.template.0.networking.0.authorized_networks", o.Get("spec.template.networking.authorizedNetworks"))
	rd.Set("spec.0.template.0.networking.0.cluster_ipv4_cidr", o.Get("spec.template.networking.clusterIPV4Cidr").String())

	rd.Set("spec.0.template.0.plan", o.Get("spec.template.plan").String())
	rd.Set("spec.0.template.0.provider", o.Get("spec.template.provider").String())
	rd.Set("spec.0.template.0.provider_details.0.type", o.Get("spec.template.providerDetails.type").String())

	for _, x := range []string{"api", "audit", "authenticator", "controller_manager", "scheduler"} {
		dest := fmt.Sprintf("spec.0.template.0.provider_details.0.aws.0.%s", x)
		source := fmt.Sprintf("spec.template.providerDetails.aws.%s", x)
		rd.Set(dest, o.Get(source).Bool())
	}

	rd.Set("spec.0.template.0.provider_details.0.aws.0.enable_endpoint_private", o.Get("spec.template.providerDetails.aws.enableEndpointPrivate").Bool())
	rd.Set("spec.0.template.0.provider_details.0.aws.0.security_group_ids", o.Get("spec.template.providerDetails.aws.security_group_ids"))

	rd.Set("spec.0.template.0.provider_details.0.aks.0.linux_profile.0.admin_username", o.Get("spec.template.providerDetails.aks.linuxProfile.adminUsername").String())
	rd.Set("spec.0.template.0.provider_details.0.aks.0.linux_profile.0.ssh_public_keys", o.Get("spec.template.providerDetails.aks.linuxProfile.sshPublicKeys"))
	rd.Set("spec.0.template.0.provider_details.0.aks.0.sku_tier", o.Get("spec.template.providerDetails.aks.skuTier").String())
	rd.Set("spec.0.template.0.provider_details.0.aks.0.windows_profile.0.admin_username", o.Get("spec.template.providerDetails.aks.windowsProfile.adminUsername").String())
	rd.Set("spec.0.template.0.provider_details.0.aks.0.windows_profile.0.admin_password", o.Get("spec.template.providerDetails.aks.windowsProfile.adminPassword").String())
	rd.Set("spec.0.template.0.provider_details.0.aks.0.windows_profile.0.resource_group_override", o.Get("spec.template.providerDetails.aks.windowsProfile.resourceGroupOverride").String())
	rd.Set("spec.0.template.0.provider_details.0.aks.0.windows_profile.0.infra_resource_group_override", o.Get("spec.template.providerDetails.aks.windowsProfile.infraResourceGroupOverride").String())

	rd.Set("spec.0.template.0.provider_details.0.gke.0.enable_horizontal_pod_autoscaler", o.Get("spec.template.providerDetails.gke.enableHorizontalPodAutoscaler").Bool())
	rd.Set("spec.0.template.0.provider_details.0.gke.0.enable_http_loadbalancer", o.Get("spec.template.providerDetails.gke.enableHTTPLoadbalancer").Bool())
	rd.Set("spec.0.template.0.provider_details.0.gke.0.enable_shielded_nodes", o.Get("spec.template.providerDetails.gke.enableShieldedNodes").Bool())
	rd.Set("spec.0.template.0.provider_details.0.gke.0.enable_stackdriver_logging", o.Get("spec.template.providerDetails.gke.enableStackdriverLogging").Bool())
	rd.Set("spec.0.template.0.provider_details.0.gke.0.enable_stackdriver_metrics", o.Get("spec.template.providerDetails.gke.enableStackdriverMetrics").Bool())
	rd.Set("spec.0.template.0.provider_details.0.gke.0.master_ipv4_cidr", o.Get("spec.template.providerDetails.gke.masterIPV4Cidr").String())

	rd.Set("spec.0.template.0.security.0.allowed", o.Get("spec.template.security.allowed"))
	rd.Set("spec.0.template.0.security.0.default_profile", o.Get("spec.template.security.defaultProfile").String())
	rd.Set("spec.0.template.0.security.0.enabled", o.Get("spec.template.security.enabled").Bool())

	rd.Set("spec.0.template.0.stage", o.Get("spec.template.stage").String())
	rd.Set("spec.0.template.0.version", o.Get("spec.template.version").String())

	return diag.Diagnostics{}
}
