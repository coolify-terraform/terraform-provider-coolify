# Multi-Environment Example

This example shows how to deploy the same application stack to multiple
environments (dev, staging, production) using a reusable Terraform module
and per-environment variable files.

## Usage

Initialize the workspace:

```bash
terraform init
```

Create a local `.tfvars` file from the tracked example, then deploy the target environment:

```bash
cp environments/dev.tfvars.example environments/dev.tfvars
cp environments/staging.tfvars.example environments/staging.tfvars
cp environments/prod.tfvars.example environments/prod.tfvars

# Development
terraform apply -var-file=environments/dev.tfvars

# Staging
terraform apply -var-file=environments/staging.tfvars

# Production
terraform apply -var-file=environments/prod.tfvars
```

Provide connection values and secrets outside version control:

```bash
export TF_VAR_coolify_endpoint="https://coolify.example.com"
export TF_VAR_coolify_token="your-api-token"
export TF_VAR_server_uuid="your-server-uuid"
export TF_VAR_db_password="change-me-in-production"

terraform apply -var-file=environments/dev.tfvars
```

## Structure

- `main.tf` - Root module that wires provider config to the app module.
- `modules/app/main.tf` - Reusable module that creates a project, database,
  application, and environment variable.
- `environments/*.tfvars.example` - Tracked environment templates. Copy one to a local `.tfvars` file before applying.
