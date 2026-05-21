# NOTE: github_app_uuid cannot be recovered from import (Coolify stores it
# internally as source_id). Set github_app_uuid in your .tf config BEFORE
# running terraform plan, or Terraform will propose replacing the resource.
# Simple import (project_uuid and server_uuid must be added to config manually):
terraform import coolify_application_github_app.example <application-uuid>

# Compound import (recommended, populates all required fields):
terraform import coolify_application_github_app.example <project-uuid>:<server-uuid>:<environment-name>:<application-uuid>
