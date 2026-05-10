data "coolify_github_app_branches" "example" {
  github_app_id = 42
  owner         = "my-org"
  repo          = "my-repo"
}

output "branch_names" {
  value = [for b in data.coolify_github_app_branches.example.branches : b.name]
}
