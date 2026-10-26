data "coolify_hostinger_data_centers" "all" {
  cloud_provider_token_uuid = coolify_cloud_token.hostinger.uuid
}
