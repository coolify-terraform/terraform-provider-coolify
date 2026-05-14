# Note: Current Coolify (v4) may not expose a public API for S3 storage.
# This data source targets an API surface that may not be available in your Coolify version.

data "coolify_s3_storage" "example" {
  uuid = "550e8400-e29b-41d4-a716-446655440007"
}
