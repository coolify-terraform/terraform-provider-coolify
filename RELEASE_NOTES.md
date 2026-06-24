## What's New

This release fixes a critical bug for users running Coolify v4.1.0 or v4.1.1, adds a contract compatibility CI check to prevent similar regressions, and bumps the minimum supported Coolify version to v4.1.0.

### Fix: Database Health Check 422 Error ([#549](https://github.com/coolify-terraform/terraform-provider-coolify/issues/549))

Users importing PostgreSQL (or any database) on **Coolify v4.1.0 or v4.1.1** would see a 422 error on the next `terraform apply`:

```
Error: Error updating PostgreSQL database

health_check_interval: ["This field is not allowed."];
health_check_timeout: ["This field is not allowed."];
...
```

**Root cause:** The schema used `Default` values for health check attributes. After import, Terraform saw the API-returned values as "new" (state was null, Default filled the plan), generated a spurious diff, and sent those fields in a PATCH request. Coolify v4.1.0/v4.1.1 does not accept health check fields on database updates.

**Fix:** Replaced `Default` with `UseStateForUnknown()` on all 5 health check attributes. This preserves the value returned by GET in state, preventing the spurious diff entirely. Users on Coolify v4.1.2+ can still explicitly update health check fields.

### Minimum Coolify Version: v4.1.0

The minimum supported Coolify version is now **v4.1.0** (previously v4.0.0). The provider's health check at startup validates this and reports a clear error if the instance is too old.

Between v4.0.0 and v4.1.0, Coolify added 69 application create fields. Only 12 of 81 fields existed in v4.0.0, making it effectively unsupported. Formalizing v4.1.0 as the minimum aligns the documentation with reality.

### New: Contract Compatibility CI Check

A new `make contract-compat` target compares `allowed_fields` across versioned Coolify API contracts. This catches version-dependent fields at development time, before they reach users. The check runs as part of `make ci` and includes a `KNOWN_VERSION_DEPENDENT` registry for fields that have been intentionally handled with `UseStateForUnknown()`.

Versioned contracts are now included for v4.1.0, v4.1.1, and v4.1.2.

### Test Coverage

Health check update tests now cover all 8 database types (PostgreSQL, Redis, ClickHouse, MongoDB, MariaDB, MySQL, KeyDB, Dragonfly), ensuring the #549 fix cannot regress for any database.

### Internal Improvements

- Contract extraction script now handles `array_merge($allowedFields, [...])` patterns in Coolify PHP source
- Destroy no-op coverage and `.gitignore` audit gaps closed
- Hetzner test helper extracted for reuse across Hetzner resource tests
- Scenario tests now trigger on `internal/` changes
- FOSSA license scan skipped on Dependabot PRs (eliminates false failures)
- Social preview image auto-updates with live stats on each release

### Dependencies

- `actions/checkout` 6.0.3 to 7.0.0
- `github.com/pb33f/libopenapi` 0.37.3 to 0.38.0
- Go minor dependency group updates

960+ tests passing (unit + acceptance).

**Full Changelog**: https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.5...v0.1.6