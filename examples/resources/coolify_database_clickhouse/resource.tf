resource "coolify_database_clickhouse" "example" {
  name         = "my-clickhouse"
  project_uuid = coolify_project.example.uuid
  server_uuid  = coolify_server.example.uuid
  image        = "clickhouse/clickhouse-server:24"
}
