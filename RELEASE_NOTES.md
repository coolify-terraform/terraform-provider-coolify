## Highlights

v0.1.10 deepens day-to-day Coolify control in Terraform: environment variable
deploy flags, application preview and stop settings, scheduled task targeting,
and GitHub App SSH options. It also fixes plan-time validation with variables,
accepts short Coolify IDs again, and makes API errors easier to read.

Minimum Coolify is still **4.1.0**. A few application settings need **>= v4.2.0**
(see Compatibility).

## New features

- **Environment variable deploy flags.** On `coolify_environment_variable` you
  can set `is_runtime`, `is_literal`, `is_multiline`, and `comment`.
  `is_runtime` and `is_build` apply to applications; service and database
  variables omit those application-only flags.
  ([#625](https://github.com/coolify-terraform/terraform-provider-coolify/pull/625))

- **Application preview, build secrets, and stop grace.** Optional attributes
  on applications: `is_preview_deployments_enabled`, `use_build_secrets`, and
  `stop_grace_period` (1–3600 seconds). Preview and build-secrets need
  Coolify **>= v4.2.0**. Nested Coolify `settings` values are promoted into
  these top-level attributes on read.
  ([#630](https://github.com/coolify-terraform/terraform-provider-coolify/pull/630))

- **Scheduled task container and timeout.** `coolify_scheduled_task` (and the
  matching data sources) support `container` and `timeout` (default 300
  seconds). Six-field cron (with seconds) remains valid for task frequency.
  ([#630](https://github.com/coolify-terraform/terraform-provider-coolify/pull/630))

- **GitHub App SSH and system-wide flag.** On `coolify_github_app`: `custom_user`
  (default `git`), `custom_port` (default `22`, range 1–65535), and
  `is_system_wide` (default `false`; applied on self-hosted Coolify).
  ([#633](https://github.com/coolify-terraform/terraform-provider-coolify/pull/633))

Example (application env and task):

```hcl
resource "coolify_environment_variable" "api_url" {
  application_uuid = coolify_application.app.uuid
  key              = "API_URL"
  value            = "https://api.example.com"
  is_runtime       = true
  is_literal       = true
  comment          = "Public API base URL"
}

resource "coolify_scheduled_task" "backup" {
  application_uuid = coolify_application.app.uuid
  name             = "backup-db"
  command          = "pg_dump mydb"
  frequency        = "0 2 * * *"
  container        = "worker"
  timeout          = 600
}
```

## Bug fixes

- **Plan failed when service compose or backup fields used variables.**
  `ValidateConfig` treated unknown values as missing, so references like
  `docker_compose_raw = var.workers_compose` or backup `save_s3` /
  `s3_storage_uuid` set from other resources failed at plan. Unknown values
  are skipped until they are known. Thanks
  [@oter](https://github.com/oter) for the service compose fix
  ([#618](https://github.com/coolify-terraform/terraform-provider-coolify/pull/618);
  follows [#616](https://github.com/coolify-terraform/terraform-provider-coolify/pull/616)).
  Backup unknown-value guards:
  ([#620](https://github.com/coolify-terraform/terraform-provider-coolify/pull/620))

- **Short Coolify IDs rejected as invalid UUIDs.** Coolify still issues
  7-character identifiers on some resources. UUID validation accepts those
  legacy ids again (full UUIDs remain valid). Thanks
  [@oter](https://github.com/oter)
  ([#611](https://github.com/coolify-terraform/terraform-provider-coolify/pull/611)).
  Helper and docs coverage:
  ([#615](https://github.com/coolify-terraform/terraform-provider-coolify/pull/615))

- **Harder-to-debug API failures.** Client errors include HTTP method and
  path (for example `expected status 201, got 200 for POST /api/v1/projects`),
  and docs cover Coolify ID format troubleshooting.
  ([#617](https://github.com/coolify-terraform/terraform-provider-coolify/pull/617),
  [#620](https://github.com/coolify-terraform/terraform-provider-coolify/pull/620))

## Compatibility

| Surface | Coolify requirement |
|---------|---------------------|
| Existing resources from v0.1.9 | Still work on **>= 4.1.0** |
| `is_preview_deployments_enabled`, `use_build_secrets` | **>= v4.2.0** |
| Env flags, scheduled task container/timeout, GitHub App SSH fields | Available where your Coolify API exposes them (all supported contract pins) |
| `coolify_storage_backup` (from v0.1.9) | Still needs Coolify tip after [coolify#10946](https://github.com/coollabsio/coolify/pull/10946) |

## Numbers

| | v0.1.9 | v0.1.10 |
|--|--------|---------|
| Resources | 37 | 37 |
| Data sources | 54 | 54 |
| Unit + acceptance tests | 1060+ | 1100+ |
| ACME scenarios | 17 | 17 |
| Minimum Coolify | 4.1.0 | 4.1.0 |

## Upgrading

```hcl
terraform {
  required_providers {
    coolify = {
      source  = "coolify-terraform/coolify"
      version = "~> 0.1.10"
    }
  }
}
```

```bash
terraform init -upgrade
```

No state migration is required. New attributes are optional; import and
existing configs keep working. If you manage applications on Coolify older
than 4.2, leave preview and build-secrets attributes unset.

Docs: https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs

## Contributors

Thanks to external contributors in this release:

- [@oter](https://github.com/oter) for accepting short Coolify identifiers
  in UUID validation
  ([#611](https://github.com/coolify-terraform/terraform-provider-coolify/pull/611))
- [@oter](https://github.com/oter) for accepting unknown
  `docker_compose_raw` / `type` values in service plan validation
  ([#618](https://github.com/coolify-terraform/terraform-provider-coolify/pull/618);
  original PR
  [#616](https://github.com/coolify-terraform/terraform-provider-coolify/pull/616))

## Full changelog

https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.9...v0.1.10
