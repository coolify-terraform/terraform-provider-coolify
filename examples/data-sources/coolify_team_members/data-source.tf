data "coolify_team_members" "current" {}

output "member_emails" {
  value = [for m in data.coolify_team_members.current.members : m.email]
}

data "coolify_team_members" "by_id" {
  id = 1
}

output "team_1_members" {
  value = [for m in data.coolify_team_members.by_id.members : m.name]
}
