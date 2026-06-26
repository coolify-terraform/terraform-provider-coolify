# ACME Corp Import Existing Resources
#
# Demonstrates importing pre-existing Coolify resources into Terraform.
# This is the most common entry point for teams adopting Terraform on
# an existing Coolify instance.
#
# The scenario covers:
# - Importing a project by UUID
# - Importing an application using the compound format
#   (project_uuid:server_uuid:environment_name:app_uuid)
# - Importing an environment variable with its composite key
#   (type:parent_uuid:env_var_uuid)
# - Data source read-back to discover UUIDs before importing

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

# --- Step 1: Discover existing resources ---
#
# Before importing, use data sources to find UUIDs. In a real workflow
# you would also use the Coolify API directly:
#   curl -s -H "Authorization: Bearer $TOKEN" "$ENDPOINT/api/v1/projects" | jq '.[].uuid'

data "coolify_server" "target" {
  uuid = var.server_uuid
}

# --- Step 2: Define the resource blocks ---
#
# Write the Terraform configuration FIRST, matching your existing
# Coolify setup. These blocks describe what is already running.

resource "coolify_project" "existing" {
  name        = "acme-import-demo"
  description = var.project_description
}

resource "coolify_application_docker_image" "web" {
  name             = "acme-imported-web"
  project_uuid     = coolify_project.existing.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  docker_image     = "nginx:latest"
  ports_exposes    = "80"
}

resource "coolify_environment_variable" "app_env" {
  application_uuid = coolify_application_docker_image.web.uuid
  key              = "APP_ENV"
  value            = "production"
  is_build         = false
  is_preview       = false
}

# --- Step 3: Import commands ---
#
# After writing the blocks above, run these commands to bring the
# existing resources under Terraform management:
#
#   # Simple UUID import (projects, servers, private keys):
#   terraform import coolify_project.existing <project-uuid>
#
#   # Compound import for applications (avoids post-import diffs):
#   terraform import coolify_application_docker_image.web \
#     <project-uuid>:<server-uuid>:production:<app-uuid>
#
#   # Composite key import for environment variables:
#   terraform import coolify_environment_variable.app_env \
#     application:<app-uuid>:<env-var-uuid>
#
# Alternatively, use Terraform 1.5+ import blocks (see import.tf).

# --- Data source read-backs ---

data "coolify_project" "verify" {
  uuid = coolify_project.existing.uuid
}

data "coolify_application" "verify" {
  uuid = coolify_application_docker_image.web.uuid
}
