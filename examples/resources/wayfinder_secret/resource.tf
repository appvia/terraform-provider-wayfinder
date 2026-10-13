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
