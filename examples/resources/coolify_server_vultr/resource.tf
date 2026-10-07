resource "coolify_cloud_token" "vultr" {
  name           = "vultr"
  cloud_provider = "vultr"
  token          = var.vultr_token
}

resource "coolify_server_vultr" "app" {
  name                      = "vultr-app-node"
  cloud_provider_token_uuid = coolify_cloud_token.vultr.uuid
  region                    = "ewr"
  plan                      = "vc2-1c-1gb"
  os_id                     = 1743
  private_key_uuid          = coolify_private_key.main.uuid

  # enable_ipv6 and disable_public_ipv4 are create-only. Changing either value recreates the server.
  # enable_ipv6          = true
  # disable_public_ipv4 = false
}

variable "vultr_token" {
  type      = string
  sensitive = true
}
