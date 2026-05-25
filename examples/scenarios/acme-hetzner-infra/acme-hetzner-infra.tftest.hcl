# Acceptance test for ACME Corp Hetzner Cloud infrastructure.
#
# Tests: cloud_token, private_key, server_hetzner (production + build).
# Provisions real Hetzner Cloud servers (terraform destroy cleans up).
#
# Required variables via TF_VAR_*:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token,
#   TF_VAR_hetzner_api_token, TF_VAR_deploy_ssh_key

run "provision_and_verify" {
  command = apply

  # --- Cloud Token ---
  assert {
    condition     = coolify_cloud_token.hetzner.name == "acme-hetzner-token"
    error_message = "Cloud token name mismatch: got ${coolify_cloud_token.hetzner.name}"
  }
  assert {
    condition     = coolify_cloud_token.hetzner.cloud_provider == "hetzner"
    error_message = "Cloud provider mismatch: got ${coolify_cloud_token.hetzner.cloud_provider}"
  }

  # --- SSH Key ---
  assert {
    condition     = coolify_private_key.deploy.name == "acme-hetzner-deploy-key"
    error_message = "Private key name mismatch: got ${coolify_private_key.deploy.name}"
  }

  # --- Production Server ---
  assert {
    condition     = coolify_server_hetzner.production.name == "acme-production"
    error_message = "Production server name mismatch: got ${coolify_server_hetzner.production.name}"
  }
  assert {
    condition     = coolify_server_hetzner.production.server_type == "cx22"
    error_message = "Production server type mismatch: got ${coolify_server_hetzner.production.server_type}"
  }

  # --- Build Server ---
  assert {
    condition     = coolify_server_hetzner.build.name == "acme-build"
    error_message = "Build server name mismatch: got ${coolify_server_hetzner.build.name}"
  }
}
