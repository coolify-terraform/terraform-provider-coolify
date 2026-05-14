# Warning: Current Coolify (v4) may not expose the public API used by this data source.
# Prefer referencing UI-managed storage UUIDs unless you have confirmed the
# top-level storage read endpoints exist in your target version.

data "coolify_s3_storage" "example" {
  uuid = "550e8400-e29b-41d4-a716-446655440007"
}
