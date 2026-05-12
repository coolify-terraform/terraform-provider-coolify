# Acceptance test for the ACME Corp Order Processing API scenario.
#
# Required variables are provided via TF_VAR_* environment variables:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

run "create_and_verify" {
  command = apply

  # --- Project ---
  assert {
    condition     = coolify_project.acme.name == "acme-orders"
    error_message = "Project name mismatch: got ${coolify_project.acme.name}"
  }

  # --- Databases ---
  assert {
    condition     = coolify_postgresql_database.orders.name == "acme-orders-db"
    error_message = "PostgreSQL name mismatch: got ${coolify_postgresql_database.orders.name}"
  }
  assert {
    condition     = coolify_postgresql_database.orders.postgres_user == "orders"
    error_message = "PostgreSQL user mismatch: got ${coolify_postgresql_database.orders.postgres_user}"
  }
  assert {
    condition     = coolify_redis_database.queue.name == "acme-queue"
    error_message = "Redis name mismatch: got ${coolify_redis_database.queue.name}"
  }

  # --- Applications ---
  assert {
    condition     = coolify_dockerfile_application.api.name == "acme-orders-api"
    error_message = "API app name mismatch: got ${coolify_dockerfile_application.api.name}"
  }
  assert {
    condition     = coolify_dockerfile_application.api.ports_exposes == "3000"
    error_message = "API ports_exposes mismatch: got ${coolify_dockerfile_application.api.ports_exposes}"
  }
  assert {
    condition     = coolify_docker_image_application.worker.name == "acme-orders-worker"
    error_message = "Worker app name mismatch: got ${coolify_docker_image_application.worker.name}"
  }
  # Quirk #2: Coolify strips :latest tag. Provider must preserve original.
  assert {
    condition     = coolify_docker_image_application.worker.docker_image == "alpine:latest"
    error_message = "Docker image :latest not preserved: got ${coolify_docker_image_application.worker.docker_image}"
  }

  # --- Environment Variables ---
  assert {
    condition     = coolify_environment_variable.api_db_url.key == "DATABASE_URL"
    error_message = "API env var key mismatch: got ${coolify_environment_variable.api_db_url.key}"
  }
  assert {
    condition     = coolify_environment_variable.api_redis_url.key == "REDIS_URL"
    error_message = "API env var key mismatch: got ${coolify_environment_variable.api_redis_url.key}"
  }
  assert {
    condition     = coolify_environment_variable.worker_db_url.key == "DATABASE_URL"
    error_message = "Worker env var key mismatch: got ${coolify_environment_variable.worker_db_url.key}"
  }
  assert {
    condition     = coolify_environment_variable.worker_redis_url.key == "REDIS_URL"
    error_message = "Worker env var key mismatch: got ${coolify_environment_variable.worker_redis_url.key}"
  }

  # --- Scheduled Task ---
  assert {
    condition     = coolify_scheduled_task.cleanup.name == "nightly-cleanup"
    error_message = "Scheduled task name mismatch: got ${coolify_scheduled_task.cleanup.name}"
  }
  assert {
    condition     = coolify_scheduled_task.cleanup.frequency == "@daily"
    error_message = "Scheduled task frequency mismatch: got ${coolify_scheduled_task.cleanup.frequency}"
  }

  # --- Database Backup ---
  assert {
    condition     = coolify_database_backup.orders.uuid != ""
    error_message = "Database backup was not created: uuid is empty"
  }
  assert {
    condition     = coolify_database_backup.orders.frequency == "@daily"
    error_message = "Backup frequency mismatch: got ${coolify_database_backup.orders.frequency}"
  }
}

# Idempotency: re-plan should produce no changes.
run "idempotency" {
  command = plan

  assert {
    condition     = coolify_project.acme.name == "acme-orders"
    error_message = "Project name changed after re-plan (state corruption)"
  }
}
