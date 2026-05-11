# Full-stack example: deploy a web app with database on Coolify
#
# This example provisions:
# 1. A project to group resources
# 2. An SSH key for server access
# 3. A server registration
# 4. A PostgreSQL database
# 5. A web application from a Git repository
# 6. Environment variables connecting the app to the database

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

variable "server_ip" {
  description = "IP address of the server to deploy to"
  type        = string
}

variable "ssh_private_key" {
  description = "Path to SSH private key for server access"
  type        = string
  default     = "~/.ssh/id_ed25519"
}

variable "app_domain" {
  description = "Domain name for the application"
  type        = string
  default     = ""
}

# --- Project ---

resource "coolify_project" "main" {
  name        = "my-fullstack-app"
  description = "Full-stack application managed by Terraform"
}

# --- Server ---

resource "coolify_private_key" "deploy" {
  name        = "deploy-key"
  description = "SSH key for deployment server"
  private_key = file(var.ssh_private_key)
}

resource "coolify_server" "web" {
  name             = "web-server"
  description      = "Primary web server"
  ip               = var.server_ip
  port             = 22
  user             = "root"
  private_key_uuid = coolify_private_key.deploy.uuid
}

# --- Database ---

resource "coolify_postgresql_database" "app_db" {
  name              = "app-database"
  project_uuid      = coolify_project.main.uuid
  server_uuid       = coolify_server.web.uuid
  environment_name  = "production"
  image             = "postgres:16"
  postgres_user     = "app"
  postgres_password = "change-me-in-production"
  postgres_db       = "myapp"
  is_public         = false
}

# --- Application ---

resource "coolify_application" "web" {
  name             = "web-app"
  project_uuid     = coolify_project.main.uuid
  server_uuid      = coolify_server.web.uuid
  environment_name = "production"
  git_repository   = "https://github.com/example/my-web-app"
  git_branch       = "main"
  build_pack       = "nixpacks"
  ports_exposes    = "3000"
  fqdn             = var.app_domain != "" ? "https://${var.app_domain}" : null
}

# --- Environment Variables ---
# In Coolify, the resource 'name' is also the Docker container name,
# which serves as the internal hostname for service-to-service communication.

resource "coolify_environment_variable" "database_url" {
  application_uuid = coolify_application.web.uuid
  key              = "DATABASE_URL"
  value            = "postgresql://${coolify_postgresql_database.app_db.postgres_user}:${coolify_postgresql_database.app_db.postgres_password}@${coolify_postgresql_database.app_db.name}:5432/${coolify_postgresql_database.app_db.postgres_db}"
  is_build         = false
  is_preview       = false
}

resource "coolify_environment_variable" "node_env" {
  application_uuid = coolify_application.web.uuid
  key              = "NODE_ENV"
  value            = "production"
  is_build         = true
  is_preview       = false
}

# --- Outputs ---

output "project_uuid" {
  value = coolify_project.main.uuid
}

output "server_uuid" {
  value = coolify_server.web.uuid
}

output "database_uuid" {
  value = coolify_postgresql_database.app_db.uuid
}

output "application_uuid" {
  value = coolify_application.web.uuid
}