# ACME Corp Environment Variables at Scale
#
# Demonstrates managing environment variables across multiple applications
# using coolify_envs_bulk for atomic bulk operations:
# 1. Define shared variables (database URL, log level, app environment)
# 2. Use locals + merge() for shared-plus-per-app variable patterns
# 3. Atomic bulk updates ensure all vars are set in a single API call
#
# This scenario answers: "How do I manage environment variables across
# many applications without repeating configuration?"

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

resource "coolify_project" "env_scale" {
  name        = "acme-env-scale"
  description = var.project_description
}

# --- Shared Database (source of connection string) ---

resource "coolify_database_postgresql" "shared" {
  name             = "acme-shared-db"
  project_uuid     = coolify_project.env_scale.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  image            = "postgres:16"
  postgres_user    = "acme"
  postgres_db      = "acme_db"
  is_public        = false
}

# --- Shared Environment Variables ---

locals {
  shared_env = {
    APP_ENV   = var.app_environment
    LOG_LEVEL = var.log_level
    DB_HOST   = coolify_database_postgresql.shared.internal_db_url
  }
}

# --- Docker Image Applications ---

resource "coolify_application_docker_image" "api" {
  name             = "acme-api"
  project_uuid     = coolify_project.env_scale.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  docker_image     = "nginx:latest"
  ports_exposes    = "80"
}

resource "coolify_application_docker_image" "worker" {
  name             = "acme-worker"
  project_uuid     = coolify_project.env_scale.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  docker_image     = "redis:7-alpine"
  ports_exposes    = "6379"
}

# --- Bulk Environment Variables (atomic per-app) ---

resource "coolify_envs_bulk" "api" {
  resource_type = "application"
  resource_uuid = coolify_application_docker_image.api.uuid
  variables = merge(local.shared_env, {
    API_PORT    = "3000"
    API_WORKERS = "4"
  })
}

resource "coolify_envs_bulk" "worker" {
  resource_type = "application"
  resource_uuid = coolify_application_docker_image.worker.uuid
  variables = merge(local.shared_env, {
    WORKER_CONCURRENCY = "10"
    QUEUE_NAME         = "default"
  })
}
