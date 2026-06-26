# Import using the GitHub App ID (shown in Coolify UI under Sources > GitHub App > App Id).
# NOTE: Import only restores API-readable fields. Keep client_secret,
# webhook_secret, and private_key_uuid in your Terraform config before
# running terraform plan, because Coolify does not return them after
# create/import. If the provider generated webhook_secret for you on
# create, record it in your variables or secret manager before import.
terraform import coolify_github_app.example 12345
