# Plan-only test for the ACME Corp Private Repository Deployment scenario.
#
# This scenario requires real SSH credentials to apply, so the test validates
# HCL syntax, resource graph, variable types, and schema correctness via plan.
#
# Required variables are provided via TF_VAR_* environment variables:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

variables {
  deploy_ssh_key = "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key-for-plan-only\n-----END OPENSSH PRIVATE KEY-----"
  git_repository = "git@github.com:example/test-repo.git"
  database_url   = "postgresql://user:pass@db:5432/test"
  app_secret     = "test-secret-for-plan"
}

run "plan_only" {
  command = plan

  # --- Project ---
  assert {
    condition     = coolify_project.acme.name == "acme-private"
    error_message = "Project name mismatch: got ${coolify_project.acme.name}"
  }
  assert {
    condition     = coolify_project.acme.description == "ACME Corp private repo deployment"
    error_message = "Project description mismatch: got ${coolify_project.acme.description}"
  }

  # --- SSH Key ---
  assert {
    condition     = coolify_private_key.deploy.name == "acme-deploy-key"
    error_message = "Private key name mismatch: got ${coolify_private_key.deploy.name}"
  }

  # --- Application ---
  assert {
    condition     = coolify_private_git_application.backend.name == "acme-backend"
    error_message = "Application name mismatch: got ${coolify_private_git_application.backend.name}"
  }
  assert {
    condition     = coolify_private_git_application.backend.build_pack == "dockerfile"
    error_message = "Build pack mismatch: got ${coolify_private_git_application.backend.build_pack}"
  }
  assert {
    condition     = coolify_private_git_application.backend.git_repository == "git@github.com:example/test-repo.git"
    error_message = "Git repository mismatch: got ${coolify_private_git_application.backend.git_repository}"
  }
  assert {
    condition     = coolify_private_git_application.backend.git_branch == "main"
    error_message = "Git branch mismatch: got ${coolify_private_git_application.backend.git_branch}"
  }
  assert {
    condition     = coolify_private_git_application.backend.ports_exposes == "3000"
    error_message = "Ports exposes mismatch: got ${coolify_private_git_application.backend.ports_exposes}"
  }

  # --- Environment Variables ---
  assert {
    condition     = coolify_environment_variable.database_url.key == "DATABASE_URL"
    error_message = "Env var key mismatch: got ${coolify_environment_variable.database_url.key}"
  }
  assert {
    condition     = coolify_environment_variable.app_secret.key == "APP_SECRET"
    error_message = "Env var key mismatch: got ${coolify_environment_variable.app_secret.key}"
  }
}
