resource "coolify_integration_token" "cloudflare" {
  name                 = "cloudflare-dns"
  integration_provider = "cloudflare"
  token                = "change-me-in-production"

  # Cloudflare defaults to ["dns"] when capabilities is omitted.
  capabilities = ["dns"]
}

resource "coolify_integration_token" "vault" {
  name                 = "vault"
  integration_provider = "vault"
  token                = "change-me-in-production"
  metadata_base_url    = "https://vault.example.com"
  metadata_namespace   = "admin"
  metadata_timeout     = 10
}
