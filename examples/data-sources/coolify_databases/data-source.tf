data "coolify_databases" "all" {}

output "database_names" {
  value = [for d in data.coolify_databases.all.databases : d.name]
}