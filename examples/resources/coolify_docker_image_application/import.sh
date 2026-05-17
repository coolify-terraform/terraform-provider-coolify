# Simple import (project_uuid and server_uuid must be added to config manually):
terraform import coolify_docker_image_application.example <application-uuid>

# Compound import (recommended, populates all required fields):
terraform import coolify_docker_image_application.example <project-uuid>:<server-uuid>:<environment-name>:<application-uuid>
