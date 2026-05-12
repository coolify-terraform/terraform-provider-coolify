# ACME Corp Order Processing API
#
# Deploys a microservice-based order processing backend:
# - REST API (Dockerfile-based)
# - Background worker (Docker image)
# - PostgreSQL for order storage
# - Redis for job queues and caching
# - Scheduled cleanup task
# - Daily database backup

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

resource "coolify_project" "acme" {
  name        = "acme-orders"
  description = "ACME Corp order processing microservices"
}

# --- Databases ---

resource "coolify_postgresql_database" "orders" {
  name             = "acme-orders-db"
  project_uuid     = coolify_project.acme.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  image            = "postgres:16"
  postgres_user    = "orders"
  postgres_db      = "acme_orders"
  is_public        = false
}

resource "coolify_redis_database" "queue" {
  name             = "acme-queue"
  project_uuid     = coolify_project.acme.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  image            = "redis:7"
}

# --- Applications ---

resource "coolify_dockerfile_application" "api" {
  name             = "acme-orders-api"
  project_uuid     = coolify_project.acme.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  dockerfile_location = base64encode(<<-DOCKERFILE
    FROM node:20-alpine
    WORKDIR /app
    COPY . .
    RUN npm install --production
    EXPOSE 3000
    CMD ["node", "server.js"]
  DOCKERFILE
  )
  ports_exposes = "3000"
}

resource "coolify_docker_image_application" "worker" {
  name             = "acme-orders-worker"
  project_uuid     = coolify_project.acme.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  docker_image     = "alpine:latest"
  ports_exposes    = "8080"
}

# --- Environment Variables ---
# In Coolify, the resource 'name' is also the Docker container name,
# which serves as the internal hostname for service-to-service communication.

resource "coolify_environment_variable" "api_db_url" {
  application_uuid = coolify_dockerfile_application.api.uuid
  key              = "DATABASE_URL"
  value            = "postgresql://${coolify_postgresql_database.orders.postgres_user}:${coolify_postgresql_database.orders.postgres_password}@${coolify_postgresql_database.orders.name}:5432/${coolify_postgresql_database.orders.postgres_db}"
  is_build         = false
  is_preview       = false
}

resource "coolify_environment_variable" "api_redis_url" {
  application_uuid = coolify_dockerfile_application.api.uuid
  key              = "REDIS_URL"
  value            = "redis://${coolify_redis_database.queue.name}:6379"
  is_build         = false
  is_preview       = false
}

resource "coolify_environment_variable" "worker_db_url" {
  application_uuid = coolify_docker_image_application.worker.uuid
  key              = "DATABASE_URL"
  value            = "postgresql://${coolify_postgresql_database.orders.postgres_user}:${coolify_postgresql_database.orders.postgres_password}@${coolify_postgresql_database.orders.name}:5432/${coolify_postgresql_database.orders.postgres_db}"
  is_build         = false
  is_preview       = false
}

resource "coolify_environment_variable" "worker_redis_url" {
  application_uuid = coolify_docker_image_application.worker.uuid
  key              = "REDIS_URL"
  value            = "redis://${coolify_redis_database.queue.name}:6379"
  is_build         = false
  is_preview       = false
}

# --- Scheduled Task ---

resource "coolify_scheduled_task" "cleanup" {
  application_uuid = coolify_dockerfile_application.api.uuid
  name             = "nightly-cleanup"
  command          = "echo 'Cleaning up old orders...'"
  frequency        = "@daily"
  enabled          = true
}

# --- Database Backup ---
# When s3_storage_id is omitted, backups are stored locally on the server.

resource "coolify_database_backup" "orders" {
  database_uuid         = coolify_postgresql_database.orders.uuid
  frequency             = "@daily"
  enabled               = true
  retain_amount_locally = 7
}
