# Acceptance test for ACME Corp team operations scenario.
#
# Tests data-source-heavy patterns: team, team members, servers, projects,
# environments, server details, and version.
#
# Required variables via TF_VAR_*:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

run "create_and_discover" {
  command = apply

  # --- Project ---
  assert {
    condition     = coolify_project.ops.name == "acme-team-ops"
    error_message = "Project name mismatch: got ${coolify_project.ops.name}"
  }

  # --- Version ---
  assert {
    condition     = data.coolify_version.current.version != ""
    error_message = "Coolify version is empty"
  }

  # --- Team ---
  assert {
    condition     = data.coolify_team.root.name != ""
    error_message = "Root team name is empty"
  }

  # --- Team Members ---
  assert {
    condition     = length(data.coolify_team_members.root.members) > 0
    error_message = "Root team has no members"
  }

  # --- Servers ---
  assert {
    condition     = length(data.coolify_servers.all.servers) > 0
    error_message = "No servers visible (at least one expected)"
  }

  # --- Projects ---
  assert {
    condition     = length(data.coolify_projects.all.projects) > 0
    error_message = "No projects found (we just created one)"
  }

  # --- Environments ---
  assert {
    condition     = length(data.coolify_environments.ops.environments) > 0
    error_message = "New project has no environments (Coolify creates a default)"
  }

  # --- Target Server ---
  assert {
    condition     = data.coolify_server.target.name != ""
    error_message = "Target server name is empty"
  }
  assert {
    condition     = data.coolify_server.target.ip != ""
    error_message = "Target server IP is empty"
  }
}

# Update: change project description.
run "update_project_description" {
  command = apply

  variables {
    project_description = "Updated by scenario test"
  }

  assert {
    condition     = coolify_project.ops.description == "Updated by scenario test"
    error_message = "Project description not updated: got ${coolify_project.ops.description}"
  }
}

# Idempotency: re-plan with same variables should produce no changes.
run "idempotency" {
  command = plan

  variables {
    project_description = "Updated by scenario test"
  }

  assert {
    condition     = coolify_project.ops.name == "acme-team-ops"
    error_message = "Project name changed after re-plan"
  }
}