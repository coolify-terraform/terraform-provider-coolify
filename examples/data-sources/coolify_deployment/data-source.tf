# Look up a single deployment by UUID
data "coolify_deployment" "example" {
  uuid = "your-deployment-uuid"
}

output "deployment_status" {
  value = data.coolify_deployment.example.status
}

output "deployment_server" {
  value = data.coolify_deployment.example.server_uuid
}
