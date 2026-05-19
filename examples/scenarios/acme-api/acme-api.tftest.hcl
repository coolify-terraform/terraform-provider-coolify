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
    condition     = coolify_database_postgresql.orders.name == "acme-orders-db"
    error_message = "PostgreSQL name mismatch: got ${coolify_database_postgresql.orders.name}"
  }
  assert {
    condition     = coolify_database_postgresql.orders.postgres_user == "orders"
    error_message = "PostgreSQL user mismatch: got ${coolify_database_postgresql.orders.postgres_user}"
  }
  assert {
    condition     = coolify_database_redis.queue.name == "acme-queue"
    error_message = "Redis name mismatch: got ${coolify_database_redis.queue.name}"
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

run "verify_data_sources" {
  command = apply

  assert {
    condition     = data.coolify_project.verify.name == coolify_project.acme.name
    error_message = "Project data source name does not match resource"
  }
  assert {
    condition     = data.coolify_application.verify_api.name == coolify_dockerfile_application.api.name
    error_message = "API application data source name does not match resource"
  }
}

# Update: change description and verify the update is applied.
run "update_project_description" {
  command = apply

  variables {
    project_description = "Updated by scenario test"
  }

  assert {
    condition     = coolify_project.acme.description == "Updated by scenario test"
    error_message = "Project description not updated: got ${coolify_project.acme.description}"
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
