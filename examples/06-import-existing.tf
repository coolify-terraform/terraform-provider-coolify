# 06 - Import Existing Resources: bring Coolify resources under Terraform
#
# Shows how to write resource blocks for existing Coolify resources
# and import them into Terraform state. After import, Terraform
# manages the resource lifecycle.
#
# Prerequisites:
#   - Existing resources in Coolify (project, application, database)
#   - Their UUIDs (find via API or Coolify UI)
#
# Step 1: Write the resource blocks below with matching attributes.
# Step 2: Import each resource:
#   terraform import coolify_project.existing <project-uuid>
#   terraform import coolify_application.existing <app-uuid>
#   terraform import coolify_postgresql_database.existing <db-uuid>
#
# Step 3: Run terraform plan to verify no changes are detected.
# Step 4: Adjust attributes until plan shows no diff.

terraform {
  required_providers {
    coolify = {
      source = "SebTardif/coolify"
    }
  }
}

provider "coolify" {}

# Import: terraform import coolify_project.existing <uuid>
resource "coolify_project" "existing" {
  name        = "existing-project"
  description = "Imported into Terraform"
}

# Import: terraform import coolify_application.existing <uuid>
resource "coolify_application" "existing" {
  project_uuid   = coolify_project.existing.uuid
  server_uuid    = "your-server-uuid"
  git_repository = "https://github.com/your-org/your-repo"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"
}

# Import with composite ID:
#   terraform import coolify_environment_variable.db_url \
#     application:<app-uuid>:<env-var-uuid>
resource "coolify_environment_variable" "db_url" {
  application_uuid = coolify_application.existing.uuid
  key              = "DATABASE_URL"
  value            = "postgresql://user:pass@db:5432/myapp"
  is_build_time    = false
}
