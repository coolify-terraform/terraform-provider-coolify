resource "coolify_hetzner_server" "example" {
  name                      = "my-hetzner-server"
  cloud_provider_token_uuid = coolify_cloud_token.hetzner.uuid
  server_type               = "cx22"
  location                  = "fsn1"
  image                     = "ubuntu-24.04"
  private_key_uuid          = coolify_private_key.example.uuid

  # Optional Hetzner settings:
  # enable_ipv4       = true
  # enable_ipv6       = true
  # hetzner_ssh_key_ids = "12345,67890"
  # instant_validate  = true

  # Optional server settings (same as coolify_server):
  # is_build_server   = false
  # concurrent_builds = 2
  # dynamic_timeout   = 3600

  # Extended settings (same as coolify_server):
  # wildcard_domain    = "example.com"
  # is_metrics_enabled = true
  # is_sentinel_enabled = true
  # docker_cleanup_frequency = "0 0 * * *"
}
