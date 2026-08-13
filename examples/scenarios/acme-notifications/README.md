# ACME Corp: Team Notification Channels

Configures Coolify **team-scoped** notification settings for Discord, email
(SMTP), and a generic webhook. Channels default to **disabled** so
`terraform apply` does not send real traffic.

> **Caution:** Apply replaces the API token's current team notification
> settings (webhook URLs, SMTP fields, event flags). Do not run against a
> production Coolify team that already has live channels unless you intend
> to overwrite them.

## Prerequisites

- Coolify **>= v4.3.0** (notification API routes)
- API token with permission to manage team notification settings

## What this demonstrates

| Resource | Pattern |
|----------|---------|
| `coolify_notification_discord` | Team singleton, webhook URL, event flags |
| `coolify_notification_email` | SMTP settings + event flags |
| `coolify_notification_webhook` | Generic webhook channel |

### Import

There is no UUID. Import with literal id `current`:

```bash
terraform import coolify_notification_discord.alerts current
terraform import coolify_notification_email.ops current
terraform import coolify_notification_webhook.hooks current
```

### Destroy behavior

Destroy **disables** the channel (`enabled = false` / `smtp_enabled = false`).
It does **not** delete Coolify-side webhook URLs or SMTP credentials. Keep
secrets in configuration if you re-import.

## Usage

```bash
export TF_VAR_coolify_endpoint="https://your-coolify.example.com"
export TF_VAR_coolify_token="your-token"

terraform init
terraform apply
# or: terraform test
```

## Related

- [Import guide](../../../docs/guides/import.md) (team singleton section)
- Resource docs under `docs/resources/notification_*.md`
