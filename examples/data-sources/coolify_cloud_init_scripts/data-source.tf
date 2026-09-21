data "coolify_cloud_init_scripts" "all" {}

output "cloud_init_script_names" {
  value = [for s in data.coolify_cloud_init_scripts.all.scripts : s.name]
}
