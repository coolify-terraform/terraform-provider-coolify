data "coolify_hostinger_catalog" "all" {
  cloud_provider_token_uuid = coolify_cloud_token.hostinger.uuid
}
