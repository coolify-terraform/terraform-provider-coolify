## Highlights

First apply can now bring a new application up with `coolify_deployment`. Docker image updates and database create with extra fields no longer fail against Coolify.

## Bug fixes

- **First apply of `coolify_deployment` did not start a new app.** Create used Coolify restart, which skips or returns no UUID when the app has never run. Create now queues a deploy, and adopts an already-queued job when `instant_deploy` is true ([#785](https://github.com/coolify-terraform/terraform-provider-coolify/pull/785)).

- **Changing `docker_image` on update returned 422.** Coolify update accepts the repository in `docker_registry_image_name` and the tag in `docker_registry_image_tag`. The provider now splits `image:tag` on update and keeps an explicit `docker_registry_image_tag` when you set one ([#784](https://github.com/coolify-terraform/terraform-provider-coolify/pull/784), [#788](https://github.com/coolify-terraform/terraform-provider-coolify/pull/788)).

- **Creating a database with `limits_memory` (or other extended fields) failed and tainted the resource.** Coolify create succeeded, then the follow-up PATCH sent keys Coolify rejects (`is_log_drain_enabled`, `is_include_timestamps`, `enable_ssl`). Terraform reported an invalid result object. Those keys are omitted on create PATCH ([#790](https://github.com/coolify-terraform/terraform-provider-coolify/pull/790)).

## Compatibility

| Surface | Requires |
|---------|----------|
| Existing resources | Coolify **>= 4.1.0** (unchanged) |
| Pinned API contract | Coolify **v4.3.9** (same public write surface as 4.3.5 for this provider) |

The provider still refuses to configure against Coolify older than 4.1.0.

## Numbers

| Metric | v0.1.16 | v0.1.17 |
|--------|---------|---------|
| Resources | 56 | 56 |
| Data sources | 69 | 69 |
| Tests (unit + acceptance floor) | 1450+ | 1480+ |

## Upgrading

```hcl
terraform {
  required_providers {
    coolify = {
      source  = "coolify-terraform/coolify"
      version = "~> 0.1.17"
    }
  }
}
```

No state migration is required. Existing configurations keep working.

Docs: [Terraform Registry](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs) and [Coolify Version Support](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs/guides/coolify-version-support).

## Contributors

Thanks to external contributors in this release:

- [@mheers](https://github.com/mheers) for splitting docker image name and tag on update so Coolify no longer 422s a tagged image ([#784](https://github.com/coolify-terraform/terraform-provider-coolify/pull/784))

## Full changelog

https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.16...v0.1.17
