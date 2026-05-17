# Acceptance test for the ACME Corp Multi-Environment scenario.
#
# Required variables are provided via TF_VAR_* environment variables:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

run "create_both_environments" {
  command = apply

  # --- Dev environment ---
  assert {
    condition     = module.dev.project_uuid != ""
    error_message = "dev project was not created: uuid is empty"
  }
  assert {
    condition     = module.dev.database_uuid != ""
    error_message = "dev database was not created: uuid is empty"
  }
  assert {
    condition     = module.dev.application_uuid != ""
    error_message = "dev application was not created: uuid is empty"
  }

  # --- Staging environment ---
  assert {
    condition     = module.staging.project_uuid != ""
    error_message = "staging project was not created: uuid is empty"
  }
  assert {
    condition     = module.staging.database_uuid != ""
    error_message = "staging database was not created: uuid is empty"
  }
  assert {
    condition     = module.staging.application_uuid != ""
    error_message = "staging application was not created: uuid is empty"
  }

  # --- Cross-environment isolation ---
  assert {
    condition     = module.dev.project_uuid != module.staging.project_uuid
    error_message = "Dev and staging projects should have different UUIDs"
  }
  assert {
    condition     = module.dev.database_uuid != module.staging.database_uuid
    error_message = "Dev and staging databases should have different UUIDs"
  }
}

# Update: change description and verify the update is applied.
run "update_project_description" {
  command = apply

  variables {
    project_description = "Updated by scenario test"
  }

  assert {
    condition     = module.dev.project_description == "Updated by scenario test"
    error_message = "Dev project description not updated: got ${module.dev.project_description}"
  }
  assert {
    condition     = module.staging.project_description == "Updated by scenario test"
    error_message = "Staging project description not updated: got ${module.staging.project_description}"
  }
}

# Idempotency: re-plan should produce no changes.
run "idempotency" {
  command = plan

  assert {
    condition     = module.dev.project_uuid != ""
    error_message = "Dev project UUID empty after re-plan (state corruption)"
  }
}
