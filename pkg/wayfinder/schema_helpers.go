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

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

// addClusterSummarySchema adds the cluster summary schema
func addClusterSummarySchema() *schema.Schema {
	return &schema.Schema{
		Description: "A short summary describing the usage for this resource",
		Optional:    true,
		Type:        schema.TypeString,
	}
}

// addClusterChannelSchema adds the cluster channel schema
func addClusterChannelSchema() *schema.Schema {
	return &schema.Schema{
		Description: "The channel to use for the plan",
		Optional:    true,
		Type:        schema.TypeString,
	}
}

// addClusterPlanNameSchema adds the cluster plan name schema
func addClusterPlanNameSchema() *schema.Schema {
	return &schema.Schema{
		Description: "The name of the plan for the cluster",
		Optional:    true,
		Type:        schema.TypeString,
		Computed:    true,
	}
}

// addClusterProviderNameSchema adds the cluster provider schema
func addClusterProviderNameSchema() *schema.Schema {
	return &schema.Schema{
		Description: "Is the name of the cluster plan this cluster is using",
		Required:    true,
		Type:        schema.TypeString,
	}
}

// addClusterRegionSchema adds the cluster region schema
func addClusterRegionSchema() *schema.Schema {
	return &schema.Schema{
		Description: "The default region for the cluster plan",
		Required:    true,
		Type:        schema.TypeString,
	}
}

// addClusterStageSchema adds the cluster stage schema
func addClusterStageSchema() *schema.Schema {
	return &schema.Schema{
		Description: "The stage name this cluster will reside in",
		Required:    true,
		Type:        schema.TypeString,
	}
}

// addClusterVersionSchema adds the cluster version schema
func addClusterVersionSchema() *schema.Schema {
	return &schema.Schema{
		Description: "The Kubernetes version of the cluster",
		Required:    true,
		Type:        schema.TypeString,
	}
}

// addClusterSecuritySchema adds the cluster security schema
func addClusterSecuritySchema() *schema.Schema {
	return &schema.Schema{
		Description: "The security configuration for the cluster plan",
		Required:    true,
		MaxItems:    1,
		Type:        schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"pod_security_standard": {
					Description: "The pod security standard to use for the cluster plan",
					Required:    true,
					Type:        schema.TypeList,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"allowed": {
								Description: "The allowed pod security standards",
								Optional:    true,
								Type:        schema.TypeList,
								Elem:        &schema.Schema{Type: schema.TypeString},
							},
							"default_profile": {
								Description: "The default pod security standard profile",
								Optional:    true,
								Type:        schema.TypeString,
							},
							"enabled": {
								Description: "Whether pod security standards are enabled for the cluster plan",
								Required:    true,
								Type:        schema.TypeBool,
							},
						},
					},
				},
			},
		},
	}
}

// addCapabilitiesSchema adds the capabilities schema
func addCapabilitiesSchema() *schema.Schema {
	return &schema.Schema{
		Description: "The terranetes controller configuration",
		Optional:    true,
		Type:        schema.TypeList,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"enabled": {
					Description: "Whether the capability is enabled",
					Required:    true,
					Type:        schema.TypeBool,
				},
				"description": {
					Description: "A description of the capability",
					Optional:    true,
					Type:        schema.TypeString,
				},
			},
		},
	}
}

// addClusterCapabilitiesSchema adds the cluster capabilities schema
func addClusterCapabilitiesSchema() *schema.Schema {
	return &schema.Schema{
		Description: "The capabilities / features which should be enabled",
		Optional:    true,
		Type:        schema.TypeList,
		MinItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"cert_manager":          addCapabilitiesSchema(),
				"external_dns_aks":      addCapabilitiesSchema(),
				"external_dns_aws":      addCapabilitiesSchema(),
				"external_dns_gke":      addCapabilitiesSchema(),
				"ingress_nginx":         addCapabilitiesSchema(),
				"psa_operator":          addCapabilitiesSchema(),
				"terranetes_controller": addCapabilitiesSchema(),
			},
		},
	}
}

// addClusterProviderSchema adds the cluster provider schema
func addClusterProviderSchema() *schema.Schema {
	return &schema.Schema{
		Description: "The cloud provider specific details for the cluster",
		Optional:    true,
		MaxItems:    1,
		Type:        schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": {
					Description: "The cloud provider type",
					Required:    true,
					Type:        schema.TypeString,
				},
				"eks": addClusterEKSProviderSchema(),
				"aks": addClusterAKSProviderSchema(),
				"gke": addClusterGKEProviderSchema(),
			},
		},
	}
}

