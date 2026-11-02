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
| **Recommended for every write this provider sends** | **≥ v4.5.1** |

The API surface changed significantly between v4.0.0 and v4.1.0, so v4.1.0
is the minimum. Destinations and DO/Vultr provisioning need **≥ v4.2.0**.
Volume backup schedules, `noindex_domains`, notifications, and S3 need
**≥ v4.3.0**. Instance email settings and `smtp_ehlo_domain` need
**≥ v4.3.10**. Preview domain PATCH, GET `domain_port_overrides`, and
restart-limit fields need **≥ v4.3.15**. Missing-backup notifications on
`coolify_database_backup` need **≥ v4.3.18**. Volume-backup missing-backup
alerts need **≥ v4.4.1**. Hostinger purchase needs **≥ v4.5.0**.
`use_path_style_endpoint`, `logdrain_axiom_domain`, and
`rolling_update_drain_seconds` are sent on **≥ v4.5.1**. On an older
instance those values stay in state and are not sent. Removing one of
them from configuration later does not clear a value Coolify already
stored. The pinned API contract is Coolify **v4.4.2**
(extract provenance), not a new feature floor.

For a full resource and attribute matrix (what works on 4.1, 4.2, 4.3, 4.4, and 4.5,
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
| `request_timeout` | 30 | Per-attempt HTTP timeout (seconds). Dial stays 30 seconds and the TLS handshake stays 10 seconds. A value below 30 also limits the dial. |

POST, PUT, and DELETE are not retried after a transport error, so a slow create has only `request_timeout`. Resource `timeouts` blocks are still the ceiling for the whole operation.

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

### When the proxy requires Authorization

`headers` cannot set `Authorization`. The provider writes the Coolify API
token there, and a second secret in that header would replace the token.

**oauth2-proxy** `--skip-jwt-bearer-tokens` reads a JWT only from
`Authorization`. There is no alternate header for that JWT. A Coolify
token that is not a JWT fails that check. With
`--bearer-token-login-fallback` left at its default, the failure does not
return 403, and a session cookie can still authenticate. Send that cookie
with `headers` as `Cookie`. `--skip-auth-route` turns the proxy check off
for a path. That is not a second credential. Leave
`--pass-authorization-header` off, because it writes a bearer token on the
upstream request and replaces the Coolify token. `--pass-basic-auth`
defaults to true and can also pass basic auth upstream. Confirm the
upstream `Authorization` value is still the Coolify bearer.

**Pomerium.** Send the programmatic token in `X-Pomerium-Authorization`.
Do not use `Authorization: Pomerium` or `Authorization: Bearer Pomerium-...`.
A route that sets `Authorization` on the upstream request removes the
Coolify token.

**Authelia.** Keep Bearer out of a HeaderAuthorization scheme list. A
Bearer value that is not in that list is skipped, not rejected. Send the
Authelia credential in `Proxy-Authorization`. The proxy-authorization
strategy answers 407, which suits forward-auth. The AuthRequest
implementation also reads `Proxy-Authorization` and answers 401, which is
what nginx `auth_request` expects.

**HTTP basic auth on the reverse proxy** (nginx `auth_basic`, Caddy
`basic_auth`, Traefik `basicAuth`) is not possible. Those checks read
`Authorization`. A user in the endpoint URL is ignored for the same
reason. Mutual TLS is a different requirement. It does not satisfy basic
auth.

### Mutual TLS

`ca_cert` trusts the Coolify server. `client_cert` and `client_key` are
the certificate the provider presents. Set both, or neither. The key must
be an unencrypted PKCS#8 or PKCS#1 PEM. `file()` can read the files.

```hcl
provider "coolify" {
  endpoint    = "https://coolify.example.com"
  token       = var.coolify_token
  ca_cert     = file("${path.module}/ca.pem")
  client_cert = file("${path.module}/client.pem")
  client_key  = file("${path.module}/client.key")
}
```

### TLS server name

`tls_server_name` is the name in the certificate and the TLS server name
(SNI). The HTTP Host header stays the host from `endpoint`. If a router
selects the site from that header, put the DNS name in `endpoint` and
resolve it with split DNS or a hosts entry. `insecure` skips verification.
It does not fix a 404 from the wrong host.

```hcl
provider "coolify" {
  endpoint        = "https://203.0.113.10"
  token           = var.coolify_token
  tls_server_name = "coolify.internal.example"
}
```

### Minting a header

On Terraform 1.10 or later, an ephemeral value can be used in `headers`.
It is left out of state and plan files. The provider still requires
Terraform 1.6, so that path is not available on every supported release,
and it does not run a CLI.

`header_command` runs a program once per Terraform command (plan and apply
each run it) and sends the stdout as one header. It is not refreshed in
the middle of apply. A run longer than the token lifetime still fails.
The command runs on the machine that executes Terraform, including an HCP
Terraform agent. Do not put the secret in an argument. Stdout is not
written to error messages.

```hcl
provider "coolify" {
  endpoint = "https://coolify.example.com"
  token    = var.coolify_token

  header_command {
    name    = "Proxy-Authorization"
    command = ["gcloud", "auth", "print-identity-token", "--audiences=AUDIENCE"]
    prefix  = "Bearer "
  }
}
```

### Outbound proxy

`HTTPS_PROXY`, `HTTP_PROXY`, and `NO_PROXY` apply to every API request,
including requests that set `ca_cert` or `insecure`. Localhost and
loopback addresses are never sent through that proxy.

`headers` are sent to the Coolify host. They are not the credential for
the proxy hop in front of the runner. Set that hop with the `proxy` block.
`http`, `socks5`, and `socks5h` are supported. `https` proxy URLs are
rejected. NTLM and Kerberos are not supported. When `url` is set it
replaces `HTTP_PROXY` and `HTTPS_PROXY`. An omitted `no_proxy` still
honors `NO_PROXY`.

```hcl
provider "coolify" {
  endpoint = "https://coolify.example.com"
  token    = var.coolify_token

  proxy {
    url      = "http://proxy.example:8080"
    username = "ci"
    password = var.proxy_password
    no_proxy = "localhost,127.0.0.1"
  }
}
```

An `http://` Coolify endpoint sends proxy basic auth on the same request
as the API call, which replaces a `Proxy-Authorization` header. Use
`https://` so that header stays on the tunneled request.

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
