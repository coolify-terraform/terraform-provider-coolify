# NOTE: Import assumes environment_name = "production". If your app
# is in a different environment, set environment_name in your config
# to match BEFORE running terraform plan.
# Simple import (project_uuid and server_uuid must be added to config manually):
terraform import coolify_application.example <application-uuid>

# Compound import (recommended, populates all required fields):
terraform import coolify_application.example <project-uuid>:<server-uuid>:<environment-name>:<application-uuid>