# Acceptance test for the ACME Corp Order Processing API scenario.
#
# Required variables are provided via TF_VAR_* environment variables:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

run "create_and_verify" {
  command = apply

  assert {
    condition     = coolify_project.acme.uuid != ""
    error_message = "Project was not created: uuid is empty"
  }

  assert {
    condition     = coolify_postgresql_database.orders.uuid != ""
    error_message = "PostgreSQL database was not created: uuid is empty"
  }

  assert {
    condition     = coolify_redis_database.queue.uuid != ""
    error_message = "Redis database was not created: uuid is empty"
  }

  assert {
    condition     = coolify_dockerfile_application.api.uuid != ""
    error_message = "API application was not created: uuid is empty"
  }

  assert {
    condition     = coolify_docker_image_application.worker.uuid != ""
    error_message = "Worker application was not created: uuid is empty"
  }

  assert {
    condition     = coolify_environment_variable.api_db_url.uuid != ""
    error_message = "API DATABASE_URL env var was not created: uuid is empty"
  }

  assert {
    condition     = coolify_environment_variable.api_redis_url.uuid != ""
    error_message = "API REDIS_URL env var was not created: uuid is empty"
  }

  assert {
    condition     = coolify_environment_variable.worker_db_url.uuid != ""
    error_message = "Worker DATABASE_URL env var was not created: uuid is empty"
  }

  assert {
    condition     = coolify_environment_variable.worker_redis_url.uuid != ""
    error_message = "Worker REDIS_URL env var was not created: uuid is empty"
  }

  assert {
    condition     = coolify_scheduled_task.cleanup.uuid != ""
    error_message = "Scheduled task was not created: uuid is empty"
  }

  assert {
    condition     = coolify_database_backup.orders.uuid != ""
    error_message = "Database backup was not created: uuid is empty"
  }
}
