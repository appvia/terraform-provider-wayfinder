# Tenant-wide variable.
resource "wayfinder_variable" "region" {
  key   = "default_region"
  value = jsonencode("eu-west-2")
}

# Variable scoped to one environment of a workspace.
resource "wayfinder_variable" "replicas" {
  key         = "replicas"
  workspace   = "plat"
  environment = "dev"
  value       = jsonencode(2)
}
