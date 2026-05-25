# Plan-only test for the ACME Corp GitHub App CI/CD Pipeline scenario.
#
# This scenario requires real GitHub App credentials to apply, so the test
# validates HCL syntax, resource graph, variable types, and schema correctness
# via plan.
#
# Required variables are provided via TF_VAR_* environment variables:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

variables {
  github_app_private_key = "-----BEGIN RSA PRIVATE KEY-----\ntest-key-for-plan-only\n-----END RSA PRIVATE KEY-----"
}

run "plan_only" {
  command = plan

  # --- Project ---
  assert {
    condition     = coolify_project.acme.name == "acme-github-cicd"
    error_message = "Project name mismatch: got ${coolify_project.acme.name}"
  }
  assert {
    condition     = coolify_project.acme.description == "ACME Corp GitHub App CI/CD pipeline"
    error_message = "Project description mismatch: got ${coolify_project.acme.description}"
  }

  # --- Private Key ---
  assert {
    condition     = coolify_private_key.github_app.name == "acme-github-app-key"
    error_message = "Private key name mismatch: got ${coolify_private_key.github_app.name}"
  }

  # --- GitHub App ---
  assert {
    condition     = coolify_github_app.acme.name == "acme-cicd-app"
    error_message = "GitHub App name mismatch: got ${coolify_github_app.acme.name}"
  }
  assert {
    condition     = coolify_github_app.acme.app_id == 12345
    error_message = "GitHub App ID mismatch: got ${coolify_github_app.acme.app_id}"
  }
  assert {
    condition     = coolify_github_app.acme.installation_id == 67890
    error_message = "GitHub App installation ID mismatch: got ${coolify_github_app.acme.installation_id}"
  }
  assert {
    condition     = coolify_github_app.acme.client_id == "Iv1.abc123def456"
    error_message = "GitHub App client ID mismatch: got ${coolify_github_app.acme.client_id}"
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
    condition     = coolify_application_github_app.api.git_repository == "https://github.com/coollabsio/coolify-examples"
    error_message = "Git repository mismatch: got ${coolify_application_github_app.api.git_repository}"
  }
  assert {
    condition     = coolify_application_github_app.api.git_branch == "main"
    error_message = "Git branch mismatch: got ${coolify_application_github_app.api.git_branch}"
  }
  assert {
    condition     = coolify_application_github_app.api.ports_exposes == "3000"
    error_message = "Ports exposes mismatch: got ${coolify_application_github_app.api.ports_exposes}"
  }

  # --- Environment Variables ---
  assert {
    condition     = coolify_environment_variable.db_url.key == "DATABASE_URL"
    error_message = "Env var key mismatch: got ${coolify_environment_variable.db_url.key}"
  }
  assert {
    condition     = coolify_environment_variable.app_env.key == "APP_ENV"
    error_message = "Env var key mismatch: got ${coolify_environment_variable.app_env.key}"
  }
  assert {
    condition     = coolify_environment_variable.app_env.value == "production"
    error_message = "Env var value mismatch: got ${coolify_environment_variable.app_env.value}"
  }
  assert {
    condition     = coolify_environment_variable.log_level.key == "LOG_LEVEL"
    error_message = "Env var key mismatch: got ${coolify_environment_variable.log_level.key}"
  }
  assert {
    condition     = coolify_environment_variable.log_level.value == "info"
    error_message = "Env var value mismatch: got ${coolify_environment_variable.log_level.value}"
  }

  # --- Deployment ---
  assert {
    condition     = coolify_deployment.api.wait_for_completion == true
    error_message = "Deployment should wait for completion"
  }
}