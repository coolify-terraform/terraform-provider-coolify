# Acceptance test for the ACME Corp Marketing Website scenario.
#
# Required variables are provided via TF_VAR_* environment variables:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

variables {
  enable_backups = false
}

run "create_and_verify" {
  command = apply

  # --- Project ---
  assert {
    condition     = coolify_project.acme.uuid != ""
    error_message = "Project was not created: uuid is empty"
  }
  assert {
    condition     = coolify_project.acme.name == "acme-website"
    error_message = "Project name mismatch: got ${coolify_project.acme.name}"
  }

  # --- Database ---
  assert {
    condition     = coolify_postgresql_database.content.uuid != ""
    error_message = "PostgreSQL database was not created: uuid is empty"
  }
  assert {
    condition     = coolify_postgresql_database.content.name == "acme-content"
    error_message = "Database name mismatch: got ${coolify_postgresql_database.content.name}"
  }
  assert {
    condition     = coolify_postgresql_database.content.postgres_user == "acme"
    error_message = "Database user mismatch: got ${coolify_postgresql_database.content.postgres_user}"
  }
  assert {
    condition     = coolify_postgresql_database.content.postgres_db == "acme_content"
    error_message = "Database name mismatch: got ${coolify_postgresql_database.content.postgres_db}"
  }

  # --- Application ---
  assert {
    condition     = coolify_application.website.uuid != ""
    error_message = "Application was not created: uuid is empty"
  }
  assert {
    condition     = coolify_application.website.name == "acme-website"
    error_message = "Application name mismatch: got ${coolify_application.website.name}"
  }
  assert {
    condition     = coolify_application.website.build_pack == "nixpacks"
    error_message = "Build pack not preserved: got ${coolify_application.website.build_pack}"
  }
  # Quirk #1: Coolify strips https://github.com/ prefix. Provider must preserve original.
  assert {
    condition     = coolify_application.website.git_repository == "https://github.com/coollabsio/coolify-examples"
    error_message = "Git repository URL was normalized: got ${coolify_application.website.git_repository}"
  }
  assert {
    condition     = coolify_application.website.ports_exposes == "3000"
    error_message = "Ports exposes mismatch: got ${coolify_application.website.ports_exposes}"
  }

  # --- Environment Variables ---
  assert {
    condition     = coolify_environment_variable.database_url.key == "DATABASE_URL"
    error_message = "Env var key mismatch: got ${coolify_environment_variable.database_url.key}"
  }
  assert {
    condition     = coolify_environment_variable.node_env.key == "NODE_ENV"
    error_message = "Env var key mismatch: got ${coolify_environment_variable.node_env.key}"
  }
  assert {
    condition     = nonsensitive(coolify_environment_variable.node_env.value) == "production"
    error_message = "Env var value mismatch"
  }
}

run "verify_data_sources" {
  command = apply

  variables {
    enable_backups = false
  }

  assert {
    condition     = data.coolify_project.verify.name == coolify_project.acme.name
    error_message = "Project data source name does not match resource"
  }
  assert {
    condition     = data.coolify_application.verify.name == coolify_application.website.name
    error_message = "Application data source name does not match resource"
  }
}

# Update: change description and verify the update is applied.
run "update_project_description" {
  command = apply

  variables {
    project_description = "Updated by scenario test"
  }

  assert {
    condition     = coolify_project.acme.description == "Updated by scenario test"
    error_message = "Project description not updated: got ${coolify_project.acme.description}"
  }
}

# Idempotency: re-plan should produce no changes.
# Catches normalization bugs where flatten returns different values than config.
# If the plan shows changes, terraform test logs them but doesn't fail on its own.
# The assertion on a known-stable attribute detects if state was corrupted.
run "idempotency" {
  command = plan

  assert {
    condition     = coolify_project.acme.name == "acme-website"
    error_message = "Project name changed after re-plan (state corruption)"
  }
}
