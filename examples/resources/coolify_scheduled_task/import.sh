# Import a scheduled task on an application:
terraform import coolify_scheduled_task.example application:<application-uuid>:<task-uuid>

# Import a scheduled task on a service:
terraform import coolify_scheduled_task.example service:<service-uuid>:<task-uuid>
