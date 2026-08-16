## Highlights

The provider now tracks Coolify 4.3.5. There are no new public API fields versus 4.3.3. This release also keeps `redirect_url` on `coolify_server_proxy` after import when the proxy type did not change.

## Bug fixes

- **Imported `coolify_server_proxy` lost `redirect_url`.** Coolify resets the URL whenever `proxy_type` is sent, even if it is already current. The provider no longer re-sends an unchanged type ([#780](https://github.com/coolify-terraform/terraform-provider-coolify/pull/780)).

## Compatibility

| Surface | Requires |
|---------|----------|
| Existing resources | Coolify **>= 4.1.0** (unchanged) |
| Pinned API contract | Coolify **v4.3.5** (same public fields as 4.3.3 and 4.3.4) |

Coolify 4.3 instance SSO (Authentik, Zitadel, Clerk) is configured in the Coolify UI. Those login providers have no public API routes, so this provider cannot manage them.

The provider still refuses to configure against Coolify older than 4.1.0.

## Numbers

| Metric | v0.1.15 | v0.1.16 |
|--------|---------|---------|
| Resources | 56 | 56 |
| Data sources | 69 | 69 |
| Tests (unit + acceptance floor) | 1450+ | 1450+ |

## Upgrading

```hcl
terraform {
  required_providers {
    coolify = {
      source  = "coolify-terraform/coolify"
      version = "~> 0.1.16"
    }
  }
}
```

No state migration is required. Existing configurations keep working.

Docs: [Terraform Registry](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs) and [Coolify Version Support](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs/guides/coolify-version-support).

## Full changelog

https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.15...v0.1.16
