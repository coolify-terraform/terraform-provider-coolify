# ACME Corp Docker Deployments
#
# Tests Docker-based resources: Docker Image apps (with :latest normalization),
# scheduled tasks, and persistent storage.

terraform {
  required_providers {
    coolify = {
      source = "SebTardifLabs/coolify"
    }
  }
}

provider "coolify" {
  endpoint = var.coolify_endpoint
  token    = var.coolify_token
}

# --- Project ---

resource "coolify_project" "docker" {
  name        = "acme-docker"
  description = var.project_description
}

# --- Docker Image Applications ---

# Uses :latest tag to test Coolify's tag stripping normalization (quirk #2).
resource "coolify_application_docker_image" "nginx" {
  name             = "acme-nginx"
  project_uuid     = coolify_project.docker.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  docker_image     = "nginx:latest"
  ports_exposes    = "80"
}

# Uses a specific tag (no normalization needed).
resource "coolify_application_docker_image" "redis" {
  name             = "acme-redis-app"
  project_uuid     = coolify_project.docker.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  docker_image     = "redis:7-alpine"
  ports_exposes    = "6379"
}

# --- Scheduled Task (attached to nginx app) ---

resource "coolify_scheduled_task" "cleanup" {
  application_uuid = coolify_application_docker_image.nginx.uuid
  name             = "log-cleanup"
  command          = "echo 'Cleaning logs...'"
  frequency        = "0 2 * * *"
  enabled          = true
}

# --- Persistent Storage (attached to nginx app) ---

resource "coolify_storage" "static" {
  application_uuid = coolify_application_docker_image.nginx.uuid
  name             = "nginx-static"
  mount_path       = "/usr/share/nginx/html"
}