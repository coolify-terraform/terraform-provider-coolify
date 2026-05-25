# ACME Corp Day-2 Operations
#
# Demonstrates operational lifecycle management:
# 1. Create infrastructure (database + application)
# 2. Restart the application using coolify_resource_action
# 3. Use the `triggers` map to force re-execution on config changes
#
# This scenario answers: "How do I programmatically restart or stop
# resources after they are deployed?"

terraform {
  required_providers {
    coolify = {
      source = "coolify-terraform/coolify"
    }
  }
}

provider "coolify" {
  endpoint = var.coolify_endpoint
  token    = var.coolify_token
}

# --- Project ---

resource "coolify_project" "ops" {
  name        = "acme-day2-ops"
  description = var.project_description
}

# --- Database ---

resource "coolify_database_postgresql" "db" {
  name             = "acme-ops-db"
  project_uuid     = coolify_project.ops.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  image            = "postgres:16"
  postgres_user    = "ops"
  postgres_db      = "ops_db"
  is_public        = false
}

# --- Application ---

resource "coolify_application_docker_image" "web" {
  name             = "acme-ops-web"
  project_uuid     = coolify_project.ops.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  docker_image     = "nginx:latest"
  ports_exposes    = "80"
}

# --- Lifecycle Action: Restart Application ---
#
# Changing `app_version` in the triggers map forces a destroy+recreate,
# which re-fires the restart action. This pattern lets you roll out
# configuration changes by bumping a version string.
#
# TIP: You can also use action = "stop" or action = "start" for
# maintenance windows. Be aware that stop has no undo on destroy --
# Terraform simply removes the action from state.

resource "coolify_resource_action" "restart_app" {
  resource_uuid = coolify_application_docker_image.web.uuid
  resource_type = "application"
  action        = "restart"

  triggers = {
    app_version = var.app_version
  }
}
