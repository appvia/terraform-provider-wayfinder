# Terraform Provider for Wayfinder

Manage your Wayfinder configuration as code with Terraform or OpenTofu: workspaces, environments, groups and role bindings, service accounts, plans, integrations, variables, secrets and more.

Most resources are generated from the Wayfinder API, so their attributes follow the Wayfinder resource reference. Kinds without a dedicated resource can be managed with `wayfinder_manifest`.

Documentation for every resource and data source is in [`docs/`](docs/index.md) and on the Terraform Registry.

Licensed under the Apache License 2.0.

## Requirements

Terraform 1.11 or later, or OpenTofu 1.11 or later. Secrets are sent to Wayfinder through write-only attributes, which need those versions.

## Installation

```hcl
terraform {
  required_version = ">= 1.11"

  required_providers {
    wayfinder = {
      source = "appvia/wayfinder"
    }
  }
}
```

The same block works with OpenTofu.

## Authentication

The provider reads its settings from the provider block or from `WAYFINDER_*` environment variables. It uses the first credential it finds, in this order:

1. `WAYFINDER_TOKEN`: an access token, or a `wf` refresh token, which is exchanged for access tokens as needed.
2. `WAYFINDER_SERVICE_ACCOUNT` (`tenant:name` or `tenant:workspace:name`) with a federated identity token. In GitHub Actions with `permissions: id-token: write`, the OIDC token is picked up automatically.
3. `WAYFINDER_CLIENT_ID` and `WAYFINDER_CLIENT_SECRET`: service account client credentials.
4. The signed-in `wf` CLI session (`use_cli`, on by default), used only when none of the above is set. `WAYFINDER_CLI_PROFILE` (`cli_profile`) picks the profile, otherwise the CLI's current profile is used. The profile's server, tenant and CA certificate apply unless you set them explicitly.

`WAYFINDER_SERVER` defaults to `https://api.on.wayfinder.run`. `WAYFINDER_TENANT` can usually be left unset, because it's derived from the service account or client ID. On your own machine, signing in with `wf login` is enough. In CI, set the variables explicitly (`use_cli = false` or `WAYFINDER_USE_CLI=false` turns the CLI session off). See [the provider documentation](docs/index.md#authentication) for the full list.

## Example

```hcl
provider "wayfinder" {}

resource "wayfinder_workspace" "platform" {
  name        = "plat"
  description = "Platform team"
}

resource "wayfinder_variable" "region" {
  key       = "default_region"
  workspace = wayfinder_workspace.platform.name
  value     = jsonencode("eu-west-2")
}

variable "db_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "wayfinder_secret" "db" {
  key              = "db_password"
  workspace        = wayfinder_workspace.platform.name
  value_wo         = jsonencode(var.db_password) # never stored in state
  value_wo_version = "1"                         # bump to send a new value
}
```

## Compatibility

The provider is built from the Wayfinder release recorded in `schema/wayfinder-manifest.json` (`wayfinderVersion`).

## Contributing

Resources are generated from the Wayfinder release pinned in `WAYFINDER_REF`. Open issues rather than PRs that edit generated files. See [DEVELOPMENT.md](DEVELOPMENT.md) for how generation, overrides and the Wayfinder pin work, and [RELEASING.md](RELEASING.md) for how releases are published.
