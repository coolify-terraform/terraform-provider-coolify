data "coolify_digitalocean_regions" "all" {
  cloud_provider_token_uuid = coolify_cloud_token.do.uuid
}
