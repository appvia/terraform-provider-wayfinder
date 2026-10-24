# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

# Rotate the webhook to generate its ingest URL, then read it. The rotation's
# name is unknown until it has run, so the URL is read after it, in the same apply.
resource "wayfinder_webhook_rotation" "alerts" {
  webhook   = "alerts"
  workspace = "plat"
}

ephemeral "wayfinder_webhook_ingest_url" "alerts" {
  name      = wayfinder_webhook_rotation.alerts.name
  workspace = "plat"
}

# Pass the URL to a write-only attribute so it never touches state.
# rotation_id changes on every rotation, so the new URL is sent when it rotates.
resource "wayfinder_secret" "alerts_ingest_url" {
  key              = "alerts_ingest_url"
  value_wo         = jsonencode(ephemeral.wayfinder_webhook_ingest_url.alerts.url)
  value_wo_version = wayfinder_webhook_rotation.alerts.rotation_id
}
