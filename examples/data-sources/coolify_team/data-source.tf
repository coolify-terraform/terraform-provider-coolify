data "coolify_team" "default" {
  id = 0
}

output "team_name" {
  value = data.coolify_team.default.name
}
