data "coolify_hostinger_post_install_scripts" "all" {
  cloud_provider_token_uuid = coolify_cloud_token.hostinger.uuid
}
