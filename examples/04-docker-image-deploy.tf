# 04 - Docker Image Deploy: deploy a pre-built container image
#
# Deploys an application from a Docker image instead of building
# from source. Useful for images from Docker Hub, GHCR, or private
# registries.
#
# Prerequisites:
#   - Coolify with a registered server
#
# Run:
#   terraform init
#   terraform plan -var="server_uuid=your-uuid"
#   terraform apply -var="server_uuid=your-uuid"

terraform {
  required_providers {
    coolify = {
      source = "SebTardifLabs/coolify"
    }
  }
}

provider "coolify" {}

variable "server_uuid" {
  type = string
}

resource "coolify_project" "containers" {
  name = "container-apps"
}

resource "coolify_application_docker_image" "nginx" {
  name          = "nginx-proxy"
  project_uuid  = coolify_project.containers.uuid
  server_uuid   = var.server_uuid
  docker_image  = "nginx:alpine"
  ports_exposes = "80"
  fqdn          = "https://proxy.example.com"

  timeouts = {
    create = "10m"
  }
}

output "nginx_uuid" {
  value = coolify_application_docker_image.nginx.uuid
}
