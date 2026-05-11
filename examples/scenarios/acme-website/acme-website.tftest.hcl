# Acceptance test for the ACME Corp Marketing Website scenario.
#
# Defaults below are placeholders that will not connect to a real Coolify
# instance. For CI, override them with TF_VAR_* environment variables:
#
#   export TF_VAR_coolify_endpoint="https://coolify.example.com"
#   export TF_VAR_coolify_token="your-api-token"
#   export TF_VAR_server_uuid="your-server-uuid"

variables {
  coolify_endpoint = "http://localhost:8000"
  coolify_token    = "test-token"
  server_uuid      = "test-server-uuid"
  enable_backups   = false
}

run "create_and_verify" {
  command = apply

  assert {
    condition     = coolify_project.acme.uuid != ""
    error_message = "Project was not created: uuid is empty"
  }

  assert {
    condition     = coolify_postgresql_database.content.uuid != ""
    error_message = "PostgreSQL database was not created: uuid is empty"
  }

  assert {
    condition     = coolify_application.website.uuid != ""
    error_message = "Application was not created: uuid is empty"
  }

  assert {
    condition     = coolify_environment_variable.database_url.uuid != ""
    error_message = "DATABASE_URL environment variable was not created: uuid is empty"
  }

  assert {
    condition     = coolify_environment_variable.node_env.uuid != ""
    error_message = "NODE_ENV environment variable was not created: uuid is empty"
  }
}
