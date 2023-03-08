resource "wayfinder_plan" "eks-default" {
  metadata {
    name      = "eks-default"
    namespace = "wayfinder"
  }

  spec {
    summary = "General purpose EKS cluster."

    allocation {
      type       = "all"
      workspaces = ["*"]
    }
    template {
      plan    = "eks-default"
      region  = "eu-west-2"
      stage   = "dev"
      version = "latest"

      capabilities {
        cert_manager {
          description = "Cert Manager"
          enabled     = true
        }
        external_dns_as {
          description = "External DNS"
          enabled     = true
        }
        ingress_nginx {
          description = "Ingress Nginx"
          enabled     = false
        }
        psa_operator {
          description = "Pod Security Admission Operator"
          enabled     = false
        }
        terranetes_controller {
          description = "Terranetes Controller"
          enabled     = false
        }
      }

      channel  = "stable"
      provider = "EKS"

      networking {
        authorized_master_networks {
          cidr = "0.0.0.0/0"
          name = "all"
        }
        authorized_networks {
          cidr = "0.0.0.0/0"
          name = "all"
        }
        cluster_ipv4_cidr = "10.170.0.0/19"
      }

      node_pools {
        name      = "compute"
        disk_size = 30
        image     = "AL2_x86_64"
        machine   = "t3.large"
        size      = 2
        autoscaling {
          enabled  = true
          min_size = 1
          max_size = 10
        }
      }

      #      quota_limits {
      #        default = "small"
      #        templates = {
      #          {
      #            name = "small"
      #            constraints = {
      #              hard_quota = {
      #                cpu = "2"
      #                memory = "4Gi"
      #              }
      #              limit_ranges = [
      #                {
      #                  max = {
      #                    cpu = "256m"
      #                    ephemeral_storage = "2.25Gi"
      #                    memory = "512Mi"
      #                  }
      #                  type = "Pod"
      #                }
      #              ]
      #            }
      #            resource_defaults = [
      #              {
      #                default = {
      #                  cpu = "256m"
      #                  ephemeral_storage = "2.25Gi"
      #                  memory = "512Mi"
      #                }
      #              }
      #            ]
      #          }
      #        }
      #      }
      #

      security {
        pod_security_standard {
          allowed         = ["restricted", "baseline", "privileged"]
          default_profile = "baseline"
          enabled         = true
        }
      }

    }
  }
}
