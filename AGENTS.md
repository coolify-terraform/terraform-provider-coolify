# AGENTS.md

## Skills

Read these skills when working in this repo:

- `~/.grok/skills/coolify-test-instance/SKILL.md` — Local Coolify setup, API quirks, SSH validation, acc test troubleshooting. **Read first** when setting up a test instance or debugging real-API failures.
- `~/.grok/skills/terraform-provider-coolify-contrib/SKILL.md` — Contributing conventions for the Coolify upstream project, repo namespace notes, CI gotchas.
- `~/.grok/skills/terraform-provider/SKILL.md` — General Terraform provider patterns (resource implementation, testing, CI, releases).
- `~/.grok/skills/ci-workflow-hygiene/SKILL.md` — CI workflow rules: concurrency groups, timeouts, action versions, security scanners.

## Project

Terraform provider for [Coolify](https://coolify.io/), the open-source self-hosted PaaS.
Built with Go 1.26, Terraform Plugin Framework v1.19, and GoReleaser for releases.
Builds use `GOFIPS140=latest` for FIPS 140-3 compliant cryptography (required for
government/enterprise adoption; set in `.goreleaser.yml` and `release.yml` smoke test).
33 resources, 44 data sources, 870+ tests (unit + acceptance), 9 CI jobs.
16 ACME Corp scenario examples (all with `terraform test` integration tests; acme-private-repo uses plan-only).

## Source of Truth: Coolify Source Code (NOT OpenAPI spec)

**Never guess Coolify API behavior. You have access to the real source code.**

The Coolify project is open source at `github.com/coollabsio/coolify`. When
you encounter ANY question about how the API works, the answer is in the PHP
source code. Do not hypothesize, do not assume, do not test by trial and error.
Read the code.

### When to read the source

- A field isn't returned on GET? Read the controller's response builder.
- An update returns 422? Read the controller's `$allowedFields` and validation rules.
- A field has a surprising default? Read the migration and model `$attributes`.
- A flatten function causes "inconsistent result"? Read the model's accessors
  to see if the API normalizes the value (base64 encode/decode, path prefixing, etc.).
- The OpenAPI spec says one thing but the API does another? The spec is wrong.
  The source code is the truth.

### How to read the source

```bash
# Clone once per session (or reuse /tmp/coolify if already there)
git clone --depth 1 https://github.com/coollabsio/coolify.git /tmp/coolify
```

| Question | Where to look |
|----------|--------------|
| What fields exist on a resource? | `app/Models/Application.php` (`$fillable`) |
| What type is a field? What's the default? | `database/migrations/` (column definitions) |
| Is a field encrypted/sensitive? | Model `$casts` (look for `'encrypted'`) |
| What does the API accept on create/update? | `app/Http/Controllers/Api/ApplicationsController.php` (`$allowedFields`, `$request->validate()`) |
| What validation rules apply? | Controller + `bootstrap/helpers/api.php` (`sharedDataApplications()`) |
| What regex patterns are enforced? | `app/Support/ValidationPatterns.php` |
| What does the API return after create? | Controller method (look for `return response()->json(...)`) |
| Does updating a field trigger a side effect? | Controller method (look for `queue_application_deployment`, `StartService`, etc.) |
| What are the enum values? | `app/Enums/` directory |
| Where are settings stored? | `app/Models/ApplicationSetting.php` (separate table from Application) |

### The contract extraction pipeline

We have automated extraction that reads the source and produces a machine-readable
contract JSON. Use it as the first check, then go deeper into the PHP when needed:

1. `testdata/contracts/coolify-v4.json` -- extracted contract (check here first)
2. `/tmp/coolify/app/Models/*.php` -- the real model when the contract isn't enough
3. `/tmp/coolify/app/Http/Controllers/Api/*.php` -- the real controller for validation logic

The OpenAPI spec at `testdata/specs/` is generated FROM the contract. Do NOT
treat it as authoritative. It had 22 wrong nullability annotations, 3 type
mismatches, and zero validation rules when we compared it against the source.

## Commands

- **Run all checks before pushing**: `make ci` + targeted acceptance tests
- **Note**: `make ci` does NOT include acceptance tests. It DOES include `python-test`, so ensure Python 3.9+ is available locally. For real-API changes, run `make acc-preflight` first, then `make testacc-pkg PKG=./internal/service/<changed-package>/` for serialized package-scoped acceptance coverage, or `make testacc` for the full suite.
- **If `make ci` fails on `check-tfplugindocs`**: install it with `cd tools && GOBIN=$(cd .. && pwd)/bin go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs` then re-run with `PATH="$(pwd)/bin:$PATH" make ci`. Do NOT skip `make ci` and run individual targets instead; that misses `docs-check` and causes CI failures when schema descriptions or templates change.
- Use `make testacc` for the full suite when changing shared code (client, provider, flex, validate)
- Build: `make build`
- Test (all, with race detector): `make test`
- Test (single package): `make test-pkg PKG=./internal/service/project/`
- Acceptance tests (needs running Coolify): `make testacc`
- Lint: `make lint`
- Format: `make fmt`
- Generate docs: `make docs`
- Validate HCL examples: `make validate`
- Install locally: `make install`
- Spec compliance: `make spec-check`
- API coverage doc: `make api-coverage`
- Extract contract from Coolify source: `make contract-extract VERSION=v4.1.0`
- Verify client structs cover contract: `make contract-check`
- Regenerate OpenAPI spec from contract: `make spec-generate`
- Scaffold a new resource: `make scaffold NAME=myresource`
- Merge a PR (sole maintainer): `make merge PR=123`

## Structure

- `main.go` - Provider entry point
- `internal/provider/` - Provider schema, Configure (with health check), resource/data source registration
- `internal/client/` - HTTP client for Coolify API (one file per resource type)
- `internal/service/` - One subpackage per resource type, each with resource.go, data_source.go, and tests
- `internal/flex/` - Type conversion helpers between Go and Terraform Framework types
- `internal/acctest/` - Shared test utilities (provider factories, mock server wrappers, acceptance test helpers)
- `internal/validate/` - Input validators (UUID format, FQDN URL)
- `internal/spectest/` - OpenAPI spec compliance tests, API coverage tracking, and contract coverage tests
- `scripts/` - Contract extraction (`extract-contract.py`), OpenAPI generation (`generate-openapi.py`), contract diff (`diff-contracts.sh`)
- `testdata/contracts/` - Versioned contract JSON files extracted from Coolify source (source of truth for API field definitions)
- `examples/` - Working HCL examples per resource + multi-resource scenarios
- `examples/scenarios/` - ACME Corp real-world scenarios with .tftest.hcl (tested against real Coolify)
- `templates/` - tfplugindocs templates (index.md.tmpl, guides/)
- `docs/` - Auto-generated by tfplugindocs from templates (never edit directly)
- `tools/` - Separate Go module for tfplugindocs installation

## Conventions

### Resource implementation

- One package per resource type under `internal/service/`
- Every resource implements `resource.ResourceWithImportState`
- Handle 404 in Read (call `resp.State.RemoveResource`) and Delete (silently return)
- Use `stringplanmodifier.RequiresReplace()` on immutable fields (project_uuid, server_uuid, environment_name)
- Use `stringplanmodifier.UseStateForUnknown()` on computed fields (uuid, name)
- Use `Optional: true, Computed: true` with `UseStateForUnknown()` on fields where the API returns a default value when the user omits them **and the provider does not declare a `Default`** (e.g., `databases_to_backup` defaults to `"postgres"` on the API side). `Optional` alone causes "inconsistent result after apply" because the framework sees an unexpected value on read-back.
- **Create-only fields** (sent on POST but never returned on GET, e.g. `instant_deploy`) use `Optional: true, Computed: true, Default: booldefault.StaticBool(false)`. The flatten function must preserve the state value when already set and default to `false` when null/unknown (import case): `if f.Field.IsNull() || f.Field.IsUnknown() { *f.Field = types.BoolValue(false) }`. Without this, `ImportStateVerify` fails because the API never returns the field.
- Do **not** combine `UseStateForUnknown()` with a `Default` value (e.g., `int64default.StaticInt64(25)`). The framework applies `Default` before plan modifiers, so `UseStateForUnknown` never fires and is dead code. Use one or the other: `Default` when the provider knows the default, `UseStateForUnknown` when only the API knows it.
- When the Create endpoint only accepts a subset of fields, issue a follow-up PATCH in the Create method for remaining fields so the resource converges in a single apply. Guard with a `hasNonDefault*()` helper to skip the PATCH when all values are defaults. (Example: `coolify_server_hetzner` Create POST only accepts Hetzner-specific fields; `description`, `port`, `user`, `is_build_server` need a post-create PATCH.)
- **Shared schema, distributed Create trap**: When adding fields to a shared schema function (e.g., `CommonServerAttrs` in `server/common.go`), the new fields automatically appear in Update (via `BuildServerUpdateInput` / `addExtendedSettingsUpdate`) and Read (via `FlattenServerCommon`). But Create's post-create PATCH and `hasNonDefault*()` guard are per-resource and must be updated **independently in every resource** that uses the shared schema (`server/resource.go`, `hetzner/resource.go`). Forgetting one causes silent data loss on first apply: the user sets a value, Create ignores it, and it only converges on a second apply via Update. Always grep for `hasNonDefault` after adding shared fields.
- Mark passwords and keys as `Sensitive: true`
- Wrap all client errors: `fmt.Errorf("getting project %s: %w", uuid, err)`

#### Fields the API never returns

Coolify has several patterns where a field exists in the Terraform schema but
the API does not return it (or returns a different value). Each pattern requires
specific schema configuration, flatten handling, and import behavior.

| Pattern | Schema | Flatten | Import | Examples |
|---------|--------|---------|--------|----------|
| **Sensitive, API hides** | `Sensitive: true`, `Optional+Computed` | Preserve from state when API returns `""` | Value lost; user must re-supply | `private_key`, passwords, tokens |
| **Create-only, never returned** | `Optional+Computed`, `Default: false/""` | Preserve state when set; default to `false`/`""` when null/unknown | Gets the default value | `instant_deploy`, Hetzner POST fields |
| **Write-only** | `Optional`, `Sensitive: true` | Preserve from state; never overwrite from API | Value lost; user must re-supply | `github_app.client_secret`, `private_key_uuid` |
| **Terraform-only** | `Optional+Computed` | Preserve state; resolve unknown to `""` | Empty string; add to `ImportStateVerifyIgnore` | `environment.description`, `redeploy_on_update` |
| **Normalized by API** | `Optional+Computed` | Compare normalized forms; preserve user's original if equivalent | Gets the normalized form | `dockerfile_location`, `git_repository` URL |

**When adding a new field**, determine which pattern applies by checking the
Coolify controller source. If `GET` does not include the field in its response
builder, it falls into one of these patterns. Use the matching schema
configuration and flatten handling to avoid "inconsistent result after apply"
errors and import failures.

### Client

- Use `retryablehttp` with 3 retries, 30s timeout
- `NotFoundError` type with `client.IsNotFound(err)` helper
- `context.Context` as first parameter on every method
- Check 404 before expected status code in `doWithStatus`

### Testing

- Unit tests with `httptest` mock servers (no real Coolify instance needed)
- Wrap all mock servers with `acctest.WithVersionEndpoint(handler)` (provider health check calls /api/v1/version on Configure)
- Use `resource.UnitTest` for unit tests, `resource.Test` for acceptance tests
- Every resource needs: Create, Update, Import, Disappears tests at minimum
- Validator rejection tests for custom validators (FQDN format, cron syntax)
- Run with `-race` flag

### Documentation

- Never edit `docs/` directly; edit `templates/` instead (tfplugindocs regenerates docs/ on `make docs`)
- Custom guides go in `templates/guides/*.md.tmpl`
- Every resource needs `examples/resources/coolify_<type>/resource.tf` + `import.sh`
- Every data source needs `examples/data-sources/coolify_<type>/data-source.tf`
- Run `terraform fmt -recursive examples/` before committing HCL changes

### Code style

- `gofmt -s` for formatting (enforced by CI)
- golangci-lint v2 with 20 linters: errcheck, errorlint, govet, ineffassign, staticcheck,
  unused, misspell, bodyclose, nilerr, unconvert, wastedassign, whitespace,
  funlen (150 lines/80 statements), godox (no TODO/FIXME/HACK/XXX), dupl (250 tokens),
  gocognit (complexity 20), nestif (depth 5), forbidigo (no fmt.Print), gocritic, dupword
- errcheck excluded from test files
- No em dashes in human-facing text

## Testing

- Framework: `hashicorp/terraform-plugin-testing` with `httptest` mock servers
- 870+ tests (unit + acceptance)
- Acceptance tests are skipped unless `TF_ACC=1` is set
- Run `make ci && make testacc` before pushing (ci = build, lint, test, validate, python-test, docs-check, api-coverage-check, counts-check, vulncheck, goreleaser-check, modverify; testacc = acceptance tests against real Coolify)
- Before adding a test function, grep for its name to avoid duplicates
- **Test counts use floor rounding**: `counts-check` rounds down to the nearest 10 (e.g., 857 tests -> "850+"). When updating test counts in AGENTS.md or README.md, use the floor value, not the exact count. Setting "855+" when the actual count is 857 will fail `make ci` because 855 > floor(857/10)*10 = 850.

## CI

9 GitHub Actions jobs on push to main and PRs (GitHub-hosted ubuntu-latest):
Detect Changes, DCO (PR only), Test, Lint (includes Govulncheck + GoReleaser Check),
Validate (includes HCL fmt + Docs + Trivy + Gitleaks),
Acceptance Tests, Scenario Tests, Contract Freshness (weekly only), CI (gate).
Acceptance Tests bootstrap a fresh Coolify instance on ubuntu-latest and run the full suite.
Scenario Tests bootstrap Coolify and run `terraform test` against it.
A separate Dependabot Auto-Merge workflow auto-merges minor/patch PRs.
Format check (gofmt) is included in the Lint job via golangci-lint.

## Safety

- Never commit API tokens or real credentials (test files use `"test-token"`)
- `.gitleaks.toml` allowlists test files and examples for false positive suppression
- Passwords in example HCL must be obvious placeholders (e.g., `"change-me-in-production"`)
- Run `make ci && make testacc` locally before pushing
