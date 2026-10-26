data "coolify_hostinger_ssh_keys" "all" {
  cloud_provider_token_uuid = coolify_cloud_token.hostinger.uuid
}
