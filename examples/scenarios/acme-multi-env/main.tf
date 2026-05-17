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

module "dev" {
  source              = "./modules/stack"
  project_name        = "acme-dev"
  project_description = var.project_description
  server_uuid         = var.server_uuid
  db_name             = "acme_dev"
  git_repo            = "https://github.com/coollabsio/coolify-examples"
  git_branch          = "main"
}

module "staging" {
  source              = "./modules/stack"
  project_name        = "acme-staging"
  project_description = var.project_description
  server_uuid         = var.server_uuid
  db_name             = "acme_staging"
  git_repo            = "https://github.com/coollabsio/coolify-examples"
  git_branch          = "main"
}
