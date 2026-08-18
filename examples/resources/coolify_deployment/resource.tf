resource "coolify_deployment" "web" {
  application_uuid = coolify_application.web.uuid

  # Safe on first apply when the application sets instant_deploy = true;
  # a deploy already queued for the same commit is adopted.

  triggers = {
    deploy_version = "v1.2.3"
  }
}

# Wait for deployment to complete (useful in CI pipelines)
resource "coolify_deployment" "web_wait" {
  application_uuid    = coolify_application.web.uuid
  wait_for_completion = true

  triggers = {
    deploy_version = "v1.2.3"
  }

  timeouts = {
    create = "30m"
  }
}
