---
page_title: "Common Errors"
subcategory: "Guides"
description: |-
  A reference of common error messages from the Coolify API and Terraform, with causes and fixes.
---

# Common Errors

This guide lists error messages you may encounter when using the Coolify
provider, explains what causes them, and shows how to fix them.

## Authentication Errors (401)

### "Unauthenticated"

```
Error: Error reading server: server abc-123: getting server abc-123:
unexpected status 401: {"message":"Unauthenticated."}
```

**Cause:** the Coolify API rejected your token.

**Fix:**
1. **API not enabled.** Enable the API in the Coolify UI under **Settings**.
   The API is disabled by default.
2. **Token expired or revoked.** Generate a new token under
   **Security > API Tokens**.
3. **Wrong endpoint.** Verify `COOLIFY_ENDPOINT` points to your Coolify
   instance (including the correct port, e.g., `http://localhost:8000`).
4. **Token format.** The token must include the numeric prefix:
   `42|abc123def456...`. If you copied only the hash portion, it will
   not authenticate.

### Empty sensitive fields (passwords, keys)

**Symptom:** `terraform plan` shows diffs on password fields.
The API returns empty strings for sensitive values.

**Cause:** your API token lacks `root` or `read:sensitive` permission.

**Fix:** create a new token with `root` permission in
**Security > API Tokens**. See the
[Secrets Management](secrets-management) guide for details.

## Validation Errors (422)

### Field validation failed

```
Error: Error creating application: project abc-123, server def-456:
creating application: unexpected status 422:
{"message":"The ports exposes field is required."}
```

**Cause:** the Coolify API requires a field that was not provided.
Even if the Terraform schema marks a field as `Optional`, the Coolify
API may require it for certain operations.

**Fix:** add the missing field to your resource configuration. Common
required fields that are not always obvious:

| Resource | Often-required fields |
|----------|---------------------|
| All applications | `ports_exposes` |
| `coolify_application_private_git` | `private_key_uuid` |
| `coolify_application_github_app` | `github_app_uuid` |
| `coolify_server` | `ip`, `private_key_uuid` |
| `coolify_database_backup` | `frequency` (cron expression) |

### Cannot set docker_compose_domains without docker_compose_raw

```
Error: Error setting application extended fields
Application … was created, but the post-create PATCH for extended fields failed:
… Cannot set docker_compose_domains without docker_compose_raw.
Reload the compose file from the git repository first.
```

**Cause:** Coolify refuses `docker_compose_domains` until `docker_compose_raw`
is set. For git-sourced applications with `build_pack = "dockercompose"`, the
compose file is loaded only when a deployment runs; there is no separate
"load compose" API. This guard exists on **all Coolify versions supported by
this provider** (v4.1.0 and later: verified on v4.1.0, v4.1.1, v4.1.2, and
v4.2.0).

**Fix (two-stage apply):**

1. Create the application **without** `docker_compose_domains`. Deploy once
   (`instant_deploy = true`, Coolify UI, or `coolify_deployment` with
   `wait_for_completion = true`) and wait until that deployment succeeds so
   Coolify has `docker_compose_raw`.
2. Add `docker_compose_domains` as a JSON array and apply again:

```hcl
docker_compose_domains = jsonencode([
  { name = "web", domain = "https://app.example.com" }
])
```

**Alternative:** use `coolify_service` with inline `docker_compose_raw` when
the compose YAML can live in Terraform (see the Docker Compose stacks guide).
That path does not depend on a git deploy to populate compose raw.

Do **not** force `instant_deploy = true` only to hide the constraint if you
do not want an immediate deploy; the ordering is intrinsic to Coolify.

### preview domain updates require Coolify >= v4.3.15

```
Error: preview domain updates require Coolify >= v4.3.15
```

**Cause:** `coolify_application_preview` set `domains`,
`docker_compose_domains`, or `force_domain_override` on Coolify older
than v4.3.15. Those writes are a hard apply error, not a plan warning.

**Fix:** omit the domain attributes on older Coolify, or upgrade to
**>= v4.3.15**.

### docker_compose_domains must be a JSON array

```
Error: docker_compose_domains must be a JSON array
```

**Cause:** a compose preview was given the GET object map
(`{"web":{"domain":"https://pr.example.com"}}`) instead of the write
array. Preview does not accept the GET object map.

**Fix:** send a JSON array of `{name, domain}` objects:

```hcl
docker_compose_domains = jsonencode([{ name = "web", domain = "https://pr.example.com" }])
```

### Coolify version cannot write some application settings

Full matrix of which application attributes need Coolify 4.2 vs 4.3:
[Coolify Version Support](coolify-version-support).

```
Warning: Coolify version cannot write some application settings

This Coolify instance (4.1.2) is older than v4.2.0, which is required to write:
is_gzip_enabled, …. The provider will keep these values in Terraform state but
will not send them to the Coolify API.
```

