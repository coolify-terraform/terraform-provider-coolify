# ACME Corp Docker Deployments
#
# Tests Docker-based resources: Docker Image apps (with :latest normalization),
# Docker Compose apps, scheduled tasks, and persistent storage.

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
  description = "ACME Corp containerized services"
}

# --- Docker Image Applications ---

# Uses :latest tag to test Coolify's tag stripping normalization (quirk #2).
resource "coolify_docker_image_application" "nginx" {
  name             = "acme-nginx"
  project_uuid     = coolify_project.docker.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  docker_image     = "nginx:latest"
  ports_exposes    = "80"
}

# Uses a specific tag (no normalization needed).
resource "coolify_docker_image_application" "redis" {
  name             = "acme-redis-app"
  project_uuid     = coolify_project.docker.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  docker_image     = "redis:7-alpine"
  ports_exposes    = "6379"
}

# --- Docker Compose Application ---

resource "coolify_docker_compose_application" "stack" {
  name             = "acme-compose-stack"
  project_uuid     = coolify_project.docker.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  docker_compose_raw = yamlencode({
    services = {
      web = {
        image = "nginx:alpine"
        ports = ["8080:80"]
      }
    }
  })
}

# --- Scheduled Task (attached to nginx app) ---

resource "coolify_scheduled_task" "cleanup" {
  application_uuid = coolify_docker_image_application.nginx.uuid
  name             = "log-cleanup"
  command          = "echo 'Cleaning logs...'"
  frequency        = "0 2 * * *"
  enabled          = true
}

# --- Persistent Storage (attached to nginx app) ---

resource "coolify_storage" "static" {
  resource_uuid = coolify_docker_image_application.nginx.uuid
  resource_type = "application"
  fs_path       = "/data/static"
  mount_path    = "/usr/share/nginx/html"
}