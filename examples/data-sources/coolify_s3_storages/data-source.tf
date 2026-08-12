data "coolify_s3_storages" "all" {}

output "s3_storage_names" {
  value = [for s in data.coolify_s3_storages.all.s3_storages : s.name]
}
