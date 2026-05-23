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

variable "coolify_endpoint" {
  type = string
}

variable "coolify_token" {
  type      = string
  sensitive = true
}

variable "server_uuid" {
  type = string
}

variable "environment" {
  type = string
}

variable "git_branch" {
  type = string
}

variable "domain" {
  type = string
}

variable "db_password" {
  type      = string
  sensitive = true
}

module "app" {
  source       = "./modules/app"
  project_name = "myapp"
  server_uuid  = var.server_uuid
  environment  = var.environment
  git_branch   = var.git_branch
  domain       = var.domain
  db_password  = var.db_password
}
