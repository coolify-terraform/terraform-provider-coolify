# Plan-only test for the ACME Corp Hetzner Cloud Infrastructure scenario.
#
# This scenario provisions real Hetzner Cloud servers and incurs costs,
# so the test validates HCL syntax, resource graph, variable types, and
# schema correctness via plan only.
#
# Required variables are provided via TF_VAR_* environment variables:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token

variables {
  hetzner_api_token = "test-hetzner-token-for-plan"
  deploy_ssh_key    = "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key-for-plan-only\n-----END OPENSSH PRIVATE KEY-----"
}

run "plan_only" {
  command = plan

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
  assert {
    condition     = coolify_server_hetzner.production.location == "fsn1"
    error_message = "Production server location mismatch: got ${coolify_server_hetzner.production.location}"
  }
  assert {
    condition     = coolify_server_hetzner.production.image == "ubuntu-24.04"
    error_message = "Production server image mismatch: got ${coolify_server_hetzner.production.image}"
  }

  # --- Build Server ---
  assert {
    condition     = coolify_server_hetzner.build.name == "acme-build"
    error_message = "Build server name mismatch: got ${coolify_server_hetzner.build.name}"
  }
  assert {
    condition     = coolify_server_hetzner.build.is_build_server == true
    error_message = "Build server is_build_server should be true"
  }
}
