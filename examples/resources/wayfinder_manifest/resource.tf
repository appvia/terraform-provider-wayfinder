resource "wayfinder_manifest" "developers" {
  manifest = yamlencode({
    apiVersion = "org/v3"
    kind       = "Group"
    metadata = {
      name = "developers"
    }
    spec = {
      description = "Development team members"
      members = [
        {
          kind = "User"
          name = "alice@example.com"
        },
      ]
    }
  })
}

variable "webhook_secret" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "wayfinder_manifest" "github_webhook" {
  manifest = yamlencode({
    apiVersion = "integrations/v3"
    kind       = "Webhook"
    metadata   = { name = "github", workspace = "plat" }
    spec = {
      integrationRef = "github"
      security       = { algorithm = "hmac-sha256", signatureHeader = "X-Hub-Signature-256" }
    }
  })
  manifest_wo = yamlencode({
    spec = { security = { secret = var.webhook_secret } } # never stored in state
  })
  manifest_wo_version = {
    "spec.security.secret" = "1" # bump to send a new secret
  }
}
