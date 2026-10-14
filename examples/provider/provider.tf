terraform {
  required_version = ">= 1.11"

  required_providers {
    wayfinder = {
      source = "appvia/wayfinder"
    }
  }
}

# Configured from WAYFINDER_* environment variables, for example
# WAYFINDER_CLIENT_ID and WAYFINDER_CLIENT_SECRET.
provider "wayfinder" {}

# Or configured explicitly. Pass credentials in through variables
# rather than writing them into configuration.
variable "wayfinder_client_id" {
  type = string
}

variable "wayfinder_client_secret" {
  type      = string
  sensitive = true
  ephemeral = true
}

provider "wayfinder" {
  alias         = "explicit"
  server        = "https://api.on.wayfinder.run"
  tenant        = "my-tenant"
  client_id     = var.wayfinder_client_id
  client_secret = var.wayfinder_client_secret
}

# Or, on your own machine, use the signed-in wf CLI session (wf login).
# This is the default when no credentials are set; cli_profile picks a
# profile other than the CLI's current one.
provider "wayfinder" {
  alias       = "cli"
  cli_profile = "my-profile"
}
