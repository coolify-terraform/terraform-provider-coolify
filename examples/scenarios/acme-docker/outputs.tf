output "nginx_uuid" {
  value = coolify_application_docker_image.nginx.uuid
}

output "redis_uuid" {
  value = coolify_application_docker_image.redis.uuid
}

output "task_uuid" {
  value = coolify_scheduled_task.cleanup.uuid
}

output "storage_uuid" {
  value = coolify_storage.static.uuid
}