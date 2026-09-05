# ACME Corp Preview Deployments
#
# Tracks PR-based preview environments that Coolify already created
# (webhook or UI). This resource does not create those previews.
# 1. Deploy an application using a Dockerfile
# 2. Track preview environments for specific pull requests
# 3. terraform destroy DELETEs each tracked preview if it exists
#
# Domain PATCH examples live in
# examples/resources/coolify_application_preview/resource.tf
#
# This scenario answers: "How do I track Coolify PR previews in Terraform?"

terraform {
  required_providers {
    coolify = {
      source = "coolify-terraform/coolify"
    }
  }
}

provider "coolify" {
  endpoint = var.coolify_endpoint
  token    = var.coolify_token
}

# --- Project ---

resource "coolify_project" "acme" {
  name        = "acme-preview-deploy"
  description = var.project_description
}

# --- Application (Dockerfile-based) ---
#
# Any application type supports previews. We use a simple Dockerfile
# app here to keep the scenario self-contained (no external Git
# credentials needed).

resource "coolify_application_dockerfile" "web" {
  name                = "acme-web"
  project_uuid        = coolify_project.acme.uuid
  server_uuid         = var.server_uuid
  environment_name    = "production"
  dockerfile_location = base64encode("FROM nginx:alpine\nEXPOSE 80\n")
  ports_exposes       = "80"
  instant_deploy      = false
}

# --- PR Preview Environments ---
#
# Each resource tracks a preview Coolify already created (webhook/UI).
# terraform destroy DELETEs the tracked preview if it exists.

resource "coolify_application_preview" "pr_1" {
  application_uuid = coolify_application_dockerfile.web.uuid
  pull_request_id  = 1
}

resource "coolify_application_preview" "pr_2" {
  application_uuid = coolify_application_dockerfile.web.uuid
  pull_request_id  = 2
}
