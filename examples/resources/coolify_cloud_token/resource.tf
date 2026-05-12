variable "hetzner_api_token" {
  type      = string
  sensitive = true
}

resource "coolify_cloud_token" "example" {
  name           = "my-cloud-token"
  cloud_provider = "hetzner"
  token          = var.hetzner_api_token
}
