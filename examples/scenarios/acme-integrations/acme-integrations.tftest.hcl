# Acceptance test for ACME Corp external integrations.
#
# Tests: managed service from the Coolify catalog.
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

# Update: change description and verify the update is applied.
run "update_project_description" {
  command = apply

  variables {
    project_description = "Updated by scenario test"
  }

  assert {
    condition     = coolify_project.integrations.description == "Updated by scenario test"
    error_message = "Project description not updated: got ${coolify_project.integrations.description}"
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