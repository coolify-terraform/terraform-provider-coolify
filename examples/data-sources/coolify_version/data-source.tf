data "coolify_version" "current" {}

output "coolify_version" {
  value = data.coolify_version.current.version
}
