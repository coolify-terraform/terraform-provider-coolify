## Highlights

You can set custom domains on an existing Coolify PR preview. On Coolify 4.3.15 and later, `coolify_application_preview` writes `domains` and `docker_compose_domains` instead of only tracking and deleting the preview.

Application restart limits are writable. First apply of `max_restart_count` now converges without a second apply.

## New features

- **Preview domains** (Coolify >= v4.3.15). `coolify_application_preview` can PATCH `domains`, `docker_compose_domains`, and `force_domain_override` on a preview Coolify already created (webhook or UI). The resource still does not create the preview ([#834](https://github.com/coolify-terraform/terraform-provider-coolify/pull/834), [#835](https://github.com/coolify-terraform/terraform-provider-coolify/pull/835)).

```hcl
# PR 42 must already have a Coolify preview deploy.
resource "coolify_application_preview" "pr_42" {
  application_uuid = coolify_application.api.uuid
  pull_request_id  = 42
  domains          = "https://pr.example.com"
}
```

- **Application `max_restart_count`.** Set the restart limit in HCL. Coolify 4.3 and later apply it on create through a follow-up PATCH. Applications also expose computed `container_present` and `restart_limit_reached`. Standalone databases expose computed restart-limit fields. Notification channels gain a `restart_limit_reached` event ([#826](https://github.com/coolify-terraform/terraform-provider-coolify/pull/826), [#828](https://github.com/coolify-terraform/terraform-provider-coolify/pull/828)).

- **Computed `domain_port_overrides`.** Coolify returns this map on GET. It is read-only in Terraform ([#832](https://github.com/coolify-terraform/terraform-provider-coolify/pull/832)).

- **Pinned API contract v4.3.17.** Recommended full-feature Coolify is v4.3.15 or later ([#834](https://github.com/coolify-terraform/terraform-provider-coolify/pull/834)).

## Bug fixes

- **Preview apply failed with Coolify 422.** Extra compose JSON keys, empty `redirect`, empty compose `[]`, or `pull_request_id` 0 are rejected at plan. A 404 means trigger a preview first. A 409 includes Coolify's warning and conflict list; set `force_domain_override` when the domain is already in use ([#836](https://github.com/coolify-terraform/terraform-provider-coolify/pull/836), [#834](https://github.com/coolify-terraform/terraform-provider-coolify/pull/834), [#835](https://github.com/coolify-terraform/terraform-provider-coolify/pull/835)).

- **`max_restart_count` needed a second apply.** The post-create PATCH now sends the field. On Coolify older than 4.3, flatten keeps your configured value instead of replacing it with GET default 10 ([#828](https://github.com/coolify-terraform/terraform-provider-coolify/pull/828)).

- **Git-backed `dockerfile` dropped after create.** The field is sent on create POST and is create-only. Changing it replaces the application ([#835](https://github.com/coolify-terraform/terraform-provider-coolify/pull/835)).

- **Failed `coolify_deployment` diagnostics dumped raw deploy logs.** Those logs are no longer attached as raw diagnostics ([#835](https://github.com/coolify-terraform/terraform-provider-coolify/pull/835)).

- **A timed-out create could retry POST and duplicate the resource.** Create and delete no longer retry on transport timeout ([#832](https://github.com/coolify-terraform/terraform-provider-coolify/pull/832)).

## Breaking changes

- **Changing `github_app_uuid` on `coolify_application_github_app` now replaces the application.** Coolify accepts that UUID only on create. A PATCH used to return 422. Existing resources with an unchanged UUID are not replaced on upgrade. After import, set `github_app_uuid` in config before apply ([#832](https://github.com/coolify-terraform/terraform-provider-coolify/pull/832)).

## Compatibility

| Surface | Requires |
|---------|----------|
| Existing resources | Coolify **>= 4.1.0** (unchanged) |
| Writable `max_restart_count` | Coolify **>= v4.3.0** |
| Preview domain writes | Coolify **>= v4.3.15** |
| Pinned API contract | Coolify **v4.3.17** |

The provider still refuses to configure against Coolify older than 4.1.0. Nightly remains `4.4-rc.1`. This release only moves the pin to current stable.

## Numbers

| Metric | v0.1.19 | v0.1.20 |
|--------|---------|---------|
| Resources | 57 | 57 |
| Data sources | 70 | 70 |
| Tests (unit + acceptance floor) | 1540+ | 1640+ |

## Upgrading

```hcl
terraform {
  required_providers {
    coolify = {
      source  = "coolify-terraform/coolify"
      version = "~> 0.1.20"
    }
  }
}
```

No state migration is required. Existing configurations keep working unless you change `github_app_uuid` on `coolify_application_github_app` (that now plans replace). Preview domain attributes are ignored on Coolify older than v4.3.15.

Docs: [Terraform Registry](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs) and [Coolify Version Support](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs/guides/coolify-version-support).

## Full changelog

https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.19...v0.1.20
