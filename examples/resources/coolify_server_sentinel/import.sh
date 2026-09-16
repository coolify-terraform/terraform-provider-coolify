# NOTE: Coolify GET typically hides sentinel_token, so import cannot seed it.
# Keep sentinel_token in your Terraform config before terraform plan.
# Keep sentinel_custom_url in config too if GET still omits it (the provider
# seeds it when GET returns a non-empty value).
terraform import coolify_server_sentinel.example <server-uuid>
