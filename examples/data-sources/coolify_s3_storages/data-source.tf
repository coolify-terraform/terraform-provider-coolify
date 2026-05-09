data "coolify_s3_storages" "all" {}

output "storage_names" {
  value = [for s in data.coolify_s3_storages.all.storages : s.name]
}
