# ACME Corp Hetzner Cloud Infrastructure
#
# Demonstrates provisioning Hetzner Cloud servers through Coolify:
# 1. Register SSH keys for server access
# 2. Register a Hetzner API token with Coolify
# 3. Provision a production application server
# 4. Provision a dedicated build server
#
# This scenario answers: "How do I provision cloud servers for Coolify
# using Hetzner Cloud?"

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

# --- SSH Key ---

resource "coolify_private_key" "deploy" {
  name        = "acme-hetzner-deploy-key"
  description = "SSH key for ACME Hetzner Cloud server access"
  private_key = var.deploy_ssh_key
}

# --- Cloud Token ---

resource "coolify_cloud_token" "hetzner" {
  name           = "acme-hetzner-token"
  cloud_provider = "hetzner"
  token          = var.hetzner_api_token
}

# --- Production Server ---

resource "coolify_server_hetzner" "production" {
  name                      = "acme-production"
  cloud_provider_token_uuid = coolify_cloud_token.hetzner.uuid
  server_type               = "cx22"
  location                  = "fsn1"
  image                     = "ubuntu-24.04"
  private_key_uuid          = coolify_private_key.deploy.uuid
}

# --- Build Server ---

resource "coolify_server_hetzner" "build" {
  name                      = "acme-build"
  cloud_provider_token_uuid = coolify_cloud_token.hetzner.uuid
  server_type               = "cx32"
  location                  = "nbg1"
  image                     = "ubuntu-24.04"
  private_key_uuid          = coolify_private_key.deploy.uuid
  is_build_server           = true
}
