# ACME Corp External Integrations
#
# Tests external integration resources via Coolify managed services
# from the built-in service catalog.

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
  description = var.project_description
}

# --- Managed Service (from Coolify service catalog) ---

resource "coolify_service" "uptime" {
  name             = "acme-uptime-monitor"
  project_uuid     = coolify_project.integrations.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  type             = "uptime-kuma"
}