# ACME Corp Platform Setup
#
# Tests infrastructure-level resources: private keys, custom environments,
# persistent storage, deployments, cloud tokens, and data source read-backs.

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

# --- Infrastructure data sources ---

data "coolify_version" "this" {}

data "coolify_health" "this" {}

data "coolify_server" "target" {
  uuid = var.server_uuid
}

data "coolify_teams" "all" {}

# --- Private Key ---

resource "coolify_private_key" "deploy" {
  name        = "acme-deploy-key"
  description = "Deployment key for ACME platform"
  private_key = var.deploy_key
}

# --- Project with custom environment ---

resource "coolify_project" "platform" {
  name        = "acme-platform"
  description = var.project_description
}

resource "coolify_environment" "staging" {
  project_uuid = coolify_project.platform.uuid
  name         = "staging"
  description  = "Staging environment for pre-production testing"
}

# --- Application with storage ---

resource "coolify_application" "api" {
  name           = "acme-platform-api"
  project_uuid   = coolify_project.platform.uuid
  server_uuid    = var.server_uuid
  git_repository = "https://github.com/coollabsio/coolify-examples"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"
}

resource "coolify_storage" "uploads" {
  application_uuid = coolify_application.api.uuid
  name             = "app-uploads"
  mount_path       = "/app/uploads"
}