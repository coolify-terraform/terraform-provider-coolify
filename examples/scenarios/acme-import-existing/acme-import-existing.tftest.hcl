# Acceptance test for the ACME Corp Import Existing Resources scenario.
#
# Creates resources, then verifies they can be read back via data sources
# (the same Read code path used by terraform import).
#
# Required variables are provided via TF_VAR_* environment variables:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

run "create_resources" {
  command = apply

  # --- Project ---
  assert {
    condition     = coolify_project.existing.uuid != ""
    error_message = "Project was not created: uuid is empty"
  }
  assert {
    condition     = coolify_project.existing.name == "acme-import-demo"
    error_message = "Project name mismatch: got ${coolify_project.existing.name}"
  }

  # --- Application ---
  assert {
    condition     = coolify_application_docker_image.web.uuid != ""
    error_message = "Application was not created: uuid is empty"
  }
  assert {
    condition     = coolify_application_docker_image.web.name == "acme-imported-web"
    error_message = "Application name mismatch: got ${coolify_application_docker_image.web.name}"
  }
  assert {
    condition     = coolify_application_docker_image.web.docker_image == "nginx:latest"
    error_message = "Docker image not preserved: got ${coolify_application_docker_image.web.docker_image}"
  }
  assert {
    condition     = coolify_application_docker_image.web.ports_exposes == "80"
    error_message = "Ports mismatch: got ${coolify_application_docker_image.web.ports_exposes}"
  }

  # --- Environment variable ---
  assert {
    condition     = coolify_environment_variable.app_env.key == "APP_ENV"
    error_message = "Env var key mismatch: got ${coolify_environment_variable.app_env.key}"
  }
  assert {
    condition     = nonsensitive(coolify_environment_variable.app_env.value) == "production"
    error_message = "Env var value mismatch"
  }
}

# Verify data sources can read back the created resources.
# Data source reads use the same Read code path that terraform import
# calls after ImportState, so this validates the import round-trip.
run "verify_read_back" {
  command = apply

  assert {
    condition     = data.coolify_project.verify.name == coolify_project.existing.name
    error_message = "Project data source name does not match resource"
  }
  assert {
    condition     = data.coolify_application.verify.name == coolify_application_docker_image.web.name
    error_message = "Application data source name does not match resource"
  }
  assert {
    condition     = data.coolify_server.target.uuid == var.server_uuid
    error_message = "Server data source UUID does not match input"
  }
}

# Update: change description and verify the update is applied.
run "update_project_description" {
  command = apply

  variables {
    project_description = "Updated by import scenario test"
  }

  assert {
    condition     = coolify_project.existing.description == "Updated by import scenario test"
    error_message = "Project description not updated: got ${coolify_project.existing.description}"
  }
}

# Idempotency: re-plan should produce no changes.
run "idempotency" {
  command = plan

  variables {
    project_description = "Updated by import scenario test"
  }

  assert {
    condition     = coolify_project.existing.name == "acme-import-demo"
    error_message = "Project name changed after re-plan (state corruption)"
  }
}
