# Note: Current Coolify (v4) may not expose a public API for S3 storage.
# This data source targets an API surface that may not be available in your Coolify version.

data "coolify_s3_storages" "all" {}

output "storage_names" {
  value = [for s in data.coolify_s3_storages.all.storages : s.name]
}
