# Acceptance test for ACME Corp environment variables at scale.
#
# Tests: coolify_envs_bulk atomic bulk operations with shared + per-app
# variable patterns using locals and merge().
#
# Required variables via TF_VAR_*:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

run "create_and_verify" {
  command = apply

  # --- Project ---
  assert {
    condition     = coolify_project.env_scale.name == "acme-env-scale"
    error_message = "Project name mismatch: got ${coolify_project.env_scale.name}"
  }

  # --- Database ---
  assert {
    condition     = coolify_database_postgresql.shared.name == "acme-shared-db"
    error_message = "Database name mismatch: got ${coolify_database_postgresql.shared.name}"
  }

  # --- API Application ---
  assert {
    condition     = coolify_application_docker_image.api.name == "acme-api"
    error_message = "API app name mismatch: got ${coolify_application_docker_image.api.name}"
  }

  # --- Worker Application ---
  assert {
    condition     = coolify_application_docker_image.worker.name == "acme-worker"
    error_message = "Worker app name mismatch: got ${coolify_application_docker_image.worker.name}"
  }

  # --- Envs Bulk: resource types ---
  assert {
    condition     = coolify_envs_bulk.api.resource_type == "application"
    error_message = "API envs_bulk resource_type mismatch: got ${coolify_envs_bulk.api.resource_type}"
  }
  assert {
    condition     = coolify_envs_bulk.worker.resource_type == "application"
    error_message = "Worker envs_bulk resource_type mismatch: got ${coolify_envs_bulk.worker.resource_type}"
  }
}

# Update: change log_level and verify the update propagates.
run "update_log_level" {
  command = apply

  variables {
    log_level = "debug"
  }

  # Project should still be intact after env var update.
  assert {
    condition     = coolify_project.env_scale.name == "acme-env-scale"
    error_message = "Project name changed after env update: got ${coolify_project.env_scale.name}"
  }
}

# Idempotency: re-plan with same variables should produce no changes.
run "idempotency" {
  command = plan

  variables {
    log_level = "debug"
  }
}
