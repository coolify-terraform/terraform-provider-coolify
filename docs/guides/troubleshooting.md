---
page_title: "Troubleshooting - coolify Provider"
subcategory: ""
description: |-
  Debugging tips and how to collect diagnostic logs for the Coolify Terraform provider.
---

# Troubleshooting

This guide explains how to enable debug logging, interpret log output,
and prepare diagnostic information for bug reports.

## Enable Debug Logging

Terraform providers use the `TF_LOG_PROVIDER` environment variable for
structured logging. Set it before running any Terraform command:

```bash
# Show provider-level debug messages (CRUD operations, state changes)
TF_LOG_PROVIDER=DEBUG terraform plan

# Show full HTTP request/response tracing (API calls, retries, payloads)
TF_LOG_PROVIDER=TRACE terraform plan

# Save logs to a file for sharing
TF_LOG_PROVIDER=DEBUG terraform plan 2>debug.log
```

### Log Levels

| Level | What you see | When to use |
|-------|-------------|-------------|
| `WARN` | Read-back failures, unexpected API formats | First pass: check for obvious problems |
| `DEBUG` | CRUD entry/exit with resource UUID, state removals | Debugging state drift or missing resources |
| `TRACE` | Full HTTP requests/responses with redacted JSON payloads, retry attempts | Investigating API-level issues |

> **Tip:** Start with `DEBUG`. Only switch to `TRACE` if you need to see
the raw API communication.

## What the Logs Show

### At DEBUG Level

```
[DEBUG] creating resource: resource_type=coolify_server
[DEBUG] reading resource: resource_type=coolify_server uuid=abc-123
[DEBUG] resource not found, removing from state: resource_type=coolify_server uuid=abc-123
```

### At TRACE Level

```
[TRACE] API request: method=POST path=/api/v1/servers body={"name":"my-server","ip":"10.0.0.1","password":"[REDACTED]"}
[TRACE] API response: method=POST path=/api/v1/servers status=200 body_excerpt={"uuid":"abc-123"}
[TRACE] [retry] retrying request: method=GET url=/api/v1/servers/abc-123 attempt=2
```

Sensitive fields in structured JSON payloads, including passwords,
tokens, private keys, and environment variable values, are automatically
replaced with `[REDACTED]`. Non-JSON bodies are omitted. Logged response
body excerpts are truncated to 500 characters.

## Common Issues

### "Provider produced inconsistent result after apply"

This usually means the provider's schema default doesn't match what the
Coolify API actually returns. Enable `TRACE` logging to see what the API
returned vs. what the provider expected.

### Resource disappears from state unexpectedly

If `terraform plan` shows a resource needs to be created that already
exists, the provider's Read method received a 404 from the API and
removed it from state. Check `DEBUG` logs for
`"resource not found, removing from state"`.

Common causes:
- The resource was deleted outside Terraform (via Coolify UI)
- The API token lacks permissions to read the resource
- The Coolify server was restarted and the resource hasn't finished
  starting up

### Perpetual diffs (plan always shows changes)

Enable `TRACE` logging and compare the API response values against your
configuration. Common causes:
- Coolify normalizes a value (e.g., strips `https://github.com/` from
  git URLs)
- A field has a different default than expected
- A sensitive field is hidden by the API (token needs `read:sensitive`
  permission)

### Rate limiting (429 errors)

The provider retries 429 responses automatically with backoff. If you
see persistent 429s in `TRACE` logs, increase the Coolify API rate limit:

```bash
# On your Coolify server
echo "API_RATE_LIMIT=1000" >> /data/coolify/source/.env
cd /data/coolify/source
docker compose --env-file .env -f docker-compose.yml -f docker-compose.prod.yml \
  up -d coolify --force-recreate
```

## Provider Debug Mode (Delve)

For provider developers who need interactive debugging beyond log output,
the provider supports Go's standard debugger workflow via
[Delve](https://github.com/go-delve/delve).

### Prerequisites

Install Delve:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

### Starting a Debug Session

1. **Build the provider with debug symbols:**

```bash
go build -gcflags="all=-N -l" -o terraform-provider-coolify .
```

The `-gcflags` flags disable compiler optimizations so breakpoints
work reliably.

2. **Start the provider in debug mode:**

```bash
./terraform-provider-coolify --debug
```

The provider prints a `TF_REATTACH_PROVIDERS` JSON string to stderr
and waits for Terraform to connect:

```
Provider started. To attach Terraform CLI, set the TF_REATTACH_PROVIDERS
environment variable:

    TF_REATTACH_PROVIDERS='{"registry.terraform.io/SebTardifLabs/coolify":{...}}'
```

3. **In another terminal, run Terraform with the reattach variable:**

```bash
export TF_REATTACH_PROVIDERS='...'  # paste the value from step 2
terraform plan
```

Terraform connects to your running provider instance instead of
launching a new one.

### Using Delve Breakpoints

To set breakpoints and step through code, start the provider under Delve
instead of running it directly:

```bash
dlv exec ./terraform-provider-coolify -- --debug
```

Then use Delve commands (`break`, `continue`, `next`, `print`) to set
breakpoints in Create, Read, Update, or Delete methods.

### When to Use Debug Mode vs. Logging

| Scenario | Recommended approach |
|----------|---------------------|
| Investigating API request/response payloads | `TF_LOG_PROVIDER=TRACE` |
| Checking state values during plan/apply | `TF_LOG_PROVIDER=DEBUG` |
| Stepping through resource Create/Read/Update logic | Delve debug mode |
| Inspecting Terraform Plugin Framework internals | Delve debug mode |
| Reproducing a race condition | Delve with conditional breakpoints |

> **Note:** Debug mode is a provider development tool. For end-user
troubleshooting, use `TF_LOG_PROVIDER` logging instead.

## Preparing a Bug Report

When filing a bug report, include:

1. **Terraform and provider versions:** `terraform version`
2. **Coolify version:** visible in the Coolify dashboard footer
3. **Your `.tf` configuration** (redact real credentials)
4. **Debug log output:**

```bash
TF_LOG_PROVIDER=DEBUG terraform plan 2>debug.log
```

5. **Review the log for sensitive values** before sharing. The provider
   redacts most sensitive fields in structured JSON payloads automatically
   and omits non-JSON bodies, but you should still review the log before
   posting it publicly.

> File bug reports at
[github.com/SebTardifLabs/terraform-provider-coolify/issues](https://github.com/SebTardifLabs/terraform-provider-coolify/issues)
using the **Bug Report** template.