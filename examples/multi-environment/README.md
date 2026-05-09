# Multi-Environment Example

This example shows how to deploy the same application stack to multiple
environments (dev, staging, production) using a reusable Terraform module
and per-environment variable files.

## Usage

Initialize the workspace:

```bash
terraform init
```

Deploy to a specific environment:

```bash
# Development
terraform apply -var-file=environments/dev.tfvars

# Staging
terraform apply -var-file=environments/staging.tfvars

# Production
terraform apply -var-file=environments/prod.tfvars
```

You must also provide the required variables that are not in the tfvars files:

```bash
terraform apply \
  -var-file=environments/dev.tfvars \
  -var="coolify_endpoint=https://coolify.example.com" \
  -var="coolify_token=your-api-token" \
  -var="server_uuid=your-server-uuid"
```

## Structure

- `main.tf` - Root module that wires provider config to the app module.
- `modules/app/main.tf` - Reusable module that creates a project, database,
  application, and environment variable.
- `environments/*.tfvars` - Per-environment variable overrides.
