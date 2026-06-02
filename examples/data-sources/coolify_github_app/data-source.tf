# Look up a GitHub App by its numeric ID.
data "coolify_github_app" "example" {
  id = 1
}

output "github_app_uuid" {
  value = data.coolify_github_app.example.uuid
}
