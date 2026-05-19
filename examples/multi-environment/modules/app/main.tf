variable "project_name" {
  type = string
}

variable "server_uuid" {
  type = string
}

variable "environment" {
  type = string
}

variable "git_branch" {
  type = string
}

variable "domain" {
  type = string
}

variable "db_password" {
  type      = string
  sensitive = true
}

resource "coolify_project" "this" {
  name        = "${var.project_name}-${var.environment}"
  description = "${var.environment} environment"
}

resource "coolify_environment" "this" {
  count        = var.environment != "production" ? 1 : 0
  project_uuid = coolify_project.this.uuid
  name         = var.environment
}

resource "coolify_database_postgresql" "db" {
  depends_on        = [coolify_environment.this]
  name              = "${var.project_name}-db"
  project_uuid      = coolify_project.this.uuid
  server_uuid       = var.server_uuid
  environment_name  = var.environment
  postgres_password = var.db_password
  postgres_db       = var.project_name
}

resource "coolify_application" "app" {
  depends_on       = [coolify_environment.this]
  name             = "${var.project_name}-app"
  project_uuid     = coolify_project.this.uuid
  server_uuid      = var.server_uuid
  environment_name = var.environment
  git_repository   = "https://github.com/example/app"
  git_branch       = var.git_branch
  build_pack       = "nixpacks"
  ports_exposes    = "3000"
  fqdn             = "https://${var.domain}"
}

# The database 'name' is the Docker container name, used as internal hostname.
resource "coolify_environment_variable" "db_url" {
  application_uuid = coolify_application.app.uuid
  key              = "DATABASE_URL"
  value            = "postgresql://postgres:${var.db_password}@${coolify_database_postgresql.db.name}:5432/${var.project_name}"
}