**Cause:** Coolify rejects application write fields that are not on that
version's allow list. The provider withholds them on PATCH so Create does not
422, but still keeps configured values in state. The same warning title covers
both gates:

- Coolify **v4.1.x**: 4.2 settings (gzip, git LFS, preview deploys, build
  secrets, and related fields)
- Coolify **v4.2.x**: 4.3 settings (log drain, GPU, `custom_internal_name`,
  `noindex_domains`, `max_restart_count`, and related fields)

Notification `restart_limit_reached` on older Coolify is a real extra-key
422 (not version-gated). Omit that attribute unless your instance accepts
it (Coolify >= v4.3.15).

**Fix:** upgrade Coolify to the version named in the warning (**v4.2.0** or
**v4.3.0** as listed), or remove those attributes from configuration. The
warning is intentional; it is not a hard error. Full attribute lists:
[Coolify Version Support](coolify-version-support).

### Server has multiple destinations and you do not set destination_uuid

```
Error: Error creating application: …
Server has multiple destinations and you do not set destination_uuid.
```

**Cause:** the Coolify server has more than one Docker network destination
(Coolify >= v4.2.0). Create APIs require `destination_uuid` when more than
one destination exists. The provider auto-resolves a destination when it can
(prefers network `coolify`, then the first standalone destination, then the
first entry). Resolution fails if the destinations list is empty or the API
returns an unexpected error.

**Fix:**

1. List destinations with `data.coolify_destinations` or
   `coolify_destination` resources for the server.
2. Set `destination_uuid` on the application, database, or service resource
   to the destination you want (create-only; changing it forces replacement).
3. Prefer a single default network named `coolify` so auto-resolution is
   deterministic when `destination_uuid` is omitted.

### UUID format invalid

```
Error: Invalid Attribute Value
uuid must be a valid UUID (e.g. "550e8400-e29b-41d4-a716-446655440000")
or Coolify identifier
```

**Cause:** a UUID field received a malformed value.

**Fix:** Coolify resource ids are either:

- RFC 4122 UUIDs (`xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`), or
- Coolify identifiers: **exactly 7** alphanumeric characters (legacy
  `Cuid2(7)`, still present on older instances and some Coolify Cloud
  teams) or **20-36** alphanumeric characters (modern NanoID/Cuid2).

Copy the id from the Coolify UI URL or API (`GET /servers`, etc.), or
from `terraform state show`. Values with dashes outside the RFC form,
underscores, spaces, or lengths other than 7 and 20-36 are rejected.

### Cron syntax invalid

```
Error: Invalid Attribute Value
frequency must be a valid cron expression
```

**Cause:** the `frequency` field on `coolify_database_backup` or
`coolify_scheduled_task` received an invalid cron expression.

**Fix:** use a valid cron expression. Coolify supports standard 5-field
cron (`* * * * *`) and predefined schedules (`@daily`, `@hourly`,
`@weekly`).

## Not Found Errors (404)

### Resource disappeared from state

**Symptom:** `terraform plan` shows a resource needs to be created,
but it already exists in Coolify.

```
[DEBUG] resource not found, removing from state:
  resource_type=coolify_application uuid=abc-123
```

**Cause:** the provider's Read method received a 404 from the API and
removed the resource from Terraform state.

**Common reasons:**
- The resource was deleted via the Coolify UI or API outside Terraform
- The server is unreachable and Coolify cannot report the resource
- The API token was changed and lacks access to the resource

**Fix:**
- If the resource still exists, re-import it:
  `terraform import coolify_application.web <uuid>`
- If the resource was intentionally deleted, remove it from your `.tf`
  file and run `terraform apply`

### Hetzner list data source: token not found

```
Error: Error listing Hetzner networks
cloud_provider_token_uuid=...: listing hetzner networks: unexpected status 404:
{"message":"Hetzner cloud provider token not found."}
```

The same pattern appears for `coolify_hetzner_images`,
`coolify_hetzner_locations`, `coolify_hetzner_server_types`,
`coolify_hetzner_ssh_keys`, and `coolify_hetzner_firewalls`.

**Cause:** `cloud_provider_token_uuid` does not match a Hetzner token
that Coolify can use. Typical cases:

- The UUID is a DigitalOcean or Vultr token, not a Hetzner token
- The token was deleted in the Coolify UI
- The token belongs to a different team than the API credential

**Fix:**

1. Read `data.coolify_cloud_tokens` and confirm `cloud_provider = "hetzner"`
2. Recreate the token with `coolify_cloud_token` if it is gone
3. Firewalls and networks also require Coolify >= v4.2.0; older instances
   return 404 for those two routes even with a valid token

### "Application created but refresh failed"

```
Error: Application created but refresh failed
Coolify created application abc-123, but the provider could not read it
back: Could not read application abc-123 after create: ...
```

**Cause:** Coolify created the resource (returned a UUID) but the
subsequent GET request failed. The resource exists in Coolify's database
but is not readable through the API yet.

