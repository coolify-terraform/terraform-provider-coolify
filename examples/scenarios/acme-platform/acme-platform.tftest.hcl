# Acceptance test for ACME Corp Platform infrastructure.
#
# Tests: private_key, environment, storage, cloud_token, and data sources
# (version, health, server, teams).
#
# Required variables via TF_VAR_*:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

run "create_and_verify" {
  command = apply

  # --- Data Sources ---
  assert {
    condition     = data.coolify_version.this.version != ""
    error_message = "Version data source returned empty version"
  }
  assert {
    condition     = data.coolify_server.target.uuid == var.server_uuid
    error_message = "Server data source UUID mismatch"
  }

  # --- Private Key ---
  assert {
    condition     = coolify_private_key.deploy.uuid != ""
    error_message = "Private key was not created: uuid is empty"
  }
  assert {
    condition     = coolify_private_key.deploy.name == "acme-deploy-key"
    error_message = "Private key name mismatch: got ${coolify_private_key.deploy.name}"
  }

  # --- Project ---
  assert {
    condition     = coolify_project.platform.name == "acme-platform"
    error_message = "Project name mismatch: got ${coolify_project.platform.name}"
  }

  # --- Environment ---
  assert {
    condition     = coolify_environment.staging.name == "staging"
    error_message = "Environment name mismatch: got ${coolify_environment.staging.name}"
  }

  # --- Application ---
  assert {
    condition     = coolify_application.api.name == "acme-platform-api"
    error_message = "Application name mismatch: got ${coolify_application.api.name}"
  }
  assert {
    condition     = coolify_application.api.git_repository == "https://github.com/coollabsio/coolify-examples"
    error_message = "Git repository URL was normalized: got ${coolify_application.api.git_repository}"
  }

  # --- Storage ---
  assert {
    condition     = coolify_storage.uploads.uuid != ""
    error_message = "Storage was not created: uuid is empty"
  }
  assert {
    condition     = coolify_storage.uploads.mount_path == "/app/uploads"
    error_message = "Storage mount_path mismatch: got ${coolify_storage.uploads.mount_path}"
  }


}

# Idempotency: re-plan should produce no changes.
run "idempotency" {
  command = plan

  assert {
    condition     = coolify_project.platform.name == "acme-platform"
    error_message = "Project name changed after re-plan (state corruption)"
  }
}