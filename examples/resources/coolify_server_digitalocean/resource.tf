resource "coolify_cloud_token" "do" {
  name           = "digitalocean"
  cloud_provider = "digitalocean"
  token          = var.digitalocean_token
}

resource "coolify_server_digitalocean" "app" {
  name                      = "do-app-node"
  cloud_provider_token_uuid = coolify_cloud_token.do.uuid
  region                    = "nyc1"
  size                      = "s-1vcpu-1gb"
  image                     = "ubuntu-24-04-x64"
  private_key_uuid          = coolify_private_key.main.uuid
}

variable "digitalocean_token" {
  type      = string
  sensitive = true
}
