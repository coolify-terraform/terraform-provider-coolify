---
page_title: "Quick Start"
subcategory: "Getting Started"
description: |-
  Deploy your first application on Coolify with Terraform in under 5 minutes.
---

# Quick Start

This walkthrough takes you from zero to a running application on Coolify in
under five minutes. You will create a project and deploy a web application
from a public Git repository.

## 1. Set up credentials

```bash
export COOLIFY_ENDPOINT="https://coolify.example.com"
export COOLIFY_TOKEN="your-api-token"
```

## 2. Write the configuration

Create a file called `main.tf`:

```hcl
terraform {
  required_providers {
    coolify = {
      source = "SebTardif/coolify"
    }
  }
}

provider "coolify" {}

# Create a project to group resources
resource "coolify_project" "demo" {
  name        = "quickstart"
  description = "My first Terraform-managed project"
}

# Deploy an application from a public Git repository
resource "coolify_application" "app" {
  name           = "hello-world"
  project_uuid   = coolify_project.demo.uuid
  server_uuid    = "your-server-uuid"
  git_repository = "https://github.com/coollabsio/coolify-examples"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"
}

# Output the application UUID
output "app_uuid" {
  value = coolify_application.app.uuid
}
```

Replace `your-server-uuid` with the UUID of a server registered in Coolify.
Find it via the UI or with:

```bash
curl -s -H "Authorization: Bearer $COOLIFY_TOKEN" \
  "$COOLIFY_ENDPOINT/api/v1/servers" | jq '.[].uuid'
```

## 3. Initialize and apply

```bash
terraform init
terraform plan
terraform apply
```

Terraform creates the project and application. Coolify begins building and
deploying the app.

## 4. Verify

Check the application status:

```bash
curl -s -H "Authorization: Bearer $COOLIFY_TOKEN" \
  "$COOLIFY_ENDPOINT/api/v1/applications/$(terraform output -raw app_uuid)" \
  | jq '{name, status, fqdn}'
```

Or use a data source to read it back:

```hcl
data "coolify_application" "app" {
  uuid = coolify_application.app.uuid
}

output "app_status" {
  value = data.coolify_application.app.status
}
```

## 5. Clean up

```bash
terraform destroy
```

## Next steps

- Add a [database and environment variables](https://github.com/SebTardif/terraform-provider-coolify/tree/main/examples/full-stack)
  for a complete stack
- Set up [automated backups](https://github.com/SebTardif/terraform-provider-coolify/tree/main/examples/resources/coolify_database_backup)
  with S3 storage
- Read the [Import Guide](import) to bring existing Coolify resources
  under Terraform management
