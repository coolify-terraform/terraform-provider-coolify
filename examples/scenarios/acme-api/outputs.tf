output "project_uuid" {
  description = "UUID of the ACME orders project"
  value       = coolify_project.acme.uuid
}

output "api_app_uuid" {
  description = "UUID of the order processing API application"
  value       = coolify_dockerfile_application.api.uuid
}

output "worker_app_uuid" {
  description = "UUID of the background worker application"
  value       = coolify_docker_image_application.worker.uuid
}

output "orders_db_uuid" {
  description = "UUID of the PostgreSQL orders database"
  value       = coolify_database_postgresql.orders.uuid
}

output "redis_uuid" {
  description = "UUID of the Redis queue database"
  value       = coolify_database_redis.queue.uuid
}
