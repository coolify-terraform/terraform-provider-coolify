# ACME Corp GitHub App CI/CD Pipeline

This scenario demonstrates a complete CI/CD pipeline using GitHub App
integration, covering the full lifecycle from app registration to deployment:

1. **Project** (`coolify_project`) as the organizational container.
2. **Private key** (`coolify_private_key`) for GitHub App authentication.
3. **GitHub App** (`coolify_github_app`) registered with Coolify using
   your app credentials.
4. **Application** (`coolify_application_github_app`) deployed from a
   GitHub repository through the registered app.
5. **Environment variables** (`coolify_environment_variable`) for runtime
   configuration (DATABASE_URL, APP_ENV, LOG_LEVEL).
6. **Deployment** (`coolify_deployment`) with `wait_for_completion = true`
   to ensure the build finishes before Terraform reports success.

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
| `coolify_private_key.github_app` | PEM key for GitHub App auth |
| `coolify_github_app.acme` | GitHub App registration |
| `coolify_application_github_app.api` | Application from GitHub repo |
| `coolify_environment_variable.db_url` | Database connection string |
| `coolify_environment_variable.app_env` | Application environment |
| `coolify_environment_variable.log_level` | Log verbosity level |
| `coolify_deployment.api` | Initial deployment (waits for completion) |