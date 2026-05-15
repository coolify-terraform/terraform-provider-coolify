# NOTE: Import only restores API-readable fields. Keep token in your
# Terraform config before running terraform plan, because Coolify may
# omit it on read/import.
terraform import coolify_cloud_token.example <cloud-token-uuid>
