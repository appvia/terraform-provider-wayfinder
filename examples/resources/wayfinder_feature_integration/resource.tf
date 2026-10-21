# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

# Enable the Azure feature with its default settings.
resource "wayfinder_feature_integration" "azure" {
  name = "azure"
}

# Enable the AI feature with a parameter. Values are JSON, so use jsonencode
# with the type the feature expects (a bool here, not the string "false").
resource "wayfinder_feature_integration" "ai" {
  name = "ai"
  params = {
    enable_wayfinder = jsonencode(false)
  }
}

# Resources that need the feature should depend on it.
resource "wayfinder_azure_subscription" "dev" {
  # ...

  depends_on = [wayfinder_feature_integration.azure]
}

# Other teams rely on GitHub, so destroying this configuration leaves it on.
# Apply this setting before you run destroy.
resource "wayfinder_feature_integration" "github" {
  name                    = "github"
  keep_enabled_on_destroy = true
}
