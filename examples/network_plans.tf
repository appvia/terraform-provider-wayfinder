
resource "wayfinder_networkfabricplans" "eks_acme_standard" {
  metadata {
    name = "eks-acme-standard"
  }

  spec {
    summary = "General purpose AWS network."

    allocation {
      type       = "all"
      workspaces = ["*"]
    }

    template {
      plan     = "eks-acme-standard"
      provider = "aws"
      stage    = "nonprod"
      location = "eu-west-2"

      ipv_4 {
        cidr_blocks {
          type = "node"
          cidr = "10.170.0.0/19"
          size = 0
        }
      }

      layout {
        mode            = "auto"
        private_subnets = 3
        public_subnets  = 3
      }
    }
  }
}
