output "coolify_version" {
  description = "Current Coolify version"
  value       = data.coolify_version.current.version
}

output "team_name" {
  description = "Root team name"
  value       = data.coolify_team.root.name
}

output "team_member_count" {
  description = "Number of root team members"
  value       = length(data.coolify_team_members.root.members)
}

output "server_count" {
  description = "Number of visible servers"
  value       = length(data.coolify_servers.all.servers)
}

output "project_count" {
  description = "Number of projects on the instance"
  value       = length(data.coolify_projects.all.projects)
}

output "target_server_name" {
  description = "Name of the target server"
  value       = data.coolify_server.target.name
}

output "target_server_ip" {
  description = "IP address of the target server"
  value       = data.coolify_server.target.ip
}

output "ops_environment_count" {
  description = "Number of environments in the ops project"
  value       = length(data.coolify_environments.ops.environments)
}