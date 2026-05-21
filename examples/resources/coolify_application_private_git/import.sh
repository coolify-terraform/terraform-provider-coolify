# NOTE: private_key_uuid cannot be recovered from import (Coolify stores
# only the numeric private_key_id). Set private_key_uuid in your .tf config
# BEFORE running terraform plan, or Terraform will propose replacing the resource.
# Simple import (project_uuid and server_uuid must be added to config manually):
terraform import coolify_application_private_git.example <application-uuid>

# Compound import (recommended, populates all required fields):
terraform import coolify_application_private_git.example <project-uuid>:<server-uuid>:<environment-name>:<application-uuid>
