# ACME Corp Private Repository Deployment
#
# Deploys an application from a private Git repository:
# - SSH key for Git authentication
# - Private Git application with environment variables
# - Deployment with wait-for-completion
#
# This scenario demonstrates the full dependency chain:
# coolify_private_key -> coolify_application_private_git ->
# coolify_environment_variable -> coolify_deployment

terraform {
  required_providers {
    coolify = {
      source = "SebTardifLabs/coolify"
    }
  }
}

provider "coolify" {
  endpoint = var.coolify_endpoint
  token    = var.coolify_token
}

# --- Project ---

resource "coolify_project" "acme" {
  name        = "acme-private"
  description = "ACME Corp private repo deployment"
}

# --- SSH Key ---

resource "coolify_private_key" "deploy" {
  name        = "acme-deploy-key"
  description = "SSH deploy key for the ACME private repo"
  private_key = var.deploy_ssh_key
}

# --- Application from private Git repo ---

resource "coolify_application_private_git" "backend" {
  name                = "acme-backend"
  project_uuid        = coolify_project.acme.uuid
  server_uuid         = var.server_uuid
  environment_name    = "production"
  private_key_uuid    = coolify_private_key.deploy.uuid
  git_repository      = var.git_repository
  git_branch          = var.git_branch
  build_pack          = "dockerfile"
  ports_exposes       = "3000"
  dockerfile_location = "/Dockerfile"
  instant_deploy      = false
}

# --- Environment Variables ---

resource "coolify_environment_variable" "database_url" {
  application_uuid = coolify_application_private_git.backend.uuid
  key              = "DATABASE_URL"
  value            = var.database_url
  is_build         = false
  is_preview       = false
}

resource "coolify_environment_variable" "app_secret" {
  application_uuid = coolify_application_private_git.backend.uuid
  key              = "APP_SECRET"
  value            = var.app_secret
  is_build         = false
  is_preview       = false
}

# --- Deploy with wait ---

resource "coolify_deployment" "backend" {
  application_uuid    = coolify_application_private_git.backend.uuid
  wait_for_completion = true

  triggers = {
    env_database_url = coolify_environment_variable.database_url.value
    env_app_secret   = coolify_environment_variable.app_secret.value
  }

  timeouts = {
    create = "15m"
  }
}
