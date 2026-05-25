# ACME Corp GitHub App CI/CD Pipeline
#
# Demonstrates a complete CI/CD pipeline using GitHub App integration:
# 1. Register a GitHub App with Coolify
# 2. Deploy an application from a GitHub repository
# 3. Configure environment variables for the application
# 4. Trigger an explicit deployment with wait-for-completion
#
# This scenario answers: "How do I set up automated GitHub deployments
# with Coolify?"

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
  name        = "acme-github-cicd"
  description = "ACME Corp GitHub App CI/CD pipeline"
}

# --- SSH Key for GitHub App ---

resource "coolify_private_key" "github_app" {
  name        = "acme-github-app-key"
  description = "Private key for the ACME GitHub App integration"
  private_key = var.github_app_private_key
}

# --- GitHub App Registration ---

resource "coolify_github_app" "acme" {
  name             = "acme-cicd-app"
  app_id           = var.github_app_id
  installation_id  = var.github_app_installation_id
  client_id        = var.github_app_client_id
  client_secret    = var.github_app_client_secret
  webhook_secret   = var.github_app_webhook_secret
  private_key_uuid = coolify_private_key.github_app.uuid
}

# --- Application from GitHub ---

resource "coolify_application_github_app" "api" {
  name             = "acme-api"
  project_uuid     = coolify_project.acme.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  github_app_uuid  = coolify_github_app.acme.uuid
  git_repository   = var.git_repository
  git_branch       = var.git_branch
  build_pack       = "nixpacks"
  ports_exposes    = "3000"
  instant_deploy   = false
}

# --- Environment Variables ---

resource "coolify_environment_variable" "db_url" {
  application_uuid = coolify_application_github_app.api.uuid
  key              = "DATABASE_URL"
  value            = var.database_url
  is_build         = false
  is_preview       = false
}

resource "coolify_environment_variable" "app_env" {
  application_uuid = coolify_application_github_app.api.uuid
  key              = "APP_ENV"
  value            = "production"
  is_build         = false
  is_preview       = false
}

resource "coolify_environment_variable" "log_level" {
  application_uuid = coolify_application_github_app.api.uuid
  key              = "LOG_LEVEL"
  value            = "info"
  is_build         = false
  is_preview       = false
}

# --- Deploy with wait ---

resource "coolify_deployment" "api" {
  application_uuid    = coolify_application_github_app.api.uuid
  wait_for_completion = true

  triggers = {
    env_db_url  = coolify_environment_variable.db_url.value
    env_app_env = coolify_environment_variable.app_env.value
    env_log_lvl = coolify_environment_variable.log_level.value
  }

  timeouts = {
    create = "15m"
  }
}