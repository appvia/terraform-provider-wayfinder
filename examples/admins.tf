
resource "wayfinder_wayfindergroups" "admin" {
  metadata {
    name = "admin"
  }

  spec {
    description = "Administrators"

    members {
      kind = "User"
      name = "localadmin"
    }
    members {
      kind = "User"
      name = "kashif.saadat@appvia.io"
    }
    members {
      kind = "User"
      name = "rohith.jayawardene@appvia.io"
    }
    members {
      kind = "User"
      name = "marcin.ciszak@appvia.io"
    }
  }
}
