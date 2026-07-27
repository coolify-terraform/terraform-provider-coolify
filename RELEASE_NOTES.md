## Highlights

v0.1.9 adds cloud servers and destinations for Coolify 4.2, plus volume
backup schedules where your Coolify build has that API. It also fixes
start/stop (POST), schedule strings like `daily`, and perpetual diffs on
custom nginx config.

Minimum Coolify is still **4.1.0**. New surfaces that need a newer API are
listed under Compatibility.

## New features

- **`coolify_server_digitalocean` and `coolify_server_vultr`** (Coolify >=
  v4.2.0). Provision DO/Vultr VMs through Coolify, like Hetzner. Lookup data
  sources cover regions, sizes/plans, images/OS, and SSH keys. Create-only
  cloud fields are empty after import; set them in config before plan.
  ([#589](https://github.com/coolify-terraform/terraform-provider-coolify/pull/589))

- **`coolify_destination`** (Coolify >= v4.2.0). Manage Docker network
  destinations (standalone or swarm), with singular and list data sources.
  ([#589](https://github.com/coolify-terraform/terraform-provider-coolify/pull/589),
  [#598](https://github.com/coolify-terraform/terraform-provider-coolify/pull/598))

- **`coolify_storage_backup`**. Schedule backups for app, database, or
  service volumes. Coolify only offers PUT/DELETE (no schedule GET); the
  provider keeps schedule fields from state and checks the parent storage
  still exists. **Requires Coolify `v4.x` after
  [coollabsio/coolify#10946](https://github.com/coollabsio/coolify/pull/10946)**
  (not git tag `v4.2.0` alone; not stable CDN 4.1.2). See the resource doc
  for tip vs tag detail.
  ([#601](https://github.com/coolify-terraform/terraform-provider-coolify/pull/601),
  [#602](https://github.com/coolify-terraform/terraform-provider-coolify/pull/602))

Example (DigitalOcean server):

```hcl
resource "coolify_server_digitalocean" "app" {
  name                      = "do-app-node"
  cloud_provider_token_uuid = coolify_cloud_token.do.uuid
  region                    = "nyc1"
  size                      = "s-1vcpu-1gb"
  image                     = "ubuntu-24-04-x64"
  private_key_uuid          = coolify_private_key.main.uuid
}
```

## Bug fixes

- **Start/stop/restart and server validate returned 405.** Coolify switched
  those endpoints to POST (`This endpoint has changed to a POST request`).
  The provider now uses POST for application, database, and service lifecycle
  actions and server validate, so `coolify_resource_action` works on current
  Coolify.
  ([#587](https://github.com/coolify-terraform/terraform-provider-coolify/pull/587))

- **Plan rejected valid schedules like `daily` or `hourly`.** Coolify accepts
  bare human names (not only `@daily`). Frequency validation matches that for
  database backups, storage backups, scheduled tasks, and server disk-check
  fields.
  ([#603](https://github.com/coolify-terraform/terraform-provider-coolify/pull/603))

- **Endless plan diffs on `custom_nginx_configuration`.** Coolify stores
  base64; config often has raw nginx text. Read now preserves equivalent raw
  values (same approach as `custom_labels`).
  ([#604](https://github.com/coolify-terraform/terraform-provider-coolify/pull/604))

- **Create could continue with an empty UUID** when Coolify returned success
  without an id (for example a server that was not actually usable). Create
  now fails with a clear empty-UUID error.
  ([#598](https://github.com/coolify-terraform/terraform-provider-coolify/pull/598))

## Compatibility

| Surface | Coolify requirement |
|---------|---------------------|
| Existing resources from v0.1.8 | Still work on **>= 4.1.0** (including stable CDN 4.1.2) |
| DigitalOcean / Vultr servers, destinations | **>= v4.2.0** (git tag / API line) |
| `coolify_storage_backup` | `v4.x` **after** [coolify#10946](https://github.com/coollabsio/coolify/pull/10946); not tag `v4.2.0` alone |

## Numbers

| | v0.1.8 | v0.1.9 |
|--|--------|--------|
| Resources | 33 | 37 |
| Data sources | 44 | 54 |
| Unit + acceptance tests | 980+ | 1060+ |
| ACME scenarios | 17 | 17 |
| Minimum Coolify | 4.1.0 | 4.1.0 |

## Upgrading

```hcl
terraform {
  required_providers {
    coolify = {
      source  = "coolify-terraform/coolify"
      version = "~> 0.1.9"
    }
  }
}
```

```bash
terraform init -upgrade
```

No state migration is required for existing resources. Stay on Coolify
4.1.x unless you adopt the Compatibility table rows that need 4.2 or tip.

Docs: https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs

## Full changelog

https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.8...v0.1.9
