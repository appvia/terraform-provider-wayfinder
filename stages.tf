resource "wayfinder_stages" "uat" {
  metadata {
    name = "uat"
  }

  spec {
    description = "Used for uat environments"
    display_name = "UAT"
  }
}

resource "wayfinder_stages" "preview" {
  metadata {
    name = "preview"
  }

  spec {
    description = "Used for preview environments"
    display_name = "PREVIEW"
  }
}
