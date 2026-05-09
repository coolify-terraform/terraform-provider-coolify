data "coolify_projects" "all" {}

output "project_names" {
  value = [for p in data.coolify_projects.all.projects : p.name]
}