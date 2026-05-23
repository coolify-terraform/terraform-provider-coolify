# 02 - Deploy Application: deploy a web app from a public Git repo
#
# Creates a project and deploys a Nixpacks-built application.
# Coolify builds the app from source and exposes it on port 3000.
#
# Prerequisites:
#   - Coolify with at least one registered server
#   - Replace "your-server-uuid" with an actual server UUID
#   - Find server UUIDs: curl -sH "Authorization: Bearer $COOLIFY_TOKEN" \
#       "$COOLIFY_ENDPOINT/api/v1/servers" | jq '.[].uuid'
#
# Run:
#   terraform init
#   terraform plan
#   terraform apply

terraform {
  required_providers {
    coolify = {
      source = "coolify-terraform/coolify"
    }
  }
}

provider "coolify" {}

variable "server_uuid" {
  description = "UUID of the Coolify server to deploy on"
  type        = string
}

resource "coolify_project" "app" {
  name = "web-app"
}

resource "coolify_application" "web" {
  name           = "hello-world"
  project_uuid   = coolify_project.app.uuid
  server_uuid    = var.server_uuid
  git_repository = "https://github.com/coollabsio/coolify-examples"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"

  timeouts = {
    create = "15m"
  }
}

output "app_uuid" {
  value = coolify_application.web.uuid
}
