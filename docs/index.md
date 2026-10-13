---
page_title: "Provider: Wayfinder"
description: |-
  Manage Wayfinder workspaces, environments, access, variables, secrets and more as code.
---

# Wayfinder Provider

The Wayfinder provider manages your Wayfinder configuration as code: workspaces, environments, groups and role bindings, service accounts, plans, integrations, variables, secrets and the rest of the objects you'd otherwise create in the Wayfinder UI or with the `wf` CLI.

Most resources are generated from the Wayfinder API, so their attributes match the Wayfinder resource reference. Kinds without a dedicated resource can be managed with [`wayfinder_manifest`](resources/manifest.md).

## Requirements

Terraform 1.11 or later, or OpenTofu 1.11 or later. Secrets are passed to Wayfinder through write-only attributes, which those versions introduced (see [Secrets](#secrets)).

## Installation

```terraform
terraform {
  required_version = ">= 1.11"

  required_providers {
    wayfinder = {
      source = "appvia/wayfinder"
    }
  }
}
```

OpenTofu uses the same block. `source = "appvia/wayfinder"` resolves through the OpenTofu registry.

## Example Usage

```terraform
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
```

## Authentication

Every provider attribute is optional and falls back to the matching environment variable:

| Attribute | Environment variable |
| --- | --- |
| `server` | `WAYFINDER_SERVER` |
| `tenant` | `WAYFINDER_TENANT` |
| `token` | `WAYFINDER_TOKEN` |
| `client_id` | `WAYFINDER_CLIENT_ID` |
| `client_secret` | `WAYFINDER_CLIENT_SECRET` |
| `service_account` | `WAYFINDER_SERVICE_ACCOUNT` |
| `federated_token_file` | `WAYFINDER_FEDERATED_TOKEN_FILE` |
| `federated_token_request_url` | `WAYFINDER_FEDERATED_TOKEN_REQUEST_URL` |
| `federated_token_request_token` | `WAYFINDER_FEDERATED_TOKEN_REQUEST_TOKEN` |
| `federated_token_request_audience` | `WAYFINDER_FEDERATED_TOKEN_REQUEST_AUDIENCE` |

`server` defaults to `https://api.on.wayfinder.run`.

The provider picks the first credential that's set, in this order:

1. `token`: an access token, used as is.
2. `service_account`: a fully qualified service account (`tenant:name` or `tenant:workspace:name`) that signs in with a federated identity token. The token is read from `federated_token_file`, or requested from `federated_token_request_url` with `federated_token_request_token`. Inside GitHub Actions, `ACTIONS_ID_TOKEN_REQUEST_URL` and `ACTIONS_ID_TOKEN_REQUEST_TOKEN` are used when no request URL is set. The audience is discovered from the server unless `federated_token_request_audience` is set.
3. `client_id` and `client_secret`: service account client credentials.

When `tenant` isn't set, it's taken from `service_account` (the part before the first `:`), or else from the `client_id` prefix before the first `:`. If both `tenant` and `service_account` are set, their tenants must match.

The provider doesn't read `wf` CLI profiles from `~/.wayfinder`. `WAYFINDER_WORKSPACE` and `WAYFINDER_ENVIRONMENT` aren't used as defaults either: set `workspace` and `environment` on each resource.

### GitHub Actions with OIDC

Give the job permission to request an ID token and name the service account. No stored secret is needed.

```yaml
permissions:
  id-token: write
  contents: read

jobs:
  apply:
    runs-on: ubuntu-latest
    env:
      WAYFINDER_SERVICE_ACCOUNT: my-tenant:platform:terraform
    steps:
      - uses: actions/checkout@v4
      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: "1.11.0"
      - run: terraform init
      - run: terraform apply -auto-approve
```

### Service account client credentials

```shell
export WAYFINDER_CLIENT_ID="my-tenant:terraform"
export WAYFINDER_CLIENT_SECRET="..." # from your secret store
terraform apply
```

## Secrets

Secret-bearing fields are write-only attributes named `<field>_wo`. Terraform sends their value to Wayfinder but never stores it in state or plan files, and Wayfinder never returns it. Each one has a companion `<field>_wo_version` attribute, which is stored in state.

The value is sent when the resource is created and again whenever `<field>_wo_version` changes. Changing only the `_wo` value does nothing, so to rotate a secret, change both:

```terraform
variable "github_token" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "wayfinder_secret" "github" {
  key              = "github_token"
  value_wo         = jsonencode(var.github_token)
  value_wo_version = "2" # bump to send a new value
}
```

Ephemeral variables and ephemeral resources can feed write-only attributes, so the secret doesn't touch state at any point. Because Wayfinder never returns secret values, changes made to them outside Terraform aren't detected.

## Waiting for readiness

After a create or update, the provider waits until Wayfinder has finished reconciling the object. A failed reconcile fails the apply with Wayfinder's message. When the object needs an approval or another action from you, the apply succeeds with a warning that says what's needed.

Set `wait_for_ready = false` on a resource to skip waiting, and use the `timeouts` block to change how long the provider waits for create, update and delete.

## Importing

Resources can be imported by ID. The ID depends on the scope of the object:

| Scope | ID |
| --- | --- |
| Tenant | `name` |
| Workspace | `workspace/name` |
| Environment | `workspace/environment/name` |

Versioned kinds, such as plans, add the version: `name@1.0.0` or `workspace/name@1.0.0`. Each resource page lists its import forms. `wayfinder_manifest`, `wayfinder_variable` and `wayfinder_secret` use their own forms, described on their pages.

## Changes made outside Terraform

Before every update, the provider checks that the object hasn't changed in Wayfinder since Terraform last wrote it. If it has, the apply stops with a "Resource changed outside Terraform" error instead of overwriting that change.

To keep the outside change, remove the resource from state, import it again, update your configuration to match, and apply:

```shell
terraform state rm wayfinder_workspace.plat
terraform import wayfinder_workspace.plat plat
```

To discard the outside change, do the same and let the next apply write your configuration back.

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `client_id` (String) Service account client ID, used with `client_secret`. Can also be set with the `WAYFINDER_CLIENT_ID` environment variable.
- `client_secret` (String, Sensitive) Service account client secret, used with `client_id`. Can also be set with the `WAYFINDER_CLIENT_SECRET` environment variable.
- `federated_token_file` (String) Path to a file holding the federated identity token exchanged for `service_account`. Can also be set with the `WAYFINDER_FEDERATED_TOKEN_FILE` environment variable.
- `federated_token_request_audience` (String) Audience requested for the federated identity token. Can also be set with the `WAYFINDER_FEDERATED_TOKEN_REQUEST_AUDIENCE` environment variable.
- `federated_token_request_token` (String, Sensitive) Bearer token for `federated_token_request_url`. Inside GitHub Actions, `ACTIONS_ID_TOKEN_REQUEST_TOKEN` is used when the request URL is unset. Can also be set with the `WAYFINDER_FEDERATED_TOKEN_REQUEST_TOKEN` environment variable.
- `federated_token_request_url` (String) URL to request the federated identity token from. Inside GitHub Actions, `ACTIONS_ID_TOKEN_REQUEST_URL` is used when this is unset. Can also be set with the `WAYFINDER_FEDERATED_TOKEN_REQUEST_URL` environment variable.
- `server` (String) Wayfinder API endpoint. Defaults to `https://api.on.wayfinder.run`. Can also be set with the `WAYFINDER_SERVER` environment variable.
- `service_account` (String) Fully qualified service account (`tenant:name` or `tenant:workspace:name`) to authenticate as with a federated identity token. Takes precedence over client credentials. Can also be set with the `WAYFINDER_SERVICE_ACCOUNT` environment variable.
- `tenant` (String) Wayfinder tenant to manage. When unset, it is derived from `service_account` (`tenant:[workspace:]name`) or from the `client_id` prefix before the first `:`. Can also be set with the `WAYFINDER_TENANT` environment variable.
- `token` (String, Sensitive) Wayfinder access token. Takes precedence over `service_account` and client credentials. Can also be set with the `WAYFINDER_TOKEN` environment variable.
