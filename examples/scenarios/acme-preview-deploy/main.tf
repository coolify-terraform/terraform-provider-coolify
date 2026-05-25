# ACME Corp Preview Deployments
#
# Demonstrates PR-based preview environments:
# 1. Register a GitHub App integration with Coolify
# 2. Deploy an application from a GitHub repository
# 3. Create preview environments for specific pull requests
# 4. Clean up previews automatically on terraform destroy
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
  description = "ACME Corp PR preview deployments"
}

# --- SSH Key for GitHub App ---

resource "coolify_private_key" "github_app" {
  name        = "acme-github-app-key"
  description = "Private key for the ACME GitHub App integration"
  private_key = var.github_app_private_key
}

# --- GitHub App Integration ---

resource "coolify_github_app" "acme" {
  name             = "acme-preview-app"
  app_id           = var.github_app_id
  installation_id  = var.github_app_installation_id
  client_id        = var.github_app_client_id
  client_secret    = var.github_app_client_secret
  webhook_secret   = var.github_app_webhook_secret
  private_key_uuid = coolify_private_key.github_app.uuid
}

# --- Application via GitHub App ---

resource "coolify_application_github_app" "web" {
  name            = "acme-web"
  project_uuid    = coolify_project.acme.uuid
  server_uuid     = var.server_uuid
  github_app_uuid = coolify_github_app.acme.uuid
  git_repository  = var.git_repository
  git_branch      = "main"
  build_pack      = "nixpacks"
  ports_exposes   = "3000"
  instant_deploy  = false
}

# --- PR Preview Environments ---

resource "coolify_application_preview" "pr_1" {
  application_uuid = coolify_application_github_app.web.uuid
  pull_request_id  = 1
}

resource "coolify_application_preview" "pr_2" {
  application_uuid = coolify_application_github_app.web.uuid
  pull_request_id  = 2
}
