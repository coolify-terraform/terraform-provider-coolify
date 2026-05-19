# Simple import (project_uuid, server_uuid, environment_name must be added to config manually):
terraform import coolify_database_mongodb.example <database-uuid>

# Compound import (recommended, populates all required fields):
terraform import coolify_database_mongodb.example <project-uuid>:<server-uuid>:<environment-name>:<database-uuid>
