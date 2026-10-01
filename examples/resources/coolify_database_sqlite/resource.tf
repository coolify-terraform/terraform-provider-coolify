resource "coolify_database_sqlite" "example" {
  project_uuid     = coolify_project.example.uuid
  server_uuid      = var.server_uuid
  name             = "app-sqlite"
  sqlite_databases = "app.sqlite,cache.sqlite"
}
