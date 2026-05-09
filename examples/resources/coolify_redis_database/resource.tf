resource "coolify_redis_database" "example" {
  name         = "my-redis"
  project_uuid = coolify_project.example.uuid
  server_uuid  = coolify_server.example.uuid
  image        = "redis:7"
}