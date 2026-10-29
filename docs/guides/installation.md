---
page_title: "Installation"
subcategory: "Getting Started"
description: |-
  Install and configure the Coolify Terraform provider.
---

# Installation

## Prerequisites

| Requirement | Minimum Version |
|-------------|-----------------|
| Terraform   | 1.6+            |
| Coolify     | v4.1.0+         |

Instances older than v4.1.0 are rejected on `terraform plan` / `terraform apply`.

## Install from Terraform Registry

Add the provider to your `required_providers` block:

```hcl
terraform {
  required_providers {
    coolify = {
      source = "coolify-terraform/coolify"
    }
  }
}
```

Run `terraform init` to download and install the provider:

```bash
terraform init
```

## Configure authentication

Coolify's API is disabled by default. Enable it in the Coolify UI under **Settings** first. Then generate an API token under **Security > API Tokens**. Otherwise provider operations fail with `Unauthenticated`.

### Token permissions

For full functionality, create a token with **root** permission. Without
`root` or `read:sensitive`, the Coolify API hides sensitive fields
(Dockerfiles, compose files, private keys, database passwords) from API
responses. This causes empty state values and unexpected diffs during
`terraform plan` and `terraform import`.

| Permission | What it unlocks |
|------------|-----------------|
| `root` | Full access to all fields (recommended for Terraform) |
| `read:sensitive` | Read-only access to sensitive fields |
| Default | Sensitive fields are hidden from responses |

Set the credentials as environment variables (recommended):

```bash
export COOLIFY_ENDPOINT="https://coolify.example.com"
export COOLIFY_TOKEN="your-api-token"
```

Or configure them directly in the provider block:

```hcl
provider "coolify" {
  endpoint = "https://coolify.example.com"
  token    = "your-api-token"
}
```

## Version requirements

The provider validates the Coolify version on `terraform plan` /
`terraform apply`. If your instance is older than **v4.1.0**, the provider
will return an error and refuse to continue. Upgrade your Coolify instance
before using the provider.

### Version compatibility

| | Coolify |
|--|---------|
| **Minimum (hard floor)** | **v4.1.0** |
| **Recommended for full features** | **≥ v4.3.18** |

The API surface changed significantly between v4.0.0 and v4.1.0, so v4.1.0
is the minimum. Destinations and DO/Vultr provisioning need **≥ v4.2.0**.
Volume backup schedules, `noindex_domains`, notifications, and S3 need
**≥ v4.3.0**. Instance email settings and `smtp_ehlo_domain` need
**≥ v4.3.10**. Preview domain PATCH, GET `domain_port_overrides`, and
restart-limit fields need **≥ v4.3.15**. Missing-backup notifications on
`coolify_database_backup` need **≥ v4.3.18**. Volume-backup missing-backup
alerts need **≥ v4.4.1**. The pinned API contract is Coolify **v4.4.2**
(extract provenance), not a new feature floor.

For a full resource and attribute matrix (what works on 4.1 vs 4.2 vs 4.3 vs 4.4,
and how version gates behave), see
**[Coolify Version Support](coolify-version-support)**.

-> **Tip:** Run `data "coolify_version" "current" {}` to check your
instance version programmatically.

## Retry configuration

API requests are retried automatically on transient failures (HTTP 429,
5xx, network errors). The defaults work for most setups, but you can tune
them:

```hcl
provider "coolify" {
  retry_max      = 5   # max attempts (default: 3)
  retry_min_wait = 2   # seconds between first retries (default: 1)
  retry_max_wait = 60  # max backoff cap in seconds (default: 30)
}
```

| Attribute | Default | Description |
|-----------|---------|-------------|
| `retry_max` | 3 | Maximum number of retry attempts |
| `retry_min_wait` | 1 | Minimum wait between retries (seconds) |
| `retry_max_wait` | 30 | Maximum wait between retries (seconds) |

## Custom HTTP headers

When Coolify sits behind a proxy that authenticates requests, such as
Google Cloud Identity-Aware Proxy (IAP) or an OAuth2 proxy, the proxy needs
its own credential. `Authorization` already carries the Coolify API token,
so send the proxy's credential in another header with `headers`. The
provider adds these headers to every request it sends to the Coolify API.

For IAP, send an OpenID Connect ID token for the IAP client in
`Proxy-Authorization`. IAP checks it and removes it before the request
reaches Coolify:

