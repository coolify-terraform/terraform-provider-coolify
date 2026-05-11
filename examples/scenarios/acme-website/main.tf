# ACME Corp Marketing Website
#
# Deploys a Node.js marketing site backed by PostgreSQL on Coolify.
# Optionally configures daily database backups to S3-compatible storage.

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
  name        = "acme-website"
  description = "ACME Corp marketing website and content database"
}

# --- Database ---

resource "coolify_postgresql_database" "content" {
  name          = "acme-content"
  project_uuid  = coolify_project.acme.uuid
  server_uuid   = var.server_uuid
  image         = "postgres:16"
  postgres_user = "acme"
  postgres_db   = "acme_content"
  is_public     = false
}

# --- Application ---

resource "coolify_application" "website" {
  name           = "acme-website"
  project_uuid   = coolify_project.acme.uuid
  server_uuid    = var.server_uuid
  git_repository = "https://github.com/coollabsio/coolify-examples"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"
}

# --- Environment Variables ---
# Coolify places all project resources on the same Docker network.
# The database container is reachable by its resource name ("acme-content").

resource "coolify_environment_variable" "database_url" {
  application_uuid = coolify_application.website.uuid
  key              = "DATABASE_URL"
  value            = "postgresql://${coolify_postgresql_database.content.postgres_user}:${coolify_postgresql_database.content.postgres_password}@${coolify_postgresql_database.content.name}:5432/${coolify_postgresql_database.content.postgres_db}"
  is_build         = false
  is_preview       = false
}

resource "coolify_environment_variable" "node_env" {
  application_uuid = coolify_application.website.uuid
  key              = "NODE_ENV"
  value            = "production"
  is_build         = true
  is_preview       = false
}

# --- Backups (optional) ---
# Set enable_backups = true and provide S3 credentials to activate.

resource "coolify_s3_storage" "backups" {
  count = var.enable_backups ? 1 : 0

  name       = "acme-backups"
  endpoint   = var.s3_endpoint
  bucket     = var.s3_bucket
  region     = var.s3_region
  access_key = var.s3_access_key
  secret_key = var.s3_secret_key
}

resource "coolify_database_backup" "daily" {
  count = var.enable_backups ? 1 : 0

  database_uuid = coolify_postgresql_database.content.uuid
  s3_storage_id = coolify_s3_storage.backups[0].uuid
  frequency     = "@daily"
  enabled       = true
}
