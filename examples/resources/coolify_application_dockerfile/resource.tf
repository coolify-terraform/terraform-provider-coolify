# dockerfile_location expects Dockerfile content (not a file path, despite
# the field name). The provider accepts plain text or pre-encoded base64;
# encoding is handled automatically.
resource "coolify_application_dockerfile" "app" {
  name                = "my-dockerfile-app"
  project_uuid        = coolify_project.example.uuid
  server_uuid         = coolify_server.example.uuid
  dockerfile_location = <<-DOCKERFILE
    FROM node:20-alpine
    WORKDIR /app
    COPY . .
    RUN npm install --production
    EXPOSE 3000
    CMD ["node", "server.js"]
  DOCKERFILE
  ports_exposes       = "3000"
  domains             = "https://app.example.com"

  # Optional fields (uncomment as needed):
  # dockerfile_target_build = "production"  # Target stage for multi-stage Docker builds
}
