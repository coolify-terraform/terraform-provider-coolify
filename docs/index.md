---
page_title: "Coolify Provider"
subcategory: ""
description: |-
  Manage your Coolify infrastructure as code: applications, databases, servers, backups, and environment variables, all version-controlled and reproducible.
---

# Coolify Provider

Define your entire [Coolify](https://coolify.io/) infrastructure as code. Deploy applications, provision databases, configure backups, and manage environment variables, all version-controlled and reproducible with `terraform apply`.

Coolify gives you a self-hosted PaaS with a great UI. This provider adds what the UI cannot: reproducible environments, pull-request-based infrastructure reviews, disaster recovery from state, and CI/CD integration. Whether you run a single VPS or manage Coolify for a team, Terraform turns your infrastructure into a reviewable, testable, git-tracked configuration.

**61 resources, 78 data sources.** Route coverage is tracked against the Coolify source contract (not upstream OpenAPI).

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.6
- A running [Coolify](https://coolify.io/) instance (v4.1.0+)

## Authentication

The provider requires a Coolify API token.

~> **The Coolify API is disabled by default.** You must enable it in the
Coolify dashboard under **Settings** before generating a token. Without
this, all provider operations fail with "Unauthenticated."

Generate a token in the Coolify UI under **Security > API Tokens**.

You can configure credentials in two ways:

### Environment variables (recommended for CI/CD)

```bash
export COOLIFY_ENDPOINT="https://coolify.example.com"
export COOLIFY_TOKEN="your-api-token"

# Optional: Cloudflare Access (when Coolify is behind CF Access)
export COOLIFY_CF_ACCESS_CLIENT_ID="your-cf-client-id"
export COOLIFY_CF_ACCESS_CLIENT_SECRET="your-cf-client-secret"

# Optional: extra headers for an authenticating proxy (JSON object)
export COOLIFY_HEADERS='{"Proxy-Authorization":"Bearer your-proxy-token"}'
```

### Provider block

```terraform
provider "coolify" {
  endpoint = var.coolify_endpoint # Or set COOLIFY_ENDPOINT env var
  token    = var.coolify_token    # Or set COOLIFY_TOKEN env var

  # Optional: Cloudflare Access (when Coolify is behind CF Access)
  # cf_access_client_id     = var.cf_access_client_id     # Or set COOLIFY_CF_ACCESS_CLIENT_ID
  # cf_access_client_secret = var.cf_access_client_secret  # Or set COOLIFY_CF_ACCESS_CLIENT_SECRET

  # Optional: extra headers for an authenticating proxy, such as Google Cloud IAP.
  # Or set COOLIFY_HEADERS to a JSON object.
  # headers = {
  #   "Proxy-Authorization" = "Bearer ${var.iap_id_token}"
  # }

  # Optional: per-attempt HTTP timeout in seconds (default 30).
  # request_timeout = 60

  # Optional: mutual TLS. Set both, or neither.
  # client_cert = file("${path.module}/client.pem")
  # client_key  = file("${path.module}/client.key")

  # Optional: certificate name when it differs from the endpoint host.
  # tls_server_name = "coolify.internal.example"

  # Optional: outbound proxy for this machine. Not the proxy in front of Coolify.
  # proxy {
  #   url      = "http://proxy.example:8080"
  #   username = "ci"
  #   password = var.proxy_password
  # }
}
```

## Getting Started

- **[Quick Start](guides/quickstart)** - Deploy your first application in under 5 minutes
- **[Installation](guides/installation)** - Detailed setup and configuration options
- **[Coolify Version Support](guides/coolify-version-support)** - What works on Coolify 4.1, 4.2, 4.3, and 4.4
- **[API Coverage](https://github.com/coolify-terraform/terraform-provider-coolify/blob/main/API_COVERAGE.md)** - Coolify HTTP routes the provider wraps, and what to use when it does not
- **[Concepts](guides/concepts)** - How Coolify resources map to Terraform
- **[Choosing an Application Type](guides/choosing-application-type)** - Pick the right resource for your deployment method
- **[Connecting Resources](guides/connecting-resources)** - Wire apps to databases using Coolify's Docker networking
- **[Domains and HTTPS](guides/domains-and-https)** - Custom domains, automatic TLS, and URL redirects
- **[Build Lifecycle](guides/build-lifecycle)** - What happens after apply: build queuing, status, and deployment waiting
- **[Secrets Management](guides/secrets-management)** - Handle passwords, tokens, and keys safely
- **[Service Catalog](guides/service-catalog)** - Deploy one-click services (Plausible, Grafana, MinIO, etc.)
- **[CI/CD Integration](guides/cicd-integration)** - Automate deployments with GitHub Actions or GitLab CI
- **[Day-2 Operations](guides/day-two-operations)** - Upgrades, rollbacks, and disaster recovery
- **[Import Guide](guides/import)** - Bring existing Coolify resources under Terraform management
- **[Common Errors](guides/common-errors)** - Error message reference with causes and fixes
- **[Troubleshooting](guides/troubleshooting)** - Debugging tips and diagnostic logs
- **[Scenario Examples](guides/scenario-testing)** - Real-world ACME Corp deployment scenarios
- **[Migration from Community Provider](guides/migration-from-community)** - Migrate from SierraJC/coolify
- **[API Contract Accuracy](guides/api-contract-accuracy)** - OpenAPI spec vs real Coolify source code

## Resource Overview

| Category | Resources |
|----------|-----------|
| Infrastructure | `coolify_project`, `coolify_environment`, `coolify_server`, `coolify_server_hetzner`, `coolify_server_hostinger`, `coolify_private_key`, `coolify_cloud_token`, `coolify_github_app` |
| Applications | `coolify_application`, `coolify_application_dockerfile`, `coolify_application_docker_image`, `coolify_application_private_git`, `coolify_application_github_app` |
| Databases | `coolify_database_postgresql`, `coolify_database_mysql`, `coolify_database_mariadb`, `coolify_database_redis`, `coolify_database_mongodb`, `coolify_database_clickhouse`, `coolify_database_keydb`, `coolify_database_dragonfly` |
| Services | `coolify_service`, `coolify_service_application` |
| Operations | `coolify_environment_variable`, `coolify_envs_bulk`, `coolify_storage`, `coolify_scheduled_task`, `coolify_deployment`, `coolify_resource_action`, `coolify_database_backup`, `coolify_backup_execution`, `coolify_application_preview` |
| Validation | `coolify_server_validate`, `coolify_cloud_token_validate`, `coolify_s3_storage_validate`, `coolify_api_settings` |

All stateful resources support `terraform import`. Action and validation resources
(`coolify_api_settings`, `coolify_application_preview`, `coolify_cloud_token_validate`,
`coolify_s3_storage_validate`, `coolify_server_validate`, `coolify_resource_action`) are lifecycle-only and do not
support import. See the [Import Guide](guides/import) for composite ID formats,
known limitations, and field normalization behaviors.

## Coolify API Behaviors

Coolify's API normalizes certain values. The provider handles these
automatically, but they are worth knowing:

| Behavior | Affected Resources | What happens |
|---|---|---|
| **Git URL prefix stripping** | All application types | `https://github.com/org/repo` is stored as `org/repo` |
| **Docker image tag stripping** | `coolify_application_docker_image` | `redis:7-alpine` is stored as `redis` (all tags, not just `:latest`) |
| **Port override** | `coolify_application_dockerfile` | Coolify may return a different port than configured |
| **Immutable fields omitted** | All databases, apps, services | `project_uuid`, `server_uuid`, `environment_name` not returned by GET |
| **Storage name prefixing** | `coolify_storage` | Coolify prepends an internal resource UUID to the storage name |
| **Base64 encoding required** | `coolify_application_dockerfile` | `dockerfile_location` must be base64-encoded despite the field name; use `base64encode()` |
| **Application refresh can lag create** | All applications | Terraform fails with `Application created but refresh failed` if Coolify returns a UUID but the app is not yet readable through the API |
| **Cloud token validation** | `coolify_cloud_token` | Coolify validates the token against the cloud provider's API on creation (cannot use placeholder values) |
| **GitHub App repository validation** | `coolify_application_github_app` | Coolify verifies repository access during create, so the referenced `coolify_github_app` must have installation access to the target repository |
| **Async deletion** | `coolify_project` | Child resources are deleted asynchronously; project delete retries automatically |

The provider preserves your configured values for all normalized fields.
You should not see diffs on `terraform plan` after the initial apply.

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `ca_cert` (String) PEM-encoded CA certificate to trust for TLS connections to the Coolify API. Use this when your Coolify instance uses a self-signed certificate or an internal CA. Env: `COOLIFY_CA_CERT`.
- `cf_access_client_id` (String) Cloudflare Access `CF-Access-Client-Id` header value. Required when the Coolify instance is behind Cloudflare Access. Env: `COOLIFY_CF_ACCESS_CLIENT_ID`.
- `cf_access_client_secret` (String, Sensitive) Cloudflare Access `CF-Access-Client-Secret` header value. Required when the Coolify instance is behind Cloudflare Access. Env: `COOLIFY_CF_ACCESS_CLIENT_SECRET`.
- `client_cert` (String) PEM-encoded client certificate for mutual TLS. Set this together with `client_key`, or set neither. `ca_cert` is still the trust anchor for the Coolify server. Env: `COOLIFY_CLIENT_CERT`.
- `client_key` (String, Sensitive) PEM-encoded private key for `client_cert`. Unencrypted PKCS#8 or PKCS#1. Encrypted keys are rejected. Env: `COOLIFY_CLIENT_KEY`.
- `endpoint` (String) Coolify API endpoint. Env: COOLIFY_ENDPOINT.
- `header_command` (Block List) Runs a command once per Terraform command and sends its stdout as one header. Use this for a short-lived proxy credential such as `gcloud auth print-identity-token`. The command is not a shell. The value is not refreshed during apply. `Authorization` is rejected. On Terraform 1.10 or later, prefer an ephemeral value in `headers` when the token already exists as one. (see [below for nested schema](#nestedblock--header_command))
- `headers` (Map of String, Sensitive) Extra HTTP headers sent with every request to the Coolify API, for a proxy in front of Coolify that authenticates requests (for example `Proxy-Authorization` for Google Cloud IAP). `Authorization`, `Accept`, `Content-Type`, `Content-Length`, `Host`, and `User-Agent` cannot be set, since the provider sets them itself. `CF-Access-Client-Id` cannot be set together with `cf_access_client_id`, nor `CF-Access-Client-Secret` with `cf_access_client_secret`. A redirect to a different host drops these headers and the Cloudflare Access headers. Env: `COOLIFY_HEADERS`, a JSON object of strings, used only when this attribute is not set.
- `insecure` (Boolean) Skip TLS certificate verification. **Not recommended for production.** Use `ca_cert` instead when possible. Env: `COOLIFY_INSECURE`.
- `proxy` (Block, Optional) Outbound proxy for the machine running Terraform (a CI runner or an HCP Terraform agent). This is not the authenticating proxy in front of Coolify; that credential belongs in `headers`. When `url` is set it replaces `HTTP_PROXY` and `HTTPS_PROXY`. Supported schemes are `http`, `socks5`, and `socks5h`. An `https` proxy URL is rejected because Go would apply the Coolify TLS settings to the proxy hop. Env: `COOLIFY_PROXY_URL`, `COOLIFY_PROXY_USERNAME`, `COOLIFY_PROXY_PASSWORD`, `COOLIFY_NO_PROXY`. A set attribute wins. `url` is required when the block is set. It is optional in the schema so an omitted block does not require it. (see [below for nested schema](#nestedblock--proxy))
- `request_timeout` (Number) Per-attempt HTTP timeout in seconds (default: 30, maximum: 300). This does not change the 30 second dial timeout or the 10 second TLS handshake timeout. A value below 30 also limits the dial, because the client timeout covers the whole attempt. Retries start a new attempt with the same limit. Env: `COOLIFY_REQUEST_TIMEOUT`.
- `retry_max` (Number) Maximum number of API request retries (default: 3).
- `retry_max_wait` (Number) Maximum wait between retries in seconds (default: 30).
- `retry_min_wait` (Number) Minimum wait between retries in seconds (default: 1).
- `tls_server_name` (String) DNS name to use for TLS SNI and certificate verification when it differs from the host in `endpoint`. The HTTP Host header stays the endpoint host, so a name-based router still needs that name in `endpoint`. Env: `COOLIFY_TLS_SERVER_NAME`.
- `token` (String, Sensitive) Coolify API token. Env: COOLIFY_TOKEN.

<a id="nestedblock--header_command"></a>
### Nested Schema for `header_command`

Required:

- `command` (List of String) Program and arguments. The first element is the program.
- `name` (String) Header name. The names rejected by `headers` are rejected here.

Optional:

- `prefix` (String) Text prepended to stdout, such as `Bearer `.
- `timeout` (Number) Seconds to wait for the command (default: 30, maximum: 300).


<a id="nestedblock--proxy"></a>
### Nested Schema for `proxy`

Optional:

- `no_proxy` (String) Comma-separated hosts that connect directly. An empty string bypasses nothing. When omitted, `NO_PROXY` from the environment still applies.
- `password` (String, Sensitive) Password for proxy basic auth.
- `url` (String) Proxy address, such as `http://proxy.example:8080` or `socks5://proxy.example:1080`. Required when the block is set.
- `username` (String) Username for proxy basic auth. Prefer this over userinfo in `url`.