---
page_title: "Domains and HTTPS"
subcategory: "Guides"
description: |-
  Configure custom domains, automatic HTTPS, and URL redirects for Coolify applications.
---

# Domains and HTTPS

This guide covers how to assign a custom domain to your application,
how Coolify handles TLS certificates, and how to configure redirects.

## How Coolify Routing Works

Coolify runs a reverse proxy (Traefik by default, Caddy is also supported)
on each server. When a request arrives, the proxy inspects the `Host`
header and routes it to the correct application container. The proxy is
managed automatically; you do not configure Traefik/Caddy directly.

```
Internet ──► Server IP ──► Traefik/Caddy ──► Application Container
                           (port 80/443)      (internal port, e.g. 3000)
```

The `ports_exposes` attribute tells Coolify which port your application
listens on inside the container. The `domains` attribute tells the proxy
which hostname(s) should route to that container.

## Setting a Domain

Set the `domains` attribute on any application resource:

```hcl
resource "coolify_application" "web" {
  name           = "my-app"
  project_uuid   = coolify_project.main.uuid
  server_uuid    = data.coolify_server.prod.uuid
  git_repository = "https://github.com/myorg/myapp"
  git_branch     = "main"
  build_pack     = "nixpacks"
  ports_exposes  = "3000"
  domains        = "https://app.example.com"
}
```

### DNS Setup

Before Coolify can route traffic and issue a TLS certificate, you need
a DNS record pointing your domain to the server's IP address:

| Record Type | Name | Value |
|------------|------|-------|
| `A` | `app.example.com` | `<server-ip>` |

Propagation typically takes 1-5 minutes but can take up to 48 hours
depending on your DNS provider.

## Automatic HTTPS

Coolify uses Let's Encrypt to issue free TLS certificates automatically.
When you set `domains = "https://app.example.com"`:

1. Coolify configures the proxy to listen for the domain
2. Let's Encrypt issues a certificate via the ACME HTTP-01 challenge
3. The proxy serves traffic over HTTPS
4. Certificates are renewed automatically before expiry

For this to work:
- The domain's DNS A record must point to the server's public IP
- Port 80 must be open (Let's Encrypt uses HTTP-01 validation)
- Port 443 must be open for HTTPS traffic

~> **Use `https://` in the `domains` value.** If you write
`http://app.example.com`, Coolify will NOT issue a TLS certificate.
The `https://` prefix signals that you want automatic HTTPS.

## Forcing HTTPS

The `is_force_https_enabled` attribute (defaults to `true`) redirects
all HTTP requests to HTTPS:

```hcl
resource "coolify_application" "web" {
  # ...
  domains                = "https://app.example.com"
  is_force_https_enabled = true  # default, HTTP requests redirect to HTTPS
}
```

Leave this enabled unless you have a specific reason to serve HTTP
(for example, a health check endpoint behind a load balancer that
does TLS termination).

## URL Redirects (WWW vs Non-WWW)

The `redirect` attribute controls how Coolify handles `www` vs non-`www`
versions of your domain:

| Value | Behavior | Example |
|-------|----------|---------|
| `"www"` | Redirects non-www to www | `app.example.com` redirects to `www.app.example.com` |
| `"non-www"` | Redirects www to non-www | `www.app.example.com` redirects to `app.example.com` |
| `"both"` | Serves both without redirect | Both `app.example.com` and `www.app.example.com` work |

```hcl
resource "coolify_application" "web" {
  # ...
  domains  = "https://app.example.com"
  redirect = "non-www"  # www.app.example.com → app.example.com
}
```

-> **Note:** If you use `"www"` or `"both"`, create a DNS record for
the `www` subdomain as well (`CNAME www → app.example.com` or another
`A` record).

## Multiple Domains

Separate multiple domains with commas:

```hcl
resource "coolify_application" "web" {
  # ...
  domains = "https://app.example.com,https://www.example.com,https://example.com"
}
```

All domains receive TLS certificates. Requests to any of them route to
the same application container.

## Wildcard Domains

Servers have a `wildcard_domain` attribute that enables automatic domain
assignment for applications that do not set an explicit `domains` value:

```hcl
resource "coolify_server" "prod" {
  # ...
  # This is a read-only attribute set via Coolify UI/API
  # wildcard_domain = "example.com"
}
```

When a wildcard domain is configured on the server, Coolify generates
a domain like `<app-name>.<wildcard-domain>` for applications that
do not have an explicit `domains` attribute set.

-> **Wildcard TLS** requires DNS-01 validation (not HTTP-01) and
additional proxy configuration. For most setups, explicit per-app
domains with HTTP-01 validation are simpler and recommended.

## Troubleshooting

### Domain not resolving

1. Verify the DNS record: `dig app.example.com` should return your server's IP
2. Wait for DNS propagation (check with `dig @8.8.8.8 app.example.com`)
3. Confirm ports 80 and 443 are open on the server's firewall

### TLS certificate not issued

1. Check that you used `https://` in the `domains` value
2. Verify port 80 is open (Let's Encrypt HTTP-01 needs it)
3. Check Coolify proxy logs: the Coolify UI shows proxy logs under the server
4. Confirm the DNS record is correct and propagated
5. Let's Encrypt has rate limits: 50 certificates per domain per week

### Redirect loops

If you get infinite redirect loops:
- Check if another proxy (Cloudflare, nginx) is also doing HTTPS redirect
- If behind Cloudflare, set SSL mode to "Full (strict)" not "Flexible"
- Disable `is_force_https_enabled` temporarily to isolate the issue
