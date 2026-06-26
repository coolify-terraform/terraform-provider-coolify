# ACME Corp Import Blocks (Terraform 1.5+)
#
# This file shows the declarative import syntax as an alternative to
# running `terraform import` commands. Add these blocks, run
# `terraform plan` to preview, then `terraform apply` to execute.
#
# In a real scenario, replace the UUIDs below with your actual values
# from the Coolify UI or API. The blocks below are placeholders that
# illustrate the syntax for each resource type.

# --- Project: simple UUID ---
# import {
#   to = coolify_project.existing
#   id = "<project-uuid>"
# }

# --- Application: compound format ---
# The compound format (project:server:env:app) populates project_uuid,
# server_uuid, and environment_name automatically, avoiding post-import
# diffs. The simple UUID format also works but may require setting those
# fields manually.
# import {
#   to = coolify_application_docker_image.web
#   id = "<project-uuid>:<server-uuid>:production:<app-uuid>"
# }

# --- Environment variable: composite key ---
# Format: resource_type:parent_uuid:env_var_uuid
# resource_type is one of: application, service, database
# import {
#   to = coolify_environment_variable.app_env
#   id = "application:<app-uuid>:<env-var-uuid>"
# }
