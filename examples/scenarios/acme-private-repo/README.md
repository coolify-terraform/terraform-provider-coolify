# ACME Corp Private Repository Deployment

This scenario deploys an application from a private Git repository,
demonstrating the full dependency chain for production workflows:

1. **SSH deploy key** (`coolify_private_key`) for authenticating to the
   private Git repository.
2. **Private Git application** (`coolify_private_git_application`) that
   clones and builds the repository.
3. **Environment variables** (`coolify_environment_variable`) for
   secrets and configuration.
4. **Deployment** (`coolify_deployment`) with `wait_for_completion = true`
   to ensure the build finishes successfully before Terraform reports
   success.

## Usage

```bash
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values
terraform init
terraform apply
```

## Resources Created

| Resource | Purpose |
|----------|---------|
| `coolify_project.acme` | Project container |
| `coolify_private_key.deploy` | SSH key for Git auth |
| `coolify_private_git_application.backend` | Application from private repo |
| `coolify_environment_variable.database_url` | Database connection string |
| `coolify_environment_variable.app_secret` | Application secret |
| `coolify_deployment.backend` | Initial deployment (waits for completion) |
