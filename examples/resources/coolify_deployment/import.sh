# Import is not supported for coolify_deployment resources.
# Deployments are ephemeral triggers. Re-run terraform apply to create a new deployment.
terraform import coolify_deployment.web <deployment-uuid>