## Highlights

v0.1.12 makes Coolify **4.1.x** and **4.2.x** safer to drive with the same
Terraform configs. Application create no longer fails with a 422 when you set
settings that only Coolify 4.2 accepts, and plan warns instead of surprising you
at apply. Docker Compose domain attributes also stop perpetual plan diffs when
Coolify returns object-shaped JSON and you wrote an array.

## Bug fixes

- **Create failed with 422 on Coolify 4.1.x when application settings were set.**
  Fields such as `is_gzip_enabled`, git submodule/LFS options, and related
  settings are only on Coolify **>= v4.2.0** allow lists. On older instances the
  provider now withholds those fields on PATCH (including post-create), so Create
  no longer aborts and leaves the resource tainted. Values remain in Terraform
  state until you upgrade Coolify or remove the attributes ([#662](https://github.com/coolify-terraform/terraform-provider-coolify/pull/662), [#665](https://github.com/coolify-terraform/terraform-provider-coolify/pull/665)).

- **Perpetual plan diff on `docker_compose_domains`.** Coolify stores the field as
  a JSON array of `{name, domain}` objects, but GET sometimes returns a
  name-keyed object. The provider normalizes both shapes, wires PATCH as a JSON
  array (not a JSON string), and preserves an equivalent user form so plan stays
  clean after apply ([#658](https://github.com/coolify-terraform/terraform-provider-coolify/pull/658)).

- **Docs for compose domains ordering.** Guides and attribute docs spell out the
  Coolify rule that `docker_compose_raw` must exist before
  `docker_compose_domains`, and recommend a two-stage apply when needed ([#656](https://github.com/coolify-terraform/terraform-provider-coolify/pull/656)).

## New features

- **Plan-time warning for 4.2-only application writes on older Coolify.** When the
  connected instance is below v4.2.0 and config sets gated settings, Terraform
  plan/apply emits a clear warning listing the attributes. The provider still
  keeps them in state and does not send them to the API. Warnings use your
  configuration (not post-apply defaults), so omitted fields do not false-fire ([#664](https://github.com/coolify-terraform/terraform-provider-coolify/pull/664)).

## Compatibility

| Surface | Requirement |
|---------|-------------|
| Existing resources | Still work on Coolify **>= 4.1.0** |
| Application settings listed in the plan warning | Write to API only on Coolify **>= v4.2.0** |
| Provider withhold + warn behavior | Coolify **4.1.x** (and any instance below 4.2.0) |

No Terraform state migration is required.

## Upgrading

```hcl
terraform {
  required_providers {
    coolify = {
      source  = "coolify-terraform/coolify"
      version = "~> 0.1.12"
    }
  }
}
```

Then run `terraform init -upgrade`. On Coolify 4.1.x, expect plan warnings (not
hard errors) if you set 4.2-only application settings; upgrade Coolify or drop
those attributes when you want the API to apply them. For compose apps, follow
the [Common Errors](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs/guides/common-errors)
and [Docker Compose](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs/guides/docker-compose-services)
guides if you set `docker_compose_domains`.

## Contributors

Thanks to external contributors in this release:

- [@pbrissaud](https://github.com/pbrissaud) for reporting Coolify 4.1.x Create
  422s from application settings and the initial version-gate approach that
  landed in the polished fix ([#662](https://github.com/coolify-terraform/terraform-provider-coolify/pull/662))

## Full changelog

https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.11...v0.1.12
