# Acceptance test for ACME Corp external integrations.
#
# Tests: github_app (with dummy credentials), service (from catalog).
#
# Required variables via TF_VAR_*:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

run "create_and_verify" {
  command = apply

  # --- Project ---
  assert {
    condition     = coolify_project.integrations.name == "acme-integrations"
    error_message = "Project name mismatch: got ${coolify_project.integrations.name}"
  }

  # --- GitHub App ---
  assert {
    condition     = coolify_github_app.acme.name == "acme-github-integration"
    error_message = "GitHub App name mismatch: got ${coolify_github_app.acme.name}"
  }
  assert {
    condition     = coolify_github_app.acme.app_id == 12345
    error_message = "GitHub App app_id mismatch"
  }
  assert {
    condition     = coolify_github_app.acme.installation_id == 67890
    error_message = "GitHub App installation_id mismatch"
  }

  # --- Service ---
  assert {
    condition     = coolify_service.uptime.uuid != ""
    error_message = "Service was not created: uuid is empty"
  }
  assert {
    condition     = coolify_service.uptime.type == "uptime-kuma"
    error_message = "Service type mismatch: got ${coolify_service.uptime.type}"
  }
}

# Idempotency: re-plan should produce no changes.
run "idempotency" {
  command = plan

  assert {
    condition     = coolify_project.integrations.name == "acme-integrations"
    error_message = "Project name changed after re-plan (state corruption)"
  }
}