# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

# Rotate the ingest URL of a webhook. Creating this resource issues the first
# URL, and any change to triggers issues a new one.
resource "wayfinder_webhook_rotation" "alerts" {
  webhook   = wayfinder_webhook.alerts.name
  workspace = "plat"

  # Rotate every 90 days. Without the time provider, use a fixed value such as
  # triggers = { rotation = "1" } and change it to rotate on demand.
  triggers = {
    rotation = time_rotating.alerts.id
  }
}

resource "time_rotating" "alerts" {
  rotation_days = 90
}

# The URL is read after the rotation, in the same apply, and never stored.
ephemeral "wayfinder_webhook_ingest_url" "alerts" {
  name      = wayfinder_webhook_rotation.alerts.name
  workspace = "plat"
}

# Keying value_wo_version on rotation_id sends the new URL exactly when it changes.
resource "wayfinder_secret" "alerts_ingest_url" {
  key              = "alerts_ingest_url"
  value_wo         = jsonencode(ephemeral.wayfinder_webhook_ingest_url.alerts.url)
  value_wo_version = wayfinder_webhook_rotation.alerts.rotation_id
}
