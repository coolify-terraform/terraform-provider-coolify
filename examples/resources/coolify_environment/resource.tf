resource "coolify_environment" "example" {
  project_uuid = coolify_project.example.uuid
  name         = "staging"
  description  = "Staging environment"
}
