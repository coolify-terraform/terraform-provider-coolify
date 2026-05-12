output "nginx_uuid" {
  value = coolify_docker_image_application.nginx.uuid
}

output "redis_uuid" {
  value = coolify_docker_image_application.redis.uuid
}

output "task_uuid" {
  value = coolify_scheduled_task.cleanup.uuid
}

output "storage_uuid" {
  value = coolify_storage.static.uuid
}