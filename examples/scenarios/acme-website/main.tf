# ACME Corp Marketing Website
#
# Deploys a Node.js marketing site backed by PostgreSQL on Coolify.
# Optionally configures daily database backups using an existing UI-managed
# S3-compatible storage.

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

# --- Data Source Read-Backs ---

data "coolify_project" "verify" {
  uuid = coolify_project.acme.uuid
}

data "coolify_application" "verify" {
  uuid = coolify_application.website.uuid
}

# --- Backups (optional) ---
# Set enable_backups = true and provide an existing UI-managed S3 storage UUID.

resource "coolify_database_backup" "daily" {
  count = var.enable_backups ? 1 : 0

  database_uuid   = coolify_postgresql_database.content.uuid
  s3_storage_uuid = var.existing_s3_storage_uuid
  frequency       = "@daily"
  enabled         = true
}
