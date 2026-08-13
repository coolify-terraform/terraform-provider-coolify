## Highlights

v0.1.14 adds team notification channels and S3-compatible backup storage as Terraform resources. You can manage Discord, Slack, email, Telegram, webhook, and Pushover settings, plus S3 destinations, without using the Coolify UI.

## New features

- **Team notification channels (Coolify >= v4.3.0).** Six team-scoped singleton resources and matching data sources: `coolify_notification_discord`, `coolify_notification_slack`, `coolify_notification_email`, `coolify_notification_telegram`, `coolify_notification_webhook`, and `coolify_notification_pushover`. Import with id `current`. Destroy disables the channel (`enabled = false`) and leaves secrets in place ([#703](https://github.com/coolify-terraform/terraform-provider-coolify/pull/703), [#705](https://github.com/coolify-terraform/terraform-provider-coolify/pull/705), [#706](https://github.com/coolify-terraform/terraform-provider-coolify/pull/706), [#725](https://github.com/coolify-terraform/terraform-provider-coolify/pull/725)).

```hcl
resource "coolify_notification_discord" "team" {
  enabled     = true
  webhook_url = var.discord_webhook_url

  deployment_failure = true
  backup_failure     = true
  server_unreachable = true
}
```

- **`coolify_s3_storage` (Coolify >= v4.3.0).** Manage S3-compatible storage for backups (endpoint, bucket, region, and credentials). Use a public HTTPS endpoint; Coolify rejects most private and Docker DNS hostnames ([#689](https://github.com/coolify-terraform/terraform-provider-coolify/pull/689)).

- **`coolify_s3_storage_validate` (Coolify >= v4.3.0).** Re-test an S3 destination after credential rotation, or gate backups on a successful validate. Change the `triggers` map to run validation again ([#698](https://github.com/coolify-terraform/terraform-provider-coolify/pull/698)).

- **Server Docker and Compose version probes (Coolify >= v4.3.2).** Read-only `docker_version`, `compose_version`, and `*_checked_at` attributes on `coolify_server` and the cloud server resources. Empty on older Coolify instances ([#708](https://github.com/coolify-terraform/terraform-provider-coolify/pull/708)).

## Bug fixes

- **Failed `wait_for_completion` deployments now include Coolify log lines.** When a deployment ends in `failed` or `error`, the provider error includes the last 40 log lines when the API token can read them. If logs are empty, the message points at the Coolify Deployments UI ([#730](https://github.com/coolify-terraform/terraform-provider-coolify/pull/730)).

- **Clearer data source errors.** Project, environment, S3 storage, and cloud token data sources include the UUID or name in diagnostics so multi-resource applies are easier to attribute ([#694](https://github.com/coolify-terraform/terraform-provider-coolify/pull/694)).

## Documentation

- Coolify version support matrix: which resources and fields need which Coolify version ([#688](https://github.com/coolify-terraform/terraform-provider-coolify/pull/688)). Guide: [Coolify version support](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs/guides/coolify-version-support).

## Compatibility

| Surface | Requires |
|---------|----------|
| Existing resources from v0.1.13 | Coolify **>= 4.1.0** (unchanged) |
| Notification channels and S3 storage / validate | Coolify **>= v4.3.0** |
| Server Docker / Compose version attributes | Coolify **>= v4.3.2** (empty on older instances) |

The provider still refuses to configure against Coolify older than 4.1.0. The pinned API contract for this release is Coolify **v4.3.2** ([#709](https://github.com/coolify-terraform/terraform-provider-coolify/pull/709)).

Provider binaries are built with Go **1.26.6** (stdlib CVEs GO-2026-6218, 6091, 6090, 6089, 6088, and 5972) ([#734](https://github.com/coolify-terraform/terraform-provider-coolify/pull/734)).

## Numbers

| Metric | v0.1.13 | v0.1.14 |
|--------|---------|---------|
| Resources | 37 | 45 |
| Data sources | 54 | 62 |
| Tests (unit + acceptance floor) | 1150+ | 1330+ |

## Upgrading

```hcl
terraform {
  required_providers {
    coolify = {
      source  = "coolify-terraform/coolify"
      version = "~> 0.1.14"
    }
  }
}
```

No state migration is required. Existing configurations keep working. New notification and S3 resources are opt-in and need Coolify >= v4.3.0.

Docs: [Terraform Registry](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs).

## Full changelog

https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.13...v0.1.14
