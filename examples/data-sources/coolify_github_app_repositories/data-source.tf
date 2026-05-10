data "coolify_github_app_repositories" "example" {
  github_app_id = 42
}

output "repository_names" {
  value = [for r in data.coolify_github_app_repositories.example.repositories : r.name]
}
