
terraform {
  required_providers {
    wayfinder = {
      source  = "github.com/appvia/wayfinder"
      version = "0.0.1"
    }
  }
}

#
## Define the Wayfinder provider
#
provider "wayfinder" {
  ## Endpoint is the URL of the Wayfinder API server
  #endpoint = "http://localhost:8080"

  ## You can configure the provider to use a token (see: )
  # refresh_token: <TOKEN>

  ## Alternatively, you can configure the provider to use
  ## a local Wayfinder configuration file
  # config_file = "/home/jest/.wayfinder/config"
}

resource "wayfinder_workspaces" "demo" {
  metadata {
    name = "demo"
  }

  spec {
    description = "my demo workspace"
    key         = "demo"

    owners {
      kind = "User"
      name = "marcin.ciszak@appvia.io"
    }
    summary = "my demo workspace"
  }
}

resource "wayfinder_workspaces" "apps" {
  metadata {
    name = "apps"
  }

  spec {
    description = "my apps1 workspace"
    key         = "apps1"

    owners {
      kind = "User"
      name = "marcin.ciszak@appvia.io"
    }
    summary = "my apps1 workspace"
  }
}

#
## Create a workspace work for demo
#
resource "wayfinder_workspacegroup" "demo" {
  depends_on = [
    wayfinder_workspaces.demo,
  ]

  metadata {
    name      = "members"
    namespace = "ws-demo"
  }

  spec {
    description = "Member of the workspace. No workspace management permissions"

    members {
      kind = "User"
      name = "marcin.ciszak@appvia.io"
    }
  }
}

#
## Create a workspace editor group for demo
#
resource "wayfinder_workspacegroup" "editor" {
  depends_on = [
    wayfinder_workspaces.demo,
  ]

  metadata {
    name      = "editors"
    namespace = "ws-demo"
  }

  spec {
    description = "Member of the workspace. No workspace management permissions"

    members {
      kind = "User"
      name = "marcin.ciszak@appvia.io"
    }
  }
}
