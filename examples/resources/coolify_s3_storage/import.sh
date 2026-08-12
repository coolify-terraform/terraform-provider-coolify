# NOTE: Import only restores API-readable fields. Keep key and secret in your
# Terraform config before running terraform plan, because Coolify may omit
# them on read/import unless the API token can read sensitive fields.
terraform import coolify_s3_storage.example <s3-storage-uuid>
