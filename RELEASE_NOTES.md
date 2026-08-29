## Highlights

Service `urls` and application `noindex_domains` keep the order you wrote in HCL. Coolify can return those lists in a different order; apply no longer fails with "Provider produced inconsistent result after apply."

The pinned Coolify API contract is now v4.3.14.

## Bug fixes

- **Provider produced inconsistent result after apply on `coolify_service.urls`.** Coolify `GET /services/{uuid}` can return applications in a different order than HCL (often alphabetical). The provider now matches each URL by container name and keeps your configured list order. Comma-separated FQDNs with the same URLs (after trim and case fold) are treated as the same ([#819](https://github.com/coolify-terraform/terraform-provider-coolify/pull/819)).

- **`noindex_domains` copied Coolify GET order.** Configured order and casing are kept when the URL set matches. On Coolify older than 4.3, GET may omit the field; flatten keeps the configured list instead of writing empty, which used to leave a perpetual plan. `data.coolify_application` now exposes computed `noindex_domains` ([#820](https://github.com/coolify-terraform/terraform-provider-coolify/pull/820), [#821](https://github.com/coolify-terraform/terraform-provider-coolify/pull/821)).

## Compatibility

| Surface | Requires |
|---------|----------|
| Existing resources | Coolify **>= 4.1.0** (unchanged) |
| Instance email and `smtp_ehlo_domain` | Coolify **>= v4.3.10** |
| Pinned API contract | Coolify **v4.3.14** |

The provider still refuses to configure against Coolify older than 4.1.0. Nightly remains `4.4-rc.1`; this release only moves the pin to current stable.

## Numbers

| Metric | v0.1.18 | v0.1.19 |
|--------|---------|---------|
| Resources | 57 | 57 |
| Data sources | 70 | 70 |
| Tests (unit + acceptance floor) | 1510+ | 1540+ |

## Upgrading

```hcl
terraform {
  required_providers {
    coolify = {
      source  = "coolify-terraform/coolify"
      version = "~> 0.1.19"
    }
  }
}
```

No state migration is required. Existing configurations keep working. If apply previously failed on `coolify_service.urls` because Coolify reordered applications, upgrade and apply again; you do not need to reorder HCL.

Docs: [Terraform Registry](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs) and [Coolify Version Support](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs/guides/coolify-version-support).

## Contributors

Thanks to external contributors in this release:

- [@arthurjarvis02](https://github.com/arthurjarvis02) for reporting service `urls` order causing inconsistent result after apply ([#818](https://github.com/coolify-terraform/terraform-provider-coolify/issues/818))

## Full changelog

https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.18...v0.1.19
