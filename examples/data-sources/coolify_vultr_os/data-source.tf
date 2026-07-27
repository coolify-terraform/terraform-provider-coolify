data "coolify_vultr_os" "all" {
  cloud_provider_token_uuid = coolify_cloud_token.vultr.uuid
}

output "vultr_os_names" {
  value = [for os in data.coolify_vultr_os.all.operating_systems : os.name]
}
