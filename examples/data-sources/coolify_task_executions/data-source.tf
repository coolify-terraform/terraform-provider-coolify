# List task executions for an application
data "coolify_task_executions" "app_task" {
  application_uuid = "your-app-uuid"
  task_uuid        = "your-task-uuid"
}

output "app_task_statuses" {
  value = [for e in data.coolify_task_executions.app_task.executions : e.status]
}

# List task executions for a service
data "coolify_task_executions" "svc_task" {
  service_uuid = "your-service-uuid"
  task_uuid    = "your-task-uuid"
}

output "svc_task_statuses" {
  value = [for e in data.coolify_task_executions.svc_task.executions : e.status]
}