```hcl
provider "coolify" {
  endpoint = "https://coolify.example.com"
  token    = var.coolify_token

  headers = {
    "Proxy-Authorization" = "Bearer ${var.iap_id_token}"
  }
}
```

Or set them as a JSON object in `COOLIFY_HEADERS`, used only when
`headers` is not set in the provider block:

```bash
export COOLIFY_HEADERS='{"Proxy-Authorization":"Bearer <id-token>"}'
```

The provider rejects `Authorization`, `Accept`, `Content-Type`,
`Content-Length`, `Host`, and `User-Agent`, which it sets itself, instead
of replacing them. `Accept` is set on requests that expect JSON, not on
the version check.

Cloudflare Access has two kinds of credential:

- A service token uses `cf_access_client_id` and `cf_access_client_secret`.
  The same two headers can go in `headers` only when those attributes are
  unset.
- A user token from `cloudflared access token` goes in `cf-access-token`.
  The `CF_Authorization` cookie can be sent as `Cookie` instead.

Pomerium can take a programmatic token in `X-Pomerium-Authorization`.
Authelia can be configured to read `Proxy-Authorization` (its
HeaderProxyAuthorization strategy). Both leave `Authorization` for the
Coolify API token.

A redirect to the same host keeps these headers. A redirect to a different
host drops them, and drops the Cloudflare Access headers, so a proxy
credential is not sent to another site.

These setups are not covered:

- The proxy requires its credential in `Authorization`. That includes HTTP
  basic auth on the proxy, oauth2-proxy's scripted JWT check
  (`--skip-jwt-bearer-tokens`), Pomerium's `Authorization: Pomerium` and
  `Authorization: Bearer Pomerium-...` forms, and Authelia's
  HeaderAuthorization strategy. A `user:password` user in the endpoint URL
  is not a second credential: the bearer token already occupies
  `Authorization`, so the URL user is ignored.
- Mutual TLS. `ca_cert` trusts the server certificate. The provider has no
  client certificate.
- Minting or refreshing a token. IAP ID tokens and Cloudflare user tokens
  expire, usually after one hour. Mint a fresh value before each run, and
  do not keep it in a `.tfvars` file.
- An HTTP forward proxy on the machine running Terraform. Use
  `HTTPS_PROXY`. `headers` are sent to the Coolify host, not to that proxy.

## Verify the connection

Create a file called `main.tf`:

```hcl
terraform {
  required_providers {
    coolify = {
      source = "coolify-terraform/coolify"
    }
  }
}

provider "coolify" {}

data "coolify_version" "current" {}

output "coolify_version" {
  value = data.coolify_version.current.version
}
```

```bash
terraform init
terraform apply
```

If the connection is successful, the output shows your Coolify instance version.

## Install from source

```bash
git clone https://github.com/coolify-terraform/terraform-provider-coolify.git
cd terraform-provider-coolify
make install
```

This compiles the provider and places it in your local Terraform plugin
cache.

## FIPS 140-3 cryptography

Release binaries include the Go native FIPS 140-3 cryptographic module
(CMVP Certificate #5247). All TLS connections the provider makes to the
Coolify API use FIPS-approved algorithms by default.

No configuration is needed. The default mode (`fips140=on`) prefers
FIPS-approved algorithms but falls back to non-approved ones when
necessary (e.g., if the Coolify server negotiates X25519 key exchange).

### Strict mode

To enforce FIPS-only algorithms (no fallbacks), set the `GODEBUG`
environment variable before running Terraform:

```bash
export GODEBUG=fips140=only
terraform plan
```

-> **Warning:** Strict mode (`fips140=only`) may break TLS connections if
the Coolify server uses X25519 key exchange, which is not FIPS-approved.
Only use strict mode when the server is configured to avoid X25519
(e.g., TLS 1.2 with RSA or ECDHE-P256 key exchange).

### Disabling FIPS

To disable FIPS entirely:

```bash
export GODEBUG=fips140=off
```

### Verifying FIPS in a binary

```bash
go version -m terraform-provider-coolify | grep fips
# Expected output includes: buildGOFIPS140=latest
```

## Next steps

Read [Core Concepts](concepts) to understand the resource model, then
follow the [Quick Start](quickstart) to deploy your first application.
