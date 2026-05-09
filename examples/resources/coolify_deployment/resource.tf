resource "coolify_deployment" "web" {
  application_uuid = coolify_application.web.uuid

  triggers = {
    deploy_version = "v1.2.3"
  }
}