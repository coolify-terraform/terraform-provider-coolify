# List all deployments
data "coolify_deployments" "all" {}

output "all_deployment_uuids" {
  value = [for d in data.coolify_deployments.all.deployments : d.uuid]
}

# List deployments for a specific application
data "coolify_deployments" "app" {
  application_uuid = "your-app-uuid"
}

output "app_deployment_statuses" {
  value = [for d in data.coolify_deployments.app.deployments : d.status]
}
