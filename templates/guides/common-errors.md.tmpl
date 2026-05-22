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

### UUID format invalid

```
Error: Invalid Attribute Value
uuid must be a valid UUID (e.g. "550e8400-e29b-41d4-a716-446655440000")
```

**Cause:** a UUID field received a malformed value.

**Fix:** check that all UUID values use the standard format
(`xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`). Find UUIDs in the Coolify UI
or with `terraform state show`.

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

**Fix:**
1. Upgrade to a `root` API token (fixes most sensitive field issues)
2. Check if the field has a known normalization (see
   [API Behaviors](/#coolify-api-behaviors) in the docs index)
3. If the field is a password you set, the token permissions are the
   most likely cause

### "forces replacement"

```
# coolify_application.web must be replaced
~ server_uuid = "old-uuid" -> "new-uuid" # forces replacement
```

**Cause:** you changed an immutable field. These fields are set at
creation time and cannot be updated. The only way to change them is
to destroy and recreate the resource.

**Immutable fields:** `project_uuid`, `server_uuid`,
`environment_name` on all applications and databases.

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