// addClusterEKSProviderSchema adds the cluster eks provider schema
func addClusterEKSProviderSchema() *schema.Schema {
	return &schema.Schema{
		Description: "The EKS specific details for the cluster",
		Optional:    true,
		MaxItems:    1,
		Type:        schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"admin_arns": {
					Description: "The ARNs of the IAM users or roles that will be granted admin access to the cluster",
					Optional:    true,
					Type:        schema.TypeList,
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
				"cloud_watch_logging": {
					Description: "Whether to enable CloudWatch logging for the cluster",
					Optional:    true,
					Type:        schema.TypeList,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"api": {
								Description: "Whether to enable CloudWatch logging for the Kubernetes API server",
								Optional:    true,
								Type:        schema.TypeBool,
							},
							"audit": {
								Description: "Whether to enable CloudWatch logging for the Kubernetes audit logs",
								Optional:    true,
								Type:        schema.TypeBool,
							},
							"authenticator": {
								Description: "Whether to enable CloudWatch logging for the Kubernetes authenticator",
								Optional:    true,
								Type:        schema.TypeBool,
							},
							"controller_manager": {
								Description: "Whether to enable CloudWatch logging for the Kubernetes controller manager",
								Optional:    true,
								Type:        schema.TypeBool,
							},
							"scheduler": {
								Description: "Whether to enable CloudWatch logging for the Kubernetes scheduler",
								Optional:    true,
								Type:        schema.TypeBool,
							},
						},
					},
				},
				"enable_endpoint_private": {
					Description: "Whether to enable private access to the cluster",
					Optional:    true,
					Type:        schema.TypeBool,
				},
				"security_group_ids": {
					Description: "The IDs of the security groups to use for the cluster",
					Optional:    true,
					Type:        schema.TypeList,
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
}

// addClusterAKSProviderSchema adds the cluster aks provider schema
func addClusterAKSProviderSchema() *schema.Schema {
	return &schema.Schema{
		Description: "The AKS specific details for the cluster",
		Optional:    true,
		MaxItems:    1,
		Type:        schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"linux_profile": {
					Description: "The Linux profile for the cluster",
					Optional:    true,
					MaxItems:    1,
					Type:        schema.TypeList,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"admin_username": {
								Description: "The admin username for the cluster",
								Required:    true,
								Type:        schema.TypeString,
							},
							"ssh_public_keys": {
								Description: "The SSH public keys for the cluster",
								Optional:    true,
								Type:        schema.TypeList,
								Elem:        &schema.Schema{Type: schema.TypeString},
							},
						},
					},
				},
				"sku_tier": {
					Description: "The SKU tier for the cluster",
					Optional:    true,
					Type:        schema.TypeString,
				},
				"windows_profile": {
					Description: "The Windows profile for the cluster",
					Optional:    true,
					MaxItems:    1,
					Type:        schema.TypeList,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"admin_password": {
								Description: "The admin password for Windows VMs",
								Required:    true,
								Type:        schema.TypeString,
							},
							"admin_username": {
								Description: "The admin is the admin username for Windows VMs",
								Required:    true,
								Type:        schema.TypeString,
							},
						},
					},
				},
				"resource_group_override": {
					Description: "The resource group will specify a resource group when required",
					Optional:    true,
					Type:        schema.TypeString,
				},
				"infra_resource_group_override": {
					Description: "The infra resource group will specify a resource group when required",
					Optional:    true,
					Type:        schema.TypeString,
				},
			},
		},
	}
}

// addClusterGKEProviderSchema adds the cluster gke provider schema
func addClusterGKEProviderSchema() *schema.Schema {
	return &schema.Schema{
		Description: "The GKE specific details for the cluster",
		Optional:    true,
		MaxItems:    1,
		Type:        schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"enable_horizontal_pod_autoscaler": {
					Description: "The enable horizontal pod autoscaler for the cluster",
					Optional:    true,
					Type:        schema.TypeBool,
				},
				"enable_http_loadbalancer": {
					Description: "The enable http loadbalancer for the cluster",
					Optional:    true,
					Type:        schema.TypeBool,
				},
				"enable_shielded_nodes": {
					Description: "The enable shielded nodes for the cluster",
					Optional:    true,
					Type:        schema.TypeBool,
				},
				"enable_stackdriver_logging": {
					Description: "The enable stackdriver logging for the cluster",
					Optional:    true,
					Type:        schema.TypeBool,
				},
				"enable_stackdriver_metrics": {
					Description: "The enable stackdriver monitoring for the cluster",
					Optional:    true,
					Type:        schema.TypeBool,
				},
				"master_ipv4_cidr_block": {
					Description: "The master ipv4 cidr block for the master ipv4",
					Optional:    true,
					Type:        schema.TypeString,
				},
			},
		},
	}
}

