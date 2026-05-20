# Validate a cloud provider token before provisioning servers.
resource "coolify_cloud_token_validate" "hetzner" {
  cloud_token_uuid = coolify_cloud_token.hetzner.uuid
}
