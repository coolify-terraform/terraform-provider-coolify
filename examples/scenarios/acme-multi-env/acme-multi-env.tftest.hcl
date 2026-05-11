# Acceptance test for the ACME Corp Multi-Environment scenario.
#
# Required variables are provided via TF_VAR_* environment variables:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

run "create_both_environments" {
  command = apply

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
}
