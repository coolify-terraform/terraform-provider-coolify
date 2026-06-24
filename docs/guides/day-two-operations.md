---
page_title: "Day-2 Operations"
subcategory: "Guides"
description: |-
  Upgrading Coolify and the provider, rolling back deployments, and disaster recovery.
---

# Day-2 Operations

This guide covers what happens after your initial setup: upgrading
Coolify, upgrading the provider, rolling back deployments, and
recovering from failures.

## Upgrading Coolify

When you upgrade your Coolify instance (e.g., from v4.0 to v4.1),
the API may gain new fields, endpoints, or changed defaults.

### What to expect

- **Existing resources continue to work.** Coolify maintains backward
  compatibility within the v4 line.
- **New fields become available.** For example, Coolify v4.1 added
  the `railpack` build pack. The provider already supports it, but
  it only works if your Coolify instance is v4.1+.
- **The provider validates the Coolify version.** On every `plan` and
  `apply`, the provider checks `/api/v1/version`. If the version is
  below v4.1.0, it refuses to continue.

### Before upgrading

1. Check the [Coolify changelog](https://github.com/coollabsio/coolify/releases)
   for breaking changes
2. Run `terraform plan` to confirm your current state is clean (no pending changes)
3. Upgrade Coolify
4. Run `terraform plan` again to see if any resource attributes changed

### Checking your Coolify version

```hcl
data "coolify_version" "current" {}

output "coolify_version" {
  value = data.coolify_version.current.version
}
```

## Upgrading the Provider

### Updating the version constraint

When a new provider version is released:

```hcl
terraform {
  required_providers {
    coolify = {
      source  = "coolify-terraform/coolify"
      version = "~> 0.5"  # update to the new major/minor
    }
  }
}
```

```bash
terraform init -upgrade
```

### Checking for breaking changes

Read the [CHANGELOG](https://github.com/coolify-terraform/terraform-provider-coolify/blob/main/CHANGELOG.md)
before upgrading. Breaking changes are listed under the **Breaking Changes**
section. Common breaking changes include:

- Renamed attributes (e.g., `host` to `endpoint`, `retain_days` to `retain_amount_locally`)
- Removed resources (e.g., `coolify_s3_storage` was removed because the API endpoint no longer exists)
- Changed attribute types or validation rules

After upgrading:

```bash
terraform init -upgrade
terraform plan  # check for unexpected diffs or errors
```

If `terraform plan` shows unexpected changes, check if a renamed
attribute needs updating in your `.tf` files.

## Rolling Back Deployments

### Re-deploy a previous version

If a deployment fails or introduces a bug, re-deploy the previous version
by changing the `triggers` value:

```hcl
resource "coolify_deployment" "web" {
  application_uuid    = coolify_application.web.uuid
  wait_for_completion = true

  triggers = {
    deploy_version = "v1.1.0"  # roll back from v1.2.0 to v1.1.0
  }
}
```

```bash
terraform apply -var="deploy_version=v1.1.0"
```

### Emergency stop via resource action

If you need to stop an application immediately without waiting for a new
build:

```hcl
resource "coolify_resource_action" "stop" {
  resource_uuid = coolify_application.web.uuid
  resource_type = "application"
  action        = "stop"
}
```

After fixing the issue, change the action to `"start"` or `"restart"`
and re-apply.

### Using the Coolify UI

For urgent situations, use the Coolify web UI directly. The UI provides
immediate start/stop/restart controls. After resolving the issue, run
`terraform plan` to reconcile state. The provider reads the current
resource state from the API, so manual UI changes are detected.

## `redeploy_on_update` vs `coolify_deployment`

These two mechanisms serve different purposes:

| Feature | `redeploy_on_update` | `coolify_deployment` |
|---------|---------------------|---------------------|
| Defined on | Application resource | Separate resource |
| Triggers on | Any config attribute change | `triggers` map value change |
| Wait for completion | No | Yes (with `wait_for_completion`) |
| Explicit control | No (automatic) | Yes (you decide when) |
| CI/CD friendly | Less (no wait) | Yes (wait + timeout) |

**Use `redeploy_on_update`** when you want automatic restarts whenever
the application config changes (ports, domains, build settings).

**Use `coolify_deployment`** when you want explicit deploy control,
especially in CI/CD pipelines where you need to wait for the build to
finish.

You can use both together. `redeploy_on_update` handles config-driven
restarts, while `coolify_deployment` handles explicit version deploys.

## Disaster Recovery

### Recreating from Terraform files

If your Coolify instance is lost and rebuilt from scratch:

1. Install Coolify on the new server
2. Register the server and get its UUID
3. Update `server_uuid` in your `.tf` files
4. Run `terraform apply` to recreate all resources

Terraform creates fresh resources. Data (database contents, uploaded
files) is not restored by Terraform; restore those from backups.

### Importing existing resources

If resources exist in Coolify but not in Terraform state (state was
lost, or resources were created via UI):

```bash
# Import each resource
terraform import coolify_project.main <project-uuid>
terraform import coolify_application.web <app-uuid>
terraform import coolify_database_postgresql.db <db-uuid>

# Check for diffs
terraform plan
```

See the [Import Guide](import) for composite ID formats and known
limitations.

### State backup

Always back up your Terraform state. If using a remote backend, the
backend handles this. For local state:

```bash
# Before any risky operation
cp terraform.tfstate terraform.tfstate.backup
```

## Maintenance Windows

Use `coolify_resource_action` to stop resources during maintenance:

```hcl
variable "maintenance_mode" {
  type    = bool
  default = false
}

resource "coolify_resource_action" "maintenance" {
  count         = var.maintenance_mode ? 1 : 0
  resource_uuid = coolify_application.web.uuid
  resource_type = "application"
  action        = "stop"
}
```

```bash
# Enter maintenance
terraform apply -var="maintenance_mode=true"

# Perform maintenance...

# Exit maintenance
terraform apply -var="maintenance_mode=false"
```

When `maintenance_mode` becomes `false`, the `coolify_resource_action`
resource is destroyed (removed from state), but the application is not
automatically restarted. Use a `coolify_deployment` or the Coolify UI
to restart after maintenance.
