# Acceptance test for ACME Corp PR preview deployments.
#
# Tests: application_dockerfile, application_preview (x2), full lifecycle.
#
# Required variables via TF_VAR_*:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid

run "create_and_verify" {
  command = apply

  # --- Project ---
  assert {
    condition     = coolify_project.acme.name == "acme-preview-deploy"
    error_message = "Project name mismatch: got ${coolify_project.acme.name}"
  }

  # --- Application ---
  assert {
    condition     = coolify_application_dockerfile.web.name == "acme-web"
    error_message = "Application name mismatch: got ${coolify_application_dockerfile.web.name}"
  }
  assert {
    condition     = coolify_application_dockerfile.web.ports_exposes == "80"
    error_message = "Ports mismatch: got ${coolify_application_dockerfile.web.ports_exposes}"
  }

  # --- Preview Environments ---
  assert {
    condition     = coolify_application_preview.pr_1.pull_request_id == 1
    error_message = "PR 1 ID mismatch: got ${coolify_application_preview.pr_1.pull_request_id}"
  }
  assert {
    condition     = coolify_application_preview.pr_2.pull_request_id == 2
    error_message = "PR 2 ID mismatch: got ${coolify_application_preview.pr_2.pull_request_id}"
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
run "idempotency" {
  command = plan

  variables {
    project_description = "Updated by scenario test"
  }

  assert {
    condition     = coolify_project.acme.name == "acme-preview-deploy"
    error_message = "Project name changed after re-plan"
  }
}
