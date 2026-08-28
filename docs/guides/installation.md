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
| **Recommended for full features** | **≥ v4.3.14** |

The API surface changed significantly between v4.0.0 and v4.1.0, so v4.1.0
is the minimum. Destinations and DO/Vultr provisioning need **≥ v4.2.0**.
Volume backup schedules and some application settings need **≥ v4.3.0**.

For a full resource and attribute matrix (what works on 4.1 vs 4.2 vs 4.3,
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
