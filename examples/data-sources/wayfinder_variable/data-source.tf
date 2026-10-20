# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

data "wayfinder_variable" "region" {
  key = "default_region"
}

output "region" {
  value = jsondecode(data.wayfinder_variable.region.value)
}

# Read a key set for one stack instance in an environment.
data "wayfinder_variable" "checkout_replicas" {
  key         = "replicas"
  workspace   = "plat"
  environment = "dev"
  instance    = "checkout"
}

output "checkout_replicas" {
  value = jsondecode(data.wayfinder_variable.checkout_replicas.value)
}
