# Microservices example: API + background worker + PostgreSQL + Redis
#
# Provisions:
# 1. A project to group all resources
# 2. PostgreSQL database for the API
# 3. Redis cache for sessions and queues
# 4. API application from a public Git repository
# 5. Worker application from a Docker image
# 6. Environment variables connecting all services
# 7. Automated database backup

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

# --- Variables ---

variable "coolify_endpoint" {
  description = "Coolify API endpoint"
  type        = string
}

variable "coolify_token" {
  description = "Coolify API token"
  type        = string
  sensitive   = true
}

variable "server_uuid" {
  description = "UUID of the server to deploy to"
  type        = string
}

variable "domain" {
  description = "Domain name for the API (e.g. api.example.com)"
  type        = string
  default     = ""
}

# --- Project ---

resource "coolify_project" "ecommerce" {
  name        = "ecommerce-platform"
  description = "E-commerce microservices platform"
}

# --- Databases ---

resource "coolify_postgresql_database" "api_db" {
  name              = "ecommerce-db"
  project_uuid      = coolify_project.ecommerce.uuid
  server_uuid       = var.server_uuid
  environment_name  = "production"
  image             = "postgres:16"
  postgres_user     = "ecommerce"
  postgres_password = "change-me-in-production"
  postgres_db       = "ecommerce"
}

resource "coolify_redis_database" "cache" {
  name         = "app-cache"
  project_uuid = coolify_project.ecommerce.uuid
  server_uuid  = var.server_uuid
  image        = "redis:7-alpine"
}

# --- Database Backup ---

resource "coolify_database_backup" "daily" {
  database_uuid         = coolify_postgresql_database.api_db.uuid
  frequency             = "0 3 * * *"
  enabled               = true
  retain_amount_locally = 14
}

# --- Applications ---

resource "coolify_application" "api" {
  name             = "ecommerce-api"
  project_uuid     = coolify_project.ecommerce.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  git_repository   = "https://github.com/example/ecommerce-api"
  git_branch       = "main"
  build_pack       = "nixpacks"
  ports_exposes    = "8080"
  fqdn             = var.domain != "" ? "https://${var.domain}" : null
}

resource "coolify_docker_image_application" "worker" {
  name             = "order-worker"
  project_uuid     = coolify_project.ecommerce.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  docker_image     = "ghcr.io/example/order-worker:latest"
  ports_exposes    = "9090"
}

# --- Environment Variables ---
# In Coolify, the resource 'name' is also the Docker container name,
# which serves as the internal hostname for service-to-service communication.

resource "coolify_environment_variable" "api_database_url" {
  application_uuid = coolify_application.api.uuid
  key              = "DATABASE_URL"
  value            = "postgresql://${coolify_postgresql_database.api_db.postgres_user}:${coolify_postgresql_database.api_db.postgres_password}@${coolify_postgresql_database.api_db.name}:5432/${coolify_postgresql_database.api_db.postgres_db}"
}

resource "coolify_environment_variable" "api_redis_url" {
  application_uuid = coolify_application.api.uuid
  key              = "REDIS_URL"
  value            = "redis://${coolify_redis_database.cache.name}:6379"
}

resource "coolify_environment_variable" "worker_database_url" {
  application_uuid = coolify_docker_image_application.worker.uuid
  key              = "DATABASE_URL"
  value            = "postgresql://${coolify_postgresql_database.api_db.postgres_user}:${coolify_postgresql_database.api_db.postgres_password}@${coolify_postgresql_database.api_db.name}:5432/${coolify_postgresql_database.api_db.postgres_db}"
}

resource "coolify_environment_variable" "worker_redis_url" {
  application_uuid = coolify_docker_image_application.worker.uuid
  key              = "REDIS_URL"
  value            = "redis://${coolify_redis_database.cache.name}:6379"
}

resource "coolify_environment_variable" "worker_api_url" {
  application_uuid = coolify_docker_image_application.worker.uuid
  key              = "API_URL"
  value            = var.domain != "" ? "https://${var.domain}" : "http://${coolify_application.api.name}:8080"
}

# --- Outputs ---

output "project_uuid" {
  value = coolify_project.ecommerce.uuid
}

output "api_uuid" {
  value = coolify_application.api.uuid
}

output "worker_uuid" {
  value = coolify_docker_image_application.worker.uuid
}

output "database_uuid" {
  value = coolify_postgresql_database.api_db.uuid
}
