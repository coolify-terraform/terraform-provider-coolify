## Highlights

v0.1.13 lets you pin which Docker network a resource uses when a Coolify
server has more than one destination. Set `destination_uuid` on applications,
databases, and services at create time instead of relying only on heuristics.

## New features

- **`destination_uuid` on applications and databases.** Optional create-only
  attribute on every application type and every database resource. Pass a
  `coolify_destination` UUID when the server has multiple networks. When you
  omit it, the provider still auto-resolves after Coolify returns the
  multi-destination error (prefers network name `coolify`). Changing the
  value forces replacement; Coolify does not allow updates to this field.
  Import cannot recover the UUID (GET returns destination morph IDs only) ([#678](https://github.com/coolify-terraform/terraform-provider-coolify/pull/678)).

- **`destination_uuid` on `coolify_service`.** Same create-only pin for catalog
  and custom Docker Compose services, with the same multi-destination
  auto-retry as applications and databases ([#680](https://github.com/coolify-terraform/terraform-provider-coolify/pull/680)).

```hcl
resource "coolify_destination" "app_net" {
  server_uuid = coolify_server.main.uuid
  network     = "coolify-app"
  name        = "App network"
  type        = "standalone"
}

resource "coolify_application" "api" {
  name             = "api"
  project_uuid     = coolify_project.main.uuid
  server_uuid      = coolify_server.main.uuid
  destination_uuid = coolify_destination.app_net.uuid
  git_repository   = "https://github.com/example/api"
  git_branch       = "main"
  build_pack       = "nixpacks"
  ports_exposes    = "3000"
}
```

## Bug fixes and reliability

- **Clearer bulk env var errors.** `coolify_envs_bulk` Create, Read, Update,
  and Import diagnostics include the parent resource type and UUID so multi-
  resource applies are easier to attribute ([#676](https://github.com/coolify-terraform/terraform-provider-coolify/pull/676)).

- **Database create multi-destination retry coverage.** Client unit tests lock
  the error-driven destination resolve path for database create (applications
  already had proactive resolve tests) ([#676](https://github.com/coolify-terraform/terraform-provider-coolify/pull/676)).

## Documentation

- Common Errors and Connecting Resources guides cover multi-destination
  create and `destination_uuid`. Examples show optional `destination_uuid`
  comments on applications, services, and PostgreSQL ([#681](https://github.com/coolify-terraform/terraform-provider-coolify/pull/681)).

## Compatibility

| Surface | Requires |
|---------|----------|
| Existing resources without `destination_uuid` | Coolify **>= 4.1.0** (unchanged) |
| Managing destinations with `coolify_destination` | Coolify **>= v4.2.0** |
| Explicit `destination_uuid` on create | Field accepted on create for supported Coolify versions; multi-destination servers are a Coolify **>= v4.2.0** surface |

Omitting `destination_uuid` keeps prior behavior: single-destination servers
need no change; multi-destination servers still use auto-resolve when Coolify
returns the multi-destination error.

## Numbers

| Metric | This release |
|--------|----------------|
| Resources | 37 |
| Data sources | 54 |
| Tests (unit + acceptance floor) | 1150+ |

## Upgrading

```hcl
terraform {
  required_providers {
    coolify = {
      source  = "coolify-terraform/coolify"
      version = "~> 0.1.13"
    }
  }
}
```

No state migration is required. Existing configs without `destination_uuid`
continue to work. After import of a multi-destination resource, re-adding
`destination_uuid` in config will force replacement unless you set state
manually or leave the attribute unset.

Docs: [Common Errors](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs/guides/common-errors),
[Connecting Resources](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs/guides/connecting-resources).

## Full changelog

https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.12...v0.1.13
