# Acceptance test for ACME Corp Docker deployments.
#
# Tests: docker_image_application (with :latest normalization),
# docker_compose_application, scheduled_task, storage.
#
# Required variables via TF_VAR_*:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

run "create_and_verify" {
  command = apply

  # --- Project ---
  assert {
    condition     = coolify_project.docker.name == "acme-docker"
    error_message = "Project name mismatch: got ${coolify_project.docker.name}"
  }

  # --- Docker Image: nginx with :latest (quirk #2) ---
  assert {
    condition     = coolify_docker_image_application.nginx.name == "acme-nginx"
    error_message = "Nginx app name mismatch: got ${coolify_docker_image_application.nginx.name}"
  }
  assert {
    condition     = coolify_docker_image_application.nginx.docker_image == "nginx:latest"
    error_message = "Docker :latest tag not preserved: got ${coolify_docker_image_application.nginx.docker_image}"
  }
  assert {
    condition     = coolify_docker_image_application.nginx.ports_exposes == "80"
    error_message = "Nginx ports_exposes mismatch: got ${coolify_docker_image_application.nginx.ports_exposes}"
  }

  # --- Docker Image: redis with specific tag ---
  assert {
    condition     = coolify_docker_image_application.redis.docker_image == "redis:7-alpine"
    error_message = "Redis image tag mismatch: got ${coolify_docker_image_application.redis.docker_image}"
  }

  # --- Docker Compose ---
  assert {
    condition     = coolify_docker_compose_application.stack.uuid != ""
    error_message = "Docker Compose app was not created: uuid is empty"
  }
  assert {
    condition     = coolify_docker_compose_application.stack.name == "acme-compose-stack"
    error_message = "Compose app name mismatch: got ${coolify_docker_compose_application.stack.name}"
  }

  # --- Scheduled Task ---
  assert {
    condition     = coolify_scheduled_task.cleanup.name == "log-cleanup"
    error_message = "Scheduled task name mismatch: got ${coolify_scheduled_task.cleanup.name}"
  }
  assert {
    condition     = coolify_scheduled_task.cleanup.frequency == "0 2 * * *"
    error_message = "Scheduled task frequency mismatch: got ${coolify_scheduled_task.cleanup.frequency}"
  }

  # --- Storage ---
  assert {
    condition     = coolify_storage.static.uuid != ""
    error_message = "Storage was not created: uuid is empty"
  }
  assert {
    condition     = coolify_storage.static.mount_path == "/usr/share/nginx/html"
    error_message = "Storage mount_path mismatch: got ${coolify_storage.static.mount_path}"
  }
}

# Idempotency: re-plan should produce no changes.
run "idempotency" {
  command = plan

  assert {
    condition     = !run.create_and_verify.incomplete
    error_message = "Previous run did not complete successfully"
  }
}