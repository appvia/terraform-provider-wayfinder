resource "wayfinder_peeringrules" "transit" {
  metadata {
    name = "transit-tf"
  }

  spec {
    filters {
      allocation {
        type = "all"
      }

      selectors {
        match_expressions {
          key      = "networking.wayfinder.io/peering"
          operator = "In"
          values   = ["true"]
        }
        match_expressions {
          key      = "appvia.io/provider"
          operator = "In"
          values   = ["terraform"]
        }
      }
    }

    connection {
      type = "gateway"

      gateway {
        enable_auto_approval = false
        identifier           = "tgw-00959620a2a44e248"
        routes = [
          "10.2.0.0/16",
          "10.32.0.0/16",
        ]

        provider_details {
          aws {
            enable_dns = true
          }
        }
      }
    }
  }
}
