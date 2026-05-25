# ACME Corp Custom Docker Compose Stack
#
# Demonstrates deploying a custom Docker Compose application using
# docker_compose_raw, the approach for user-written compose files:
# 1. Write a multi-container compose stack as HCL heredoc
# 2. Deploy via coolify_service with docker_compose_raw
# 3. Read back service state via data sources
#
# The acme-integrations scenario uses the Coolify catalog (type = "...").
# This scenario shows the custom compose approach, which is how most
# real-world applications are deployed when they have their own
# docker-compose.yml.
#
# This scenario answers: "How do I deploy my own Docker Compose file
# through Coolify?"

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

resource "coolify_project" "compose" {
  name        = "acme-compose-git"
  description = var.project_description
}

# --- Custom Compose Service ---
#
# Uses docker_compose_raw to deploy a custom multi-container stack.
# This is a simple web + cache stack: nginx serving static content
# with a Redis sidecar for caching.
#
# In a real project, this YAML would come from your repository's
# docker-compose.yml. Here we inline it via heredoc to keep the
# scenario self-contained.

resource "coolify_service" "stack" {
  name             = "acme-compose-stack"
  project_uuid     = coolify_project.compose.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"

  docker_compose_raw = <<-YAML
    services:
      web:
        image: 'nginx:alpine'
        ports:
          - '8080:80'
        depends_on:
          - cache
      cache:
        image: 'redis:7-alpine'
        ports:
          - '6379:6379'
  YAML
}

# --- Data Source: read back service state ---

data "coolify_service" "stack" {
  uuid = coolify_service.stack.uuid
}
