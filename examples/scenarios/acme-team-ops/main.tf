# ACME Corp Team Operations Dashboard
#
# Shows data-source-heavy patterns for operational visibility:
# 1. Read team info and members
# 2. Discover available servers
# 3. List all projects and their environments
# 4. Check Coolify version and health
#
# This scenario answers: "How do I audit my Coolify infrastructure state
# using Terraform data sources?"

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

# --- Infrastructure Discovery ---

data "coolify_version" "current" {}

data "coolify_team" "root" {
  id = 0 # root team (always exists on every Coolify instance)
}

data "coolify_team_members" "root" {
  id = 0 # root team
}

data "coolify_servers" "all" {}

# --- Project Inventory ---
#
# Create a project so there is at least one to list, then use data
# sources to discover everything on the instance.

resource "coolify_project" "ops" {
  name        = "acme-team-ops"
  description = var.project_description
}

data "coolify_projects" "all" {
  depends_on = [coolify_project.ops]
}

data "coolify_environments" "ops" {
  project_uuid = coolify_project.ops.uuid
}

# --- Server Details ---
#
# Read details about the target server to demonstrate single-resource
# data source usage alongside list data sources.

data "coolify_server" "target" {
  uuid = var.server_uuid
}