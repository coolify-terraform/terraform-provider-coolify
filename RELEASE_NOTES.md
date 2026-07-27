## What's New

v0.1.9 expands Coolify cloud provisioning and operations coverage for the
Coolify 4.2 line, while remaining compatible with Coolify 4.1.x installs.
You can provision DigitalOcean and Vultr servers through Coolify, manage
Docker network destinations, schedule volume backups when your Coolify
build includes that API, and rely on several correctness fixes for actions,
schedules, and nginx config round-trips.

Minimum Coolify version is still **4.1.0**. New resources that need a newer
API are called out below and in each resource doc.

### DigitalOcean and Vultr servers (Coolify >= v4.2.0)

Provision cloud VMs through Coolify with the same patterns as Hetzner:

- `coolify_server_digitalocean`
- `coolify_server_vultr`

Pair them with lookup data sources for regions, sizes/plans, images/OS, and
SSH keys (`coolify_digitalocean_*`, `coolify_vultr_*`). Create-only cloud
fields are documented on import: set them in config after import so Terraform
does not force replace. ([#589](https://github.com/coolify-terraform/terraform-provider-coolify/pull/589))

### Docker destinations (Coolify >= v4.2.0)

`coolify_destination` manages Docker network destinations (standalone or
swarm) via Terraform, with singular and plural data sources for lookup.
([#589](https://github.com/coolify-terraform/terraform-provider-coolify/pull/589),
[#598](https://github.com/coolify-terraform/terraform-provider-coolify/pull/598))

### Storage volume backup schedules

`coolify_storage_backup` schedules backups for persistent or directory
storage on applications, databases, or services.

Coolify exposes **PUT** and **DELETE** only (no GET for the schedule). The
provider keeps schedule attributes from state on refresh and checks that the
parent storage still exists.

**Coolify version note:** this API landed on Coolify branch `v4.x` after
[coollabsio/coolify#10946](https://github.com/coollabsio/coolify/pull/10946).
It is **not** in git tag `v4.2.0` or stable CDN `4.1.2`. Use a Coolify image
built from `v4.x` after that merge (CDN nightly may still report `4.2.0`
while tip commits differ). Docs and examples state the same floor.
([#601](https://github.com/coolify-terraform/terraform-provider-coolify/pull/601),
[#602](https://github.com/coolify-terraform/terraform-provider-coolify/pull/602))

### Lifecycle actions use POST

Coolify changed start/stop/restart and server validate endpoints from GET to
POST. Calling the old method returned **405** (`This endpoint has changed to
a POST request`). The provider now uses POST for application, database, and
service start/stop/restart, and for server validate, so
`coolify_resource_action` and related flows work against current Coolify.
([#587](https://github.com/coolify-terraform/terraform-provider-coolify/pull/587))

### Human-friendly backup and task schedules

Coolify accepts bare schedule names such as `daily`, `hourly`, and
`every_minute` (not only `@daily`). Terraform plan used to reject those
strings before any API call. Frequency validation now matches Coolify for
database backups, storage backups, scheduled tasks, and server disk-check
cron fields. ([#603](https://github.com/coolify-terraform/terraform-provider-coolify/pull/603))

### Custom nginx configuration stays stable across plans

`custom_nginx_configuration` is base64 on the wire, like `custom_labels`.
After apply, Read could store the API base64 form while config kept your raw
text, which produced endless plan diffs. Read now preserves equivalent raw
values the same way labels do.
([#604](https://github.com/coolify-terraform/terraform-provider-coolify/pull/604))

### Clearer create failures on empty UUIDs

If Coolify returns success without a resource UUID (for example when a
server is not actually usable), create now fails with an explicit empty-UUID
error instead of continuing with a blank id.
([#598](https://github.com/coolify-terraform/terraform-provider-coolify/pull/598))

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

No state migration is required for existing resources.

**Coolify version guidance:**

- Staying on Coolify **stable 4.1.x**: safe for resources you already use.
  Do not adopt DigitalOcean/Vultr servers, destinations, or volume backups
  until your instance has the matching API.
- Coolify **>= v4.2.0** (tag): DigitalOcean/Vultr servers and destinations.
- Volume backups: Coolify `v4.x` **after** #10946 (not the `v4.2.0` tag alone).

Registry docs: https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs
