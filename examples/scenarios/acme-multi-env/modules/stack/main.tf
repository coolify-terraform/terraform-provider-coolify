# Reusable Coolify stack: project + database + application + env var

resource "coolify_project" "this" {
  name        = var.project_name
  description = "Managed by Terraform"
}

resource "coolify_postgresql_database" "app" {
  name         = "${var.project_name}-db"
  project_uuid = coolify_project.this.uuid
  server_uuid  = var.server_uuid
  image        = var.db_image
  postgres_db  = var.db_name
}

resource "coolify_application" "app" {
  name           = "${var.project_name}-app"
  project_uuid   = coolify_project.this.uuid
  server_uuid    = var.server_uuid
  git_repository = var.git_repo
  git_branch     = var.git_branch
  build_pack     = "nixpacks"
  ports_exposes  = "3000"
}

# Wire the database to the application via DATABASE_URL.
# In Coolify the resource 'name' is also the Docker container name,
# which serves as the internal hostname for service-to-service communication.
resource "coolify_environment_variable" "database_url" {
  application_uuid = coolify_application.app.uuid
  key              = "DATABASE_URL"
  value            = "postgresql://postgres@${coolify_postgresql_database.app.name}:5432/${var.db_name}"
  is_build         = false
  is_preview       = false
}
