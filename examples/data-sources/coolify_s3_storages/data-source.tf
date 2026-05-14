# Warning: Current Coolify (v4) may not expose the public API used by this data source.
# Prefer referencing UI-managed storage UUIDs unless you have confirmed the
# top-level storage list endpoint exists in your target version.

data "coolify_s3_storages" "all" {}

output "storage_names" {
  value = [for s in data.coolify_s3_storages.all.storages : s.name]
}
