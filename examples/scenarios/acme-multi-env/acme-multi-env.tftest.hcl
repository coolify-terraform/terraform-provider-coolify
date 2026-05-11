# Acceptance test for the ACME Corp Multi-Environment scenario.
#
# Defaults below are placeholders. For CI, override with TF_VAR_* env vars:
#
#   export TF_VAR_coolify_endpoint="https://coolify.example.com"
#   export TF_VAR_coolify_token="your-api-token"
#   export TF_VAR_server_uuid="your-server-uuid"

variables {
  coolify_endpoint = "http://localhost:8000"
  coolify_token    = "test-token"
  server_uuid      = "test-server-uuid"
}

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