// addClusterNodePoolSchema adds the cluster node pool schema
func addClusterNodePoolSchema() *schema.Schema {
	return &schema.Schema{
		Description: "The node pools for the plan",
		Required:    true,
		Type:        schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"autoscaling": {
					Description: "The autoscaling configuration for the node pool",
					Optional:    true,
					MaxItems:    1,
					Type:        schema.TypeList,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"enabled": {
								Description: "Whether autoscaling is enabled",
								Required:    true,
								Type:        schema.TypeBool,
							},
							"max_size": {
								Description: "The maximum size of the node pool",
								Required:    true,
								Type:        schema.TypeInt,
							},
							"min_size": {
								Description: "The minimum size of the node pool",
								Required:    true,
								Type:        schema.TypeInt,
							},
						},
					},
				},
				"disk_size": {
					Description: "The size of the disk for the node pool",
					Required:    true,
					Type:        schema.TypeInt,
				},
				"image": {
					Description: "The image for the node pool",
					Required:    true,
					Type:        schema.TypeString,
				},
				"machine": {
					Description: "The machine type for the node pool",
					Required:    true,
					Type:        schema.TypeString,
				},
				"name": {
					Description: "The name of the node pool",
					Required:    true,
					Type:        schema.TypeString,
				},
				"provider_details": {
					Description: "The provider specific details for the node pool",
					Optional:    true,
					Type:        schema.TypeList,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"type": {
								Description: "The type of the provider details",
								Required:    true,
								Type:        schema.TypeString,
							},
							"eks": {
								Description: "The EKS specific details",
								Optional:    true,
								MaxItems:    1,
								Type:        schema.TypeList,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"ami": {
											Description: "The AMI to use for the node pool",
											Optional:    true,
											Type:        schema.TypeString,
										},
										"ec2_ssh_key": {
											Description: "The EC2 SSH key to use for the node pool",
											Optional:    true,
											Type:        schema.TypeString,
										},
										"ssh_source_security_group": {
											Description: "The SSH source security group to use for the node pool",
											Optional:    true,
											Type:        schema.TypeList,
											Elem:        &schema.Schema{Type: schema.TypeString},
										},
										"user_data": {
											Description: "User data provided to the compute instances",
											Optional:    true,
											Type:        schema.TypeString,
										},
									},
								},
							},
							"aks": {
								Description: "The Azure specific details for the node pool",
								Optional:    true,
								MaxItems:    1,
								Type:        schema.TypeList,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"mode": {
											Description: "The mode of the node pool",
											Required:    true,
											Type:        schema.TypeString,
										},
									},
								},
							},
							"gke": {
								Description: "The GKE specific details for the node pool",
								Optional:    true,
								MaxItems:    1,
								Type:        schema.TypeList,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"enable_autorepair": {
											Description: "Whether to enable auto repair for the node pool",
											Optional:    true,
											Type:        schema.TypeBool,
										},
										"enable_autoupgrade": {
											Description: "Whether to enable auto upgrade for the node pool",
											Optional:    true,
											Type:        schema.TypeBool,
										},
									},
								},
							},
						},
					},
				},
				"size": {
					Description: "Initial size of the node pool",
					Required:    true,
					Type:        schema.TypeInt,
				},
			},
		},
	}
}

// addClusterNetworkSchema adds the cluster network schema
func addClusterNetworkSchema() *schema.Schema {
	return &schema.Schema{
		Description: "The networking configuration for the plan",
		Required:    true,
		Type:        schema.TypeList,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"authorized_master_networks": {
					Description: "A list of CIDR ranges which are authorized to access the master api",
					Optional:    true,
					Type:        schema.TypeList,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"cidr": {
								Description: "The CIDR range",
								Required:    true,
								Type:        schema.TypeString,
							},
							"name": {
								Description: "The name of the CIDR range",
								Required:    true,
								Type:        schema.TypeString,
							},
						},
					},
				},
				"authorized_networks": {
					Description: "A list of CIDR ranges which are authorized to access the cluster",
					Optional:    true,
					Type:        schema.TypeList,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"cidr": {
								Description: "The CIDR range",
								Required:    true,
								Type:        schema.TypeString,
							},
							"name": {
								Description: "The name of the CIDR range",
								Required:    true,
								Type:        schema.TypeString,
							},
						},
					},
				},
				"cluster_ipv4_cidr": {
					Description: "The CIDR range for the cluster",
					Required:    true,
					Type:        schema.TypeString,
				},
			},
		},
	}
}

// addClusterAllocationSchema adds the cluster allocation schema
func addClusterAllocationSchema() *schema.Schema {
	return &schema.Schema{
		Description: "The allocation of the plan",
		Optional:    true,
		Type:        schema.TypeList,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": {
					Description: "The type of allocation",
					Required:    true,
					Type:        schema.TypeString,
				},
				"workspaces": {
					Description: "A filter to constrain which workspaces the plan can be used in",
					Optional:    true,
					Type:        schema.TypeList,
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
}
