# NOTE: Import only restores API-readable fields. Keep client_secret,
# webhook_secret, and private_key_uuid in your Terraform config before
# running terraform plan, because Coolify does not return them after
# create/import.
terraform import coolify_github_app.example 42
