data "wayfinder_variable" "region" {
  key = "default_region"
}

output "region" {
  value = jsondecode(data.wayfinder_variable.region.value)
}
