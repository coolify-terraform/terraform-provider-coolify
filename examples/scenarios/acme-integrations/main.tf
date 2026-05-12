# ACME Corp External Integrations
#
# Tests external integration resources: GitHub App registration and
# Coolify managed services (from the service catalog).

terraform {
  required_providers {
    coolify = {
      source = "SebTardifLabs/coolify"
    }
    tls = {
      source = "hashicorp/tls"
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

# --- GitHub App (dummy credentials, registers metadata only) ---
# GitHub App CRUD works with dummy values because Coolify stores the
# metadata without immediately verifying OAuth connectivity.

resource "tls_private_key" "github" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "coolify_github_app" "acme" {
  name            = "acme-github-integration"
  organization    = "acme-corp"
  app_id          = 12345
  installation_id = 67890
  client_id       = "dummy-client-id-for-testing"
  client_secret   = "dummy-client-secret-for-testing"
  private_key     = tls_private_key.github.private_key_pem
}

# --- Managed Service (from Coolify service catalog) ---

resource "coolify_service" "uptime" {
  name             = "acme-uptime-monitor"
  project_uuid     = coolify_project.integrations.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  type             = "uptime-kuma"
}