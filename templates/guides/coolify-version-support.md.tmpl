---
page_title: "Coolify Version Support"
subcategory: "Getting Started"
description: |-
  What works on Coolify 4.1, 4.2, and 4.3 with this provider: resources, fields, and version-gate behavior.
---

# Coolify Version Support

This guide answers: **which Coolify version do I need for each resource and field?**

For Coolify HTTP routes the provider does not wrap (clone, rollback, log
streams, one-shot backup run, nested compose apps), see
[API Coverage](https://github.com/coolify-terraform/terraform-provider-coolify/blob/main/API_COVERAGE.md)
on GitHub. That page lists each skip class and **what to use instead**.

The provider hard-requires Coolify **v4.1.0 or later** on every `terraform plan` /
`terraform apply`. Within that floor, some APIs only exist on newer Coolify
lines. The provider still configures against older supported instances, but it
**withholds** writes that would 422 and emits a **plan warning** when you set
those attributes.

-> **Check your instance** (see also [Installation](installation)):

```hcl
data "coolify_version" "current" {}

output "coolify_version" {
  value = data.coolify_version.current.version
}
```

## Support model at a glance

| Coolify | Provider behavior |
|---------|-------------------|
| **&lt; 4.1.0** | Configure fails. Upgrade Coolify. |
| **4.1.x** | Core resources work. Version-gated 4.2/4.3 attributes stay in state if set, but are **not sent** on write (plan warning). |
| **4.2.x** | 4.2 resources and application settings writes work. 4.3-only attributes still withheld with a plan warning. |
| **≥ 4.3.0** | 4.3 application settings, `noindex_domains`, notification channels, S3, volume backup schedules, GPU/log-drain, etc. |
| **≥ 4.3.10** | Instance email settings and `smtp_ehlo_domain` on team email notifications (email/SMTP floor). |
| **≥ 4.3.15** | Preview domain PATCH, GET `domain_port_overrides`, restart-limit fields. Recommended for the full feature set. |

Pinned API contract today: Coolify **v4.3.17** (`testdata/contracts/coolify-v4.json`).
Coolify 4.3.6 and 4.3.7 match 4.3.5. From 4.3.8, nested compose service apps
accept `is_force_https_enabled` on `PATCH /services/{uuid}/applications/{app_uuid}`.
That route stays `nested-service` (use `coolify_service` for the stack).
v4.3.10 adds instance-wide SMTP settings (`GET`/`PATCH /settings/email`) and
`smtp_ehlo_domain` on team email notifications. Tags v4.3.15 through v4.3.17
add restart-limit GET fields, GET-only `domain_port_overrides`, notification
`restart_limit_reached_*` writes, and `PATCH` preview domains on
`coolify_application_preview`. `v4.4-rc.1` was cut before those 4.3.15
fields. The provider remains usable on 4.1.0+ for the common surface.

## Resources and data sources by Coolify version

### Always available (Coolify ≥ 4.1.0)

Core lifecycle, apps, databases, services, env vars, keys, projects, Hetzner
cloud tokens, GitHub Apps, scheduled tasks, storage volumes, deployments,
teams, and most list/single data sources.

| Resource / data source family | Notes |
|-------------------------------|--------|
| `coolify_project`, `coolify_environment` | |
| `coolify_server`, `coolify_server_hetzner` | Hetzner provision path (`coolify_server_validate` needs 4.2+) |
| `coolify_application*` (public, docker image, dockerfile, private git, github app) | Some **attributes** need 4.2/4.3 (see below) |
| `coolify_application_preview` | Domain attributes (`domains`, `docker_compose_domains`, `force_domain_override`) need Coolify >= v4.3.15 (hard apply error on older) |
| Eight database engines + `coolify_database_backup`, `coolify_backup_execution` | Core CRUD works on 4.1+. Some optional settings exist only on newer Coolify lines; omit attributes you do not need so plans stay empty on older instances. |
| `coolify_service`, `coolify_storage` | |
| `coolify_environment_variable`, `coolify_envs_bulk` | |
| `coolify_private_key`, `coolify_github_app` | |
| `coolify_cloud_token`, `coolify_cloud_token_validate` (Hetzner and generic) | |
| `coolify_scheduled_task` | |
| `coolify_deployment`, `coolify_resource_action` | |
| `coolify_team` / teams | |
| `coolify_version` | Read instance version |
| Most list/single data sources without a version note in the README | |

### Requires Coolify ≥ 4.2.0

| Resource / data source | Role |
|------------------------|------|
| `coolify_destination` / `coolify_destinations` | Docker network destinations |
| `coolify_server_validate`, `coolify_server_validation` | Provider uses POST validate; 4.1.x only exposes GET |
| `coolify_api_settings` | Provider uses POST `/enable`; 4.1.x only exposes GET (MCP enable alone is older) |
| `coolify_server_digitalocean` | Provision DO droplets into Coolify |
| `coolify_server_vultr` | Provision Vultr instances into Coolify |
| `coolify_digitalocean_*` data sources | regions, sizes, images, ssh_keys |
| `coolify_vultr_*` data sources | regions, plans, os, ssh_keys |
| `coolify_hetzner_firewalls` / `coolify_hetzner_networks` | List existing Hetzner firewalls and private networks |
| `hetzner_firewall_ids` / `hetzner_network_ids` on `coolify_server_hetzner` | Attach existing Hetzner firewalls and networks at create time |

`destination_uuid` on app, database, or service create works on Coolify 4.1.0+.
Managing destinations as `coolify_destination` (and multi-destination servers)
needs Coolify >= 4.2.0; see [Connecting Resources](connecting-resources).

### Requires Coolify ≥ 4.3.0

| Resource / data source | Role |
|------------------------|------|
| `coolify_storage_backup` | Scheduled volume/directory backups (VolumeBackups API, [coollabsio/coolify#10946](https://github.com/coollabsio/coolify/pull/10946)) |
| `coolify_s3_storage` / `coolify_s3_storages` | Manage and list S3-compatible storage configurations (database backup targets) |
| `coolify_s3_storage_validate` | POST validate S3 connectivity (after credential rotation) |
| `coolify_notification_discord` | Team Discord notification settings |
| `coolify_notification_slack` | Team Slack notification settings |
| `coolify_notification_email` | Team email (SMTP/Resend) notification settings |
| `coolify_instance_email_settings` | Instance-wide SMTP/Resend resource and data source (`GET`/`PATCH /settings/email`). **Requires Coolify >= v4.3.10** and a root-team token. |
| `coolify_notification_telegram` | Team Telegram notification settings |
| `coolify_notification_webhook` | Team generic webhook notification settings |
| `coolify_notification_pushover` | Team Pushover notification settings |
| `coolify_server_sentinel` | Sentinel host metrics agent (GET/PATCH `/servers/{uuid}/sentinel`) |
| `coolify_environment` `description` | Create still works on 4.1.x without description; PATCH description is 4.3.0+ |

Coolify 4.3 also added instance SSO (Authentik, Zitadel, Clerk) in the UI.
Those login providers have no public API routes in the contract, so this
provider cannot manage them.

Server host version probes (`compose_version`, `docker_version`, and `*_checked_at`)
on `coolify_server` / cloud server resources and `coolify_server` / `coolify_servers`
data sources are populated on Coolify **≥ 4.3.2** after Coolify probes the host.
Empty on older instances.

Volume-backup CPU share (`backup_compression_cpu_percentage`) is a computed
GET-only server setting on Coolify **≥ 4.3.3**. It is not on the
public server PATCH allow list. Empty on older instances.

Project icon attributes (`icon_path`, `icon_storage_type`) are computed on
`coolify_project` and `data.coolify_project` from GET `/projects/{uuid}` on
Coolify **≥ 4.3.3**. They are not writable and are omitted from
`GET /projects` (list). Empty on older instances.

`domain_port_overrides` is computed on application resources and
`data.coolify_application` on Coolify **>= v4.3.15**. It is null on
v4.3.14 and earlier. It is not on `data.coolify_applications` and is
not writable.

`coolify_application_preview` can PATCH preview domains
(`domains` / `docker_compose_domains` / `force_domain_override`) on
Coolify **>= v4.3.15**. Omit those attributes on older instances.

## Application attributes by Coolify version

Application resources share one schema. Attributes that Coolify only accepts
from a given version are **version-gated on write**:

1. Values remain in Terraform state if you set them.
2. The client **omits** them from POST/PATCH when the instance is too old.
3. Plan/apply emits a warning listing the attributes (see
   [Common Errors](common-errors#coolify-version-cannot-write-some-application-settings)).

### Writes require Coolify ≥ 4.2.0

| Terraform attribute | API JSON key |
|---------------------|--------------|
| `is_preview_deployments_enabled` | `is_preview_deployments_enabled` |
| `use_build_secrets` | `use_build_secrets` |
| `is_git_submodules_enabled` | `is_git_submodules_enabled` |
| `is_git_lfs_enabled` | `is_git_lfs_enabled` |
| `is_git_shallow_clone_enabled` | `is_git_shallow_clone_enabled` |
| `disable_build_cache` | `disable_build_cache` |
| `inject_build_args_to_dockerfile` | `inject_build_args_to_dockerfile` |
| `include_source_commit_in_build` | `include_source_commit_in_build` |
| `is_env_sorting_enabled` | `is_env_sorting_enabled` |
| `is_pr_deployments_public_enabled` | `is_pr_deployments_public_enabled` |
| `stop_grace_period` | `stop_grace_period` |
| `docker_images_to_keep` | `docker_images_to_keep` |
| `is_gzip_enabled` | `is_gzip_enabled` |
| `is_stripprefix_enabled` | `is_stripprefix_enabled` |
| `is_raw_compose_deployment_enabled` | `is_raw_compose_deployment_enabled` |

### Writes require Coolify ≥ 4.3.0

| Terraform attribute | API JSON key |
|---------------------|--------------|
| `is_log_drain_enabled` | `is_log_drain_enabled` |
| `is_gpu_enabled` | `is_gpu_enabled` |
| `gpu_driver` | `gpu_driver` |
| `gpu_count` | `gpu_count` |
| `gpu_device_ids` | `gpu_device_ids` |
| `gpu_options` | `gpu_options` |
| `is_consistent_container_name_enabled` | `is_consistent_container_name_enabled` |
| `custom_internal_name` | `custom_internal_name` |
| `noindex_domains` | `noindex_domains` (list of domain URLs) |
| `max_restart_count` | `max_restart_count` |

### Matrix (application write gates)

| Attribute group | 4.1.x | 4.2.x | ≥ 4.3.0 |
|-----------------|-------|-------|---------|
| Core app fields (name, domains, build, limits, health checks, …) | Yes | Yes | Yes |
| 4.2 settings / preview / build secrets (table above) | State only; no write | Yes | Yes |
| 4.3 settings + `noindex_domains` | State only; no write | State only; no write | Yes |

## What "does not work" means in practice

| Situation | What you see |
|-----------|----------------|
| Coolify &lt; 4.1.0 | Hard error on configure |
| Resource missing on that Coolify (e.g. storage backup on 4.2) | API 404/405/422 on apply |
| Version-gated application attribute on older Coolify | Plan **warning**; attribute not in API payload; no 422 from that field alone |
| You expected the withheld field to change Coolify | It will not until you upgrade Coolify (or remove the attribute) |

There is no silent "half apply" for whole resources: missing endpoints fail.
Application settings are special because older Coolify rejects the entire
PATCH if any disallowed field is present, so the provider strips them.

## Provider version vs Coolify version

| Topic | Guidance |
|-------|----------|
| Minimum Coolify for this provider | **4.1.0** |
| Recommended Coolify for full feature set | **≥ 4.3.15** |
| Provider package versions | Current 0.1.x line targets Coolify 4.1+ with soft gates for 4.2/4.3 APIs |

Historical "0.2.x / 0.3.x min Coolify" rows in older docs referred to planned
provider lines; the published registry line remains **0.1.x** with a **4.1.0**
floor.

## How this stays accurate

Maintainers track version-dependent API fields with:

- Versioned contracts: `testdata/contracts/coolify-v4.*.json`
- `make contract-compat` and `KNOWN_VERSION_DEPENDENT` in
  `scripts/check-contract-compat.py`
- Write-key lists in `internal/client/version.go`
  (`ApplicationSettingsWriteJSONKeys`, `ApplicationSettingsV43WriteJSONKeys`)
- Daily/6-hourly Coolify channel watch (CDN + tip source + tip contract drift)

If a resource or attribute docs say `Requires Coolify >= vX.Y.Z`, that string
is the operator-facing source of truth for that field; this guide summarizes
the cross-cutting sets.

## Related guides

- [Installation](installation): install, auth, hard version check
- [Common Errors](common-errors): plan warning text and fixes
- [Troubleshooting](troubleshooting): DEBUG logs for withheld writes
- [Day-2 Operations](day-two-operations): upgrading Coolify safely
- [API Contract Accuracy](api-contract-accuracy): source-derived contract vs OpenAPI
