## What's New

This release fixes a usability problem with `coolify_github_app` import, adds a new scenario for importing existing Coolify resources into Terraform, and strengthens test coverage.

### Fix: GitHub App Import Now Uses the App ID ([#558](https://github.com/coolify-terraform/terraform-provider-coolify/issues/558))

Importing a `coolify_github_app` resource previously required Coolify's internal database `id`, which is not exposed anywhere in the Coolify UI. Users only see the GitHub App ID (`app_id`) on the Sources page, making import effectively broken with a confusing "Cannot import non-existent remote object" error.

**Before:**
```bash
# Had to guess the internal database id (not visible in Coolify UI)
terraform import coolify_github_app.example 42
```

**After:**
```bash
# Use the GitHub App ID shown in Coolify UI (Sources > GitHub App > App Id)
terraform import coolify_github_app.example 12345
```

The import now accepts the GitHub App ID directly. A new `GetGitHubAppByAppID` client method scans the list endpoint to resolve the app, with full unit test coverage (found, not found, skips built-in records, empty list).

### New Scenario: Import Existing Resources (`acme-import-existing`)

The 17th ACME Corp scenario covers the most common entry point for teams adopting Terraform on an existing Coolify instance. Instead of recreating resources from scratch, it shows how to bring pre-existing projects, applications, and environment variables under Terraform management using `terraform import`.

### Documentation Fixes

- Scenario count updated from 16 to 17 across all docs (AGENTS.md, README.md, ROADMAP.md, architecture guide)
- `make ci` target list corrected to include all 13 targets (`actionlint-check` and `contract-compat` were missing from docs)

970+ tests passing (unit + acceptance), 17 tested scenarios.

**Full Changelog**: https://github.com/coolify-terraform/terraform-provider-coolify/compare/v0.1.6...v0.1.7