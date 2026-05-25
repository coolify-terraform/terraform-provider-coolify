# Acceptance test for ACME Corp Day-2 Operations.
#
# Tests: project, database, docker_image application, resource_action (restart).
#
# Required variables via TF_VAR_*:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

run "create_and_manage" {
  command = apply

  # --- Project ---
  assert {
    condition     = coolify_project.ops.name == "acme-day2-ops"
    error_message = "Project name mismatch: got ${coolify_project.ops.name}"
  }

  # --- PostgreSQL Database ---
  assert {
    condition     = coolify_database_postgresql.db.name == "acme-ops-db"
    error_message = "Database name mismatch: got ${coolify_database_postgresql.db.name}"
  }
  assert {
    condition     = coolify_database_postgresql.db.postgres_user == "ops"
    error_message = "Database user mismatch: got ${coolify_database_postgresql.db.postgres_user}"
  }

  # --- Docker Image Application ---
  assert {
    condition     = coolify_application_docker_image.web.name == "acme-ops-web"
    error_message = "Application name mismatch: got ${coolify_application_docker_image.web.name}"
  }
  assert {
    condition     = coolify_application_docker_image.web.docker_image == "nginx:latest"
    error_message = "Docker image mismatch: got ${coolify_application_docker_image.web.docker_image}"
  }
  assert {
    condition     = coolify_application_docker_image.web.ports_exposes == "80"
    error_message = "Ports mismatch: got ${coolify_application_docker_image.web.ports_exposes}"
  }

  # --- Resource Action: Restart ---
  assert {
    condition     = coolify_resource_action.restart_app.action == "restart"
    error_message = "Action mismatch: got ${coolify_resource_action.restart_app.action}"
  }
  assert {
    condition     = coolify_resource_action.restart_app.resource_type == "application"
    error_message = "Resource type mismatch: got ${coolify_resource_action.restart_app.resource_type}"
  }
}

# Update: change description and verify the update is applied.
run "update_project_description" {
  command = apply

  variables {
    project_description = "Updated by scenario test"
  }

  assert {
    condition     = coolify_project.ops.description == "Updated by scenario test"
    error_message = "Project description not updated: got ${coolify_project.ops.description}"
  }
}

# Idempotency: re-plan should produce no changes.
run "idempotency" {
  command = plan

  variables {
    project_description = "Updated by scenario test"
  }

  assert {
    condition     = coolify_project.ops.name == "acme-day2-ops"
    error_message = "Project name changed after re-plan (state corruption)"
  }
}
