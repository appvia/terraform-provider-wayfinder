# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

variable "db_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "wayfinder_secret" "db" {
  key       = "db_credentials"
  workspace = "plat"

  # Write-only: sent to Wayfinder, never stored in state.
  value_wo = jsonencode({ user = "app", password = var.db_password })

  # Change this to send a new value_wo.
  value_wo_version = "1"
}

# Secret scoped to one stack instance in an environment.
resource "wayfinder_secret" "checkout_api" {
  key         = "api_credentials"
  workspace   = "plat"
  environment = "dev"
  instance    = "checkout"

  value_wo         = jsonencode({ user = "checkout", password = var.db_password })
  value_wo_version = "1"
}
