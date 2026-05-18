# 01 - First Project: create a project and read it back
#
# The simplest possible configuration. Creates a project, then uses
# a data source to verify it exists.
#
# Prerequisites:
#   - Terraform >= 1.6
#   - Coolify v4.x with an API token
#   - export COOLIFY_ENDPOINT="https://coolify.example.com"
#   - export COOLIFY_TOKEN="your-api-token"
#
# Run:
#   terraform init
#   terraform plan
#   terraform apply
#   terraform output project_uuid

terraform {
  required_providers {
    coolify = {
      source = "SebTardifLabs/coolify"
    }
  }
}

provider "coolify" {}

resource "coolify_project" "demo" {
  name        = "my-first-project"
  description = "Created with Terraform"
}

data "coolify_projects" "all" {
  depends_on = [coolify_project.demo]
}

output "project_uuid" {
  value = coolify_project.demo.uuid
}

output "total_projects" {
  value = length(data.coolify_projects.all.projects)
}
