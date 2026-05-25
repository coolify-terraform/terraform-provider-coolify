# Acceptance test for ACME Corp custom Docker Compose stack.
#
# Tests: custom compose service (docker_compose_raw), data source read-back.
#
# Required variables via TF_VAR_*:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

run "create_and_verify" {
  command = apply

  # --- Project ---
  assert {
    condition     = coolify_project.compose.name == "acme-compose-git"
    error_message = "Project name mismatch: got ${coolify_project.compose.name}"
  }

  # --- Service ---
  assert {
    condition     = coolify_service.stack.uuid != ""
    error_message = "Service was not created: uuid is empty"
  }
  assert {
    condition     = coolify_service.stack.name == "acme-compose-stack"
    error_message = "Service name mismatch: got ${coolify_service.stack.name}"
  }

  # --- Data Source read-back ---
  assert {
    condition     = data.coolify_service.stack.name == "acme-compose-stack"
    error_message = "Data source name mismatch: got ${data.coolify_service.stack.name}"
  }
}

# Update: change description and verify the update is applied.
run "update_project_description" {
  command = apply

  variables {
    project_description = "Updated by scenario test"
  }

  assert {
    condition     = coolify_project.compose.description == "Updated by scenario test"
    error_message = "Project description not updated: got ${coolify_project.compose.description}"
  }
}

# Idempotency: re-plan should produce no changes.
run "idempotency" {
  command = plan

  variables {
    project_description = "Updated by scenario test"
  }

  assert {
    condition     = coolify_project.compose.name == "acme-compose-git"
    error_message = "Project name changed after re-plan (state corruption)"
  }
}
