# Simple import (project_uuid, server_uuid, environment_name must be added to config manually):
terraform import coolify_service.example <service-uuid>

# Compound import (recommended, populates all required fields):
terraform import coolify_service.example <project-uuid>:<server-uuid>:<environment-name>:<service-uuid>
