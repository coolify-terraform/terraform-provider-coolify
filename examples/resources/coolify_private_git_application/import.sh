# NOTE: Import assumes environment_name = "production". If your app
# is in a different environment, set environment_name in your config
# to match BEFORE running terraform plan.
terraform import coolify_private_git_application.api <application-uuid>
