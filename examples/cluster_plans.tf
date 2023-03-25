
resource "wayfinder_clusterplans" "test" {
  metadata {
    name = "eks-gpu-standard-tf"
  }

  spec {
    summary = "Provisions a cluster with GPU enabled node pools"

    allocation {
      type       = "all"
      workspaces = ["*"]
    }

    default_network_fabric_plan = "eks-standard"
    labels = {
      "nvidia.com/gpu" = "gpu"
    }

    template {
      stage    = ""
      channel  = "default"
      plan     = "eks-gpu-standard-tf"
      provider = "EKS"
      region   = "eu-west-2"
      version  = "1.24.8"

      provider_details {
        type = "EKS"
      }

      networking {
        authorized_master_networks {
          name = "default"
          cidr = "0.0.0.0/0"
        }

        authorized_networks {
          name = "default"
          cidr = "0.0.0.0/0"
        }
      }

      security {
        pod_security_standard {
          enabled = true
          allowed = [
            "baseline",
            "restricted",
          ]
          default_profile = "baseline"
        }
      }

      quota_limits {
        default = "small"

        templates {
          name = "small"

          constraints {
            hard_quota = {
              "requests.cpu"    = "1"
              "requests.memory" = "2Gi"
            }

            limit_ranges {
              type = "Pod"
              max = {
                "cpu"               = "256m"
                "memory"            = "512Mi"
                "ephemeral-storage" = "2.25Gi"
              }
            }
          }

          resource_defaults {
            type = "Container"
            default = {
              "cpu"               = "256m"
              "memory"            = "512Mi"
              "ephemeral-storage" = "10Mi"
            }
            default_request = {
              "cpu"    = "50m"
              "memory" = "200m"
            }
          }
        }
      }

      node_pools {
        autoscaling {
          enabled  = true
          min_size = 1
          max_size = 10
        }
        provider_details {
          type = "EKS"
        }

        disk_size    = 30
        image        = "AL2_x86_64"
        logical_name = "gpu"
        machine      = "g5.xlarge"
        size         = 1

        taints {
          key    = "nvidia.com/gpu"
          value  = "true"
          effect = "NoSchedule"
        }
      }

      node_pools {
        autoscaling {
          enabled  = true
          min_size = 1
          max_size = 10
        }
        provider_details {
          type = "EKS"
        }

        disk_size    = 30
        image        = "AL2_x86_64"
        logical_name = "compute"
        machine      = "t3.large"
        size         = 1
      }
    }
  }
}
