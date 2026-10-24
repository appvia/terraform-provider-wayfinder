# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

# Read the ingest URL of a webhook. Generate it first with
# `wf rotate webhook alerts`.
ephemeral "wayfinder_webhook_ingest_url" "alerts" {
  name      = "alerts"
  workspace = "plat"
}

# Pass the URL to a write-only attribute so it never touches state.
# Bump value_wo_version after rotating the URL to send the new one.
resource "wayfinder_secret" "alerts_ingest_url" {
  key              = "alerts_ingest_url"
  value_wo         = jsonencode(ephemeral.wayfinder_webhook_ingest_url.alerts.url)
  value_wo_version = "1"
}
