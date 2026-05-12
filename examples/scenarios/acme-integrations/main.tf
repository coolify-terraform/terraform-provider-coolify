# ACME Corp External Integrations
#
# Tests external integration resources: GitHub App registration and
# Coolify managed services (from the service catalog).

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

resource "coolify_project" "integrations" {
  name        = "acme-integrations"
  description = "ACME Corp external service integrations"
}

# --- GitHub App ---
# Temporarily removed: Coolify validates GitHub App credentials on creation
# (returns 422 with dummy values). Unlike what acceptance tests suggest,
# real Coolify requires valid GitHub App OAuth credentials.
# See issue #44 for tracking.

# --- Managed Service (from Coolify service catalog) ---

resource "coolify_service" "uptime" {
  name             = "acme-uptime-monitor"
  project_uuid     = coolify_project.integrations.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  type             = "uptime-kuma"
}