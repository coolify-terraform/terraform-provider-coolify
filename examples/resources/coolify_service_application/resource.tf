variable "server_uuid" {
  type = string
}

resource "coolify_project" "example" {
  name = "example"
}

resource "coolify_service" "example" {
  name         = "uptime-kuma"
  type         = "uptime-kuma"
  project_uuid = coolify_project.example.uuid
  server_uuid  = var.server_uuid
}

data "coolify_service_applications" "example" {
  service_uuid = coolify_service.example.uuid
}

resource "coolify_service_application" "web" {
  service_uuid = coolify_service.example.uuid
  uuid         = data.coolify_service_applications.example.applications[0].uuid
  description  = "Public status page"

  # Requires Coolify >= v4.5.0. Older instances keep these in state and do not send them.
  # is_permanent_redirect_enabled = true
  # traefik_certificate_resolver  = "letsencrypt"
}
