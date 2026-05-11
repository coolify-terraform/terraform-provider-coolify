# 05 - Data Sources: read existing Coolify resources
#
# Demonstrates how to use data sources to query the current state
# of your Coolify instance. Useful for referencing existing resources
# or building dashboards.
#
# Prerequisites:
#   - At least one project and server in Coolify
#
# Run:
#   terraform init
#   terraform plan
#   terraform apply

terraform {
  required_providers {
    coolify = {
      source = "SebTardifLabs/coolify"
    }
  }
}

provider "coolify" {}

# Read the Coolify instance version
data "coolify_version" "current" {}

# List all projects
data "coolify_projects" "all" {}

# List all servers
data "coolify_servers" "all" {}

# Read the current team
data "coolify_team" "current" {
  id = 0
}

output "coolify_version" {
  value = data.coolify_version.current.version
}

output "project_count" {
  value = length(data.coolify_projects.all.projects)
}

output "server_count" {
  value = length(data.coolify_servers.all.servers)
}

output "team_name" {
  value = data.coolify_team.current.name
}
