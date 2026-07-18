## What's New

v0.1.8 focuses on safer imports and cleaner application apply/import behavior.
You can set GitHub (and other) webhook secrets when creating apps without Terraform
failing, compound import rejects a wrong server before it can recreate a resource
on the wrong host, and several build fields no longer leave permanent plan noise
after import.

### Safer compound import for apps, databases, and services

Compound import IDs look like:

```bash
terraform import coolify_application.app \
  <project-uuid>:<server-uuid>:<environment>:<resource-uuid>
```

Coolify often does not return `server_uuid` on GET. A typo in the server segment
used to sit quietly in state until the next replace, which could recreate the
resource on the wrong server.

**Now:** for applications, databases, and services, compound import checks
`GET /servers/{server_uuid}/resources` and fails immediately if the resource is
not on that server. Prefer UUID-only import when you are unsure, then set
`server_uuid` in config carefully. ([#579](https://github.com/coolify-terraform/terraform-provider-coolify/pull/579), [#580](https://github.com/coolify-terraform/terraform-provider-coolify/pull/580))

### Application webhook secrets on create

Setting `manual_webhook_secret_github` (or the Bitbucket, GitLab, or Gitea
variants) used to fail create with:

```text
Error: Provider produced inconsistent result after apply
.manual_webhook_secret_github: inconsistent values for sensitive attribute
```

The provider now sends those secrets in the post-create update, and keeps your
configured value when Coolify hides the field on GET (tokens without root or
`read:sensitive`). You can set secrets on first apply. ([#579](https://github.com/coolify-terraform/terraform-provider-coolify/pull/579), [#575](https://github.com/coolify-terraform/terraform-provider-coolify/issues/575))

### Cleaner plans after application import

After importing an application, fields that were set in Coolify (for example
`base_directory`, `build_command`, `publish_directory`, `start_command`,
`watch_paths`) often stayed empty in state even when the API returned them.
That produced update noise on every plan.

Import and read now seed those fields from the API when present, so matching
config can plan cleanly. ([#579](https://github.com/coolify-terraform/terraform-provider-coolify/pull/579), [#577](https://github.com/coolify-terraform/terraform-provider-coolify/issues/577))

### Clearer docs for public vs private apps

The `coolify_application` resource page now states that it is for **public** Git
HTTPS sources only, and points private-repo users to
`coolify_application_private_git` and `coolify_application_github_app` plus the
[Which application resource do I use?](https://registry.terraform.io/providers/coolify-terraform/coolify/latest/docs/guides/choosing-application-type)
guide. The secrets guide also documents how application webhook secrets behave
on create and import. ([#579](https://github.com/coolify-terraform/terraform-provider-coolify/pull/579), [#578](https://github.com/coolify-terraform/terraform-provider-coolify/issues/578))

### Security maintenance

Provider builds use Go **1.26.5**, including the fix for
[GO-2026-5856](https://pkg.go.dev/vuln/GO-2026-5856). ([#570](https://github.com/coolify-terraform/terraform-provider-coolify/pull/570))

## Numbers

| | v0.1.7 | v0.1.8 |
|--|--------|--------|
| Resources | 33 | 33 |
| Data sources | 44 | 44 |
| Unit + acceptance tests | 970+ | 980+ |
| ACME scenarios | 17 | 17 |

## Upgrading

```hcl
terraform {
  required_providers {
    coolify = {
      source  = "coolify-terraform/coolify"
      version = "~> 0.1.8"
    }
  }
}
```

```bash
terraform init -upgrade
```

No breaking schema changes in this release. After upgrade:

1. If you use compound import, re-import only when you need to correct state;
   new imports will reject a wrong `server_uuid` segment.
2. If you previously omitted `manual_webhook_secret_*` only to avoid create
   failures, you can set them in config and apply.
3. For private Git sources, prefer `coolify_application_private_git` or
   `coolify_application_github_app` rather than `coolify_application`.

## Full changelog

https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.7...v0.1.8
