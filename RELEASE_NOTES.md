## Highlights

v0.1.15 adds Terraform resources for GitLab Apps, tags, shared environment variables, extra destinations, and per-server control-plane settings. Hetzner servers can attach existing networks and firewalls and enable Hetzner backups when Coolify creates the machine.

## New features

- **Coolify API parity resources.** Manage surfaces that were previously skipped: `coolify_gitlab_app`, `coolify_tag`, `coolify_resource_tag`, `coolify_shared_environment_variable`, `coolify_cloud_init_script`, `coolify_application_destination`, plus singleton server settings `coolify_server_proxy`, `coolify_server_log_drain`, `coolify_server_cloudflare_tunnel`, `coolify_server_sentinel`, and `coolify_server_docker_cleanup`. Matching data sources exist for GitLab Apps, tags, and cloud-init scripts ([#752](https://github.com/coolify-terraform/terraform-provider-coolify/pull/752)).

- **Hetzner networks, firewalls, and backups (Coolify >= v4.2.0).** `coolify_server_hetzner` can attach existing Hetzner network and firewall IDs at create time and set `enable_backups`. Look up IDs with `data.coolify_hetzner_networks` and `data.coolify_hetzner_firewalls`. These fields are create-only; Coolify does not return them on GET ([#761](https://github.com/coolify-terraform/terraform-provider-coolify/pull/761), [#767](https://github.com/coolify-terraform/terraform-provider-coolify/pull/767)).

```hcl
resource "coolify_server_hetzner" "app" {
  name                      = "app"
  cloud_provider_token_uuid = coolify_cloud_token.hetzner.uuid
  server_type               = "cx22"
  location                  = "fsn1"
  image                     = "ubuntu-24.04"
  private_key_uuid          = coolify_private_key.example.uuid

  enable_backups       = true
  hetzner_network_ids  = [456]
  hetzner_firewall_ids = [38, 39]
}
```

- **In-place environment and destination renames.** `coolify_environment` updates name and description with PATCH instead of replacing the environment. `coolify_destination` updates display name in place; changing the Docker network still forces replace ([#752](https://github.com/coolify-terraform/terraform-provider-coolify/pull/752)).

- **Tip GET fields (Coolify 4.3.3 nightly / v4.x tip).** Project and server reads expose `icon_path`, `icon_storage_type`, and `backup_compression_cpu_percentage` when Coolify returns them. They are computed only (Coolify writes them in the UI, not via public PATCH). Empty on older instances ([#758](https://github.com/coolify-terraform/terraform-provider-coolify/pull/758)).

- **Consistent container names on application data sources.** `is_consistent_container_name_enabled` is documented for apps that lock a volume (SQLite, DuckDB, and similar). Data sources now expose the same field. Requires Coolify >= v4.3.0 ([#756](https://github.com/coolify-terraform/terraform-provider-coolify/pull/756)).

## Bug fixes

- **Hetzner SSH key IDs returned 422.** `hetzner_ssh_key_ids` in HCL is still a comma-separated string. The provider now sends a JSON integer array, which is what Coolify validates ([#766](https://github.com/coolify-terraform/terraform-provider-coolify/pull/766)).

- **Shared environment variable keys with hyphens failed at apply.** Plan now rejects keys that do not match Coolify's `ENVIRONMENT_VARIABLE_KEY_PATTERN` instead of returning 422 after apply ([#757](https://github.com/coolify-terraform/terraform-provider-coolify/pull/757)).

- **Provider debug logs could print API keys and licenses.** Log redaction now also matches `api_key`, `license_key`, and `user_key` (for example Axiom, New Relic, Resend, and Pushover). Bare `_key` suffixes such as `hetzner_ssh_key_ids` are not redacted ([#769](https://github.com/coolify-terraform/terraform-provider-coolify/pull/769)).

## Documentation

- API coverage pages now say which Terraform resource or data source to use for each Coolify route, including skipped routes ([#759](https://github.com/coolify-terraform/terraform-provider-coolify/pull/759)).

- `redirect_enabled = false` and `generate_exact_labels = false` on `coolify_server_proxy` are ignored by Coolify (Laravel `has()`). Use `redirect_url` to persist a proxy change. Schema and the common-errors guide document this ([#768](https://github.com/coolify-terraform/terraform-provider-coolify/pull/768), [#769](https://github.com/coolify-terraform/terraform-provider-coolify/pull/769)).

## Compatibility

| Surface | Requires |
|---------|----------|
| Existing resources from v0.1.14 | Coolify **>= 4.1.0** (unchanged) |
| Tags, resource tags, extra destinations | Coolify **>= v4.2.0** |
| Hetzner networks, firewalls, backups, SSH key ID arrays | Coolify **>= v4.2.0** |
| GitLab Apps, shared env vars, cloud-init, server control-plane | Coolify **>= v4.3.0** |
| Project icons and backup compression CPU | Coolify tip / nightly 4.3.3 (empty on older instances) |

The provider still refuses to configure against Coolify older than 4.1.0. The pinned API contract remains Coolify **v4.3.2**.

## Numbers

| Metric | v0.1.14 | v0.1.15 |
|--------|---------|---------|
| Resources | 45 | 56 |
| Data sources | 62 | 69 |
| Tests (unit + acceptance floor) | 1330+ | 1450+ |

## Upgrading

```hcl
terraform {
  required_providers {
    coolify = {
      source  = "coolify-terraform/coolify"
      version = "~> 0.1.15"
    }
  }
}
```

No state migration is required. Existing configurations keep working. New resources are opt-in and need the Coolify version in the table above.

Docs: [Terraform Registry](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs).

## Full changelog

https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.14...v0.1.15
