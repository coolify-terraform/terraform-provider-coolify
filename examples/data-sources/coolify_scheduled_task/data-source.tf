# Look up a single scheduled task by UUID
data "coolify_scheduled_task" "example" {
  uuid             = "your-task-uuid"
  application_uuid = "your-app-uuid"
}

output "task_command" {
  value = data.coolify_scheduled_task.example.command
}

output "task_frequency" {
  value = data.coolify_scheduled_task.example.frequency
}
