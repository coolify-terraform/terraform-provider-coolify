# Acceptance test for ACME Corp GitHub App CI/CD pipeline.
#
# Tests: github_app, application_github_app, environment_variable (x3),
# deployment with wait-for-completion.
#
# Required variables via TF_VAR_*:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid,
#   TF_VAR_github_app_private_key, TF_VAR_github_app_id,
#   TF_VAR_github_app_installation_id, TF_VAR_github_app_client_id,
#   TF_VAR_github_app_client_secret, TF_VAR_git_repository

run "create_and_deploy" {
  command = apply

  # --- Project ---
  assert {
    condition     = coolify_project.acme.name == "acme-github-cicd"
    error_message = "Project name mismatch: got ${coolify_project.acme.name}"
  }

  # --- GitHub App ---
  assert {
    condition     = coolify_github_app.acme.name == "acme-cicd-app"
    error_message = "GitHub App name mismatch: got ${coolify_github_app.acme.name}"
  }

  # --- Application ---
  assert {
    condition     = coolify_application_github_app.api.name == "acme-api"
    error_message = "Application name mismatch: got ${coolify_application_github_app.api.name}"
  }
  assert {
    condition     = coolify_application_github_app.api.build_pack == "nixpacks"
    error_message = "Build pack mismatch: got ${coolify_application_github_app.api.build_pack}"
  }
  assert {
    condition     = coolify_application_github_app.api.ports_exposes == "3000"
    error_message = "Ports mismatch: got ${coolify_application_github_app.api.ports_exposes}"
  }

  # --- Environment Variables ---
  assert {
    condition     = coolify_environment_variable.db_url.key == "DATABASE_URL"
    error_message = "DB URL env var key mismatch: got ${coolify_environment_variable.db_url.key}"
  }
  assert {
    condition     = coolify_environment_variable.app_env.key == "APP_ENV"
    error_message = "APP_ENV key mismatch: got ${coolify_environment_variable.app_env.key}"
  }
  assert {
    condition     = coolify_environment_variable.log_level.key == "LOG_LEVEL"
    error_message = "LOG_LEVEL key mismatch: got ${coolify_environment_variable.log_level.key}"
  }
}