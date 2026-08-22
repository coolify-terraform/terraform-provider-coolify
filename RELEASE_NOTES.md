## Highlights

You can manage Coolify instance-wide SMTP and Resend settings in Terraform. Team email notifications can send the hostname Coolify uses on SMTP EHLO.

## New features

- **`coolify_instance_email_settings`** (Coolify >= v4.3.10, root-team API token).
  Resource and data source for `GET`/`PATCH /settings/email`. Use this for the instance singleton; team notifications can inherit it with `use_instance_email_settings = true` ([#802](https://github.com/coolify-terraform/terraform-provider-coolify/pull/802), [#804](https://github.com/coolify-terraform/terraform-provider-coolify/pull/804)).

```hcl
resource "coolify_instance_email_settings" "main" {
  smtp_enabled      = true
  smtp_host         = "smtp.example.com"
  smtp_port         = 587
  smtp_encryption   = "starttls"
  smtp_from_address = "alerts@example.com"
  smtp_from_name    = "Coolify"
  smtp_username     = "smtp-user"
  smtp_password     = "change-me-in-production"
  smtp_ehlo_domain  = "mail.example.com"
}
```

- **`smtp_ehlo_domain` on `coolify_notification_email`** (Coolify >= v4.3.10).
  Sets the hostname sent with SMTP EHLO. Older Coolify instances omit the field ([#800](https://github.com/coolify-terraform/terraform-provider-coolify/pull/800)).

## Bug fixes

- **Invalid SMTP From addresses were accepted at plan time.** `smtp_from_address` on instance and team email now requires a single mailbox address (for example `alerts@example.com`), not a display-name form ([#805](https://github.com/coolify-terraform/terraform-provider-coolify/pull/805)).

## Compatibility

| Surface | Requires |
|---------|----------|
| Existing resources | Coolify **>= 4.1.0** (unchanged) |
| `coolify_instance_email_settings` | Coolify **>= v4.3.10** and a root-team token |
| `smtp_ehlo_domain` | Coolify **>= v4.3.10** |
| Pinned API contract | Coolify **v4.3.10** |

The provider still refuses to configure against Coolify older than 4.1.0.

## Numbers

| Metric | v0.1.17 | v0.1.18 |
|--------|---------|---------|
| Resources | 56 | 57 |
| Data sources | 69 | 70 |
| Tests (unit + acceptance floor) | 1480+ | 1510+ |

## Upgrading

```hcl
terraform {
  required_providers {
    coolify = {
      source  = "coolify-terraform/coolify"
      version = "~> 0.1.18"
    }
  }
}
```

No state migration is required. Existing configurations keep working. Instance email and `smtp_ehlo_domain` stay unused until you add them.

Docs: [Terraform Registry](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs) and [Coolify Version Support](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs/guides/coolify-version-support).

## Full changelog

https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.17...v0.1.18
