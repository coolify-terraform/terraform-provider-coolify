# ACME Corp Preview Deployments
#
# Demonstrates PR-based preview environments:
# 1. Deploy an application using a Dockerfile
# 2. Create preview environments for specific pull requests
# 3. Clean up previews automatically on terraform destroy
#
# Preview environments let you spin up isolated copies of your app
# for each PR. Coolify tracks them by pull_request_id so destroy
# removes only those previews.
#
# This scenario answers: "How do I set up preview deployments for PRs?"

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
# Each preview tracks a specific pull request. When the PR is merged
# or closed, run terraform destroy to clean up the preview environment.

resource "coolify_application_preview" "pr_1" {
  application_uuid = coolify_application_dockerfile.web.uuid
  pull_request_id  = 1
}

resource "coolify_application_preview" "pr_2" {
  application_uuid = coolify_application_dockerfile.web.uuid
  pull_request_id  = 2
}
