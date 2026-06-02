variable "redis_password" {
  type      = string
  sensitive = true
}

resource "coolify_database_redis" "example" {
  name           = "my-redis"
  project_uuid   = coolify_project.example.uuid
  server_uuid    = coolify_server.example.uuid
  image          = "redis:7"
  redis_password = var.redis_password
}
