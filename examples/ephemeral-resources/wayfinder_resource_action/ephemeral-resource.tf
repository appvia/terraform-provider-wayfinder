# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

# Read a sub-resource of any Wayfinder object. This reads a webhook's
# ingest URL, which wayfinder_webhook_ingest_url also covers.
ephemeral "wayfinder_resource_action" "alerts_ingest_url" {
  type                   = "integrations/v3/Webhook"
  name                   = "alerts"
  workspace              = "plat"
  action                 = "ingesturl"
  response_export_values = ["url"]
}

# output is sensitive and ephemeral, so feed it to a write-only attribute.
resource "wayfinder_secret" "alerts_ingest_url" {
  key              = "alerts_ingest_url"
  value_wo         = jsonencode(ephemeral.wayfinder_resource_action.alerts_ingest_url.output.url)
  value_wo_version = "1"
}
