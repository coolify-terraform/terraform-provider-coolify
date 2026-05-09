data "coolify_servers" "all" {}

output "server_ips" {
  value = [for s in data.coolify_servers.all.servers : s.ip]
}