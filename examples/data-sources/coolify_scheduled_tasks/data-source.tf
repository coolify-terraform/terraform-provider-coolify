# List scheduled tasks for an application
data "coolify_scheduled_tasks" "app_tasks" {
  application_uuid = "your-app-uuid"
}

output "task_names" {
  value = [for t in data.coolify_scheduled_tasks.app_tasks.tasks : t.name]
}

# List scheduled tasks for a service
data "coolify_scheduled_tasks" "svc_tasks" {
  service_uuid = "your-service-uuid"
}

output "svc_task_names" {
  value = [for t in data.coolify_scheduled_tasks.svc_tasks.tasks : t.name]
}