**Common reasons:**
- The target server is not SSH-reachable
- The server has not finished validation (`is_usable = false`)
- Transient network issue between Coolify and the server

**Fix:**
1. Check server status: the server must have `is_usable = true`
2. Validate the server: `coolify_server_validate` or the Coolify UI
3. Run `terraform apply` again; the partial state was saved

## Conflict Errors

### "Project has resources, so it cannot be deleted"

```
Error: Error deleting project: project abc-123:
unexpected status 500: {"message":"Project has resources, ..."}
```

**Cause:** `terraform destroy` tries to delete the project after
deleting its child resources, but Coolify deletes applications
asynchronously. The apps have not finished deleting when the project
delete fires.

**Fix:** the provider retries project deletion automatically. If you
see this error, it usually means the retry limit was reached. Wait
a few seconds and run `terraform destroy` again. The applications
will have finished deleting by then.

## Terraform-Specific Errors

### Can't configure a value for a computed attribute

```
Error: Can't configure a value for "domain_port_overrides"
```

**Cause:** setting a computed GET-only attribute such as
`domain_port_overrides` in a resource block.

**Fix:** remove it from the resource configuration. Read it from the
resource or `data.coolify_application` instead.

### "Provider produced inconsistent result after apply"

```
Error: Provider produced inconsistent result after apply
When applying changes to coolify_database_postgresql.db, provider
produced an unexpected new value for .postgres_password
```

**Cause:** the value Terraform set during Create does not match the
value the API returned on Read. Common triggers:

- **Sensitive field hidden:** the API returned an empty string because
  the token lacks `read:sensitive` permission
- **Value normalized by API:** Coolify changed the value (e.g., stripped
  a URL prefix, base64-encoded content)
- **Default mismatch:** the provider's schema default differs from
  Coolify's actual default
- **List reorder on `urls` / `noindex_domains`:** Coolify GET may return
  these lists in a different order than HCL. The provider keeps the HCL
  order when the set of values matches. If you still see this on an
  older provider, upgrade, or reorder HCL to match GET as a workaround.

**Fix:**
1. Upgrade to a `root` API token (fixes most sensitive field issues)
2. Check if the field has a known normalization (see
   [API Behaviors](/#coolify-api-behaviors) in the docs index)
3. If the field is a password you set, the token permissions are the
   most likely cause
4. On `urls` or `noindex_domains`, upgrade the provider (it keeps HCL
   order) or temporarily match HCL to the GET order

### Unexpected public domain (`sslip.io` or wildcard FQDN)

**Symptom:** after `terraform apply`, the application has a public URL
like `http://{uuid}.{ip}.sslip.io` even though you never set `domains`.

**Cause:** Coolify defaults `autogenerate_domain` to `true` on create.
When `domains` is blank, it generates a Traefik host automatically.

**Fix:** set `autogenerate_domain = false` for internal apps (workers,
queues, sidecars). See the [Domains and HTTPS](domains-and-https)
guide. Clearing an existing FQDN with `domains = ""` is blocked by a
Coolify update-path bug (`$request->has('domains')`); tracked as #647.

### Server proxy `false` flags stay `true`

**Symptom:** `coolify_server_proxy` plan sets `redirect_enabled = false`
or `generate_exact_labels = false`, apply succeeds, and the next plan
shows the value still `true`.

**Cause:** Coolify's proxy PATCH uses Laravel `$request->has()` for those
bools. JSON `false` is treated as absent, so the stored default (`true`)
stays. `redirect_url` uses `exists()` and does persist.

**Fix:** leave `redirect_enabled` and `generate_exact_labels` unset or
`true` until Coolify switches those gates to `exists()`. Prove the update
path with `redirect_url` (use a resolvable host such as
`https://example.com`; reserved names like `example.invalid` return 422).

### "forces replacement"

```
# coolify_application.web must be replaced
~ server_uuid = "old-uuid" -> "new-uuid" # forces replacement
```

**Cause:** you changed an immutable field. These fields are set at
creation time and cannot be updated. The only way to change them is
to destroy and recreate the resource.

**Immutable fields:** `project_uuid`, `server_uuid`,
`environment_name` on all applications and databases;
`github_app_uuid` on `coolify_application_github_app`.

**Fix:** if you intentionally want to move a resource to a different
server or project, accept the replacement. If this was accidental,
revert the field value in your `.tf` file.

### Import state mismatch

```
Error: import - coolify_application.web attribute "project_uuid"
expected "" got "abc-123"
```

**Cause:** after `terraform import`, some fields are missing from state
because the Coolify API does not return them in GET responses.

**Fix:** set the missing fields in your `.tf` configuration before
running `terraform plan`. See the
[Import Guide](import#known-limitations) for the full list of fields
the API may not return.

-> **Tip:** Use the compound import format for applications, databases,
and services to populate `project_uuid`, `server_uuid`, and
`environment_name` automatically:
`terraform import coolify_application.web <project-uuid>:<server-uuid>:production:<app-uuid>`
