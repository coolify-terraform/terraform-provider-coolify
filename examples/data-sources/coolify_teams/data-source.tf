data "coolify_teams" "all" {}

output "team_names" {
  value = [for t in data.coolify_teams.all.teams : t.name]
}
