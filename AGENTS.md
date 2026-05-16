# AGENTS.md

## Skills

Read these skills when working in this repo:

- `~/.grok/skills/coolify-test-instance/SKILL.md` — Local Coolify setup, API quirks, SSH validation, acc test troubleshooting. **Read first** when setting up a test instance or debugging real-API failures.
- `~/.grok/skills/terraform-provider-coolify-contrib/SKILL.md` — Contributing conventions for the Coolify upstream project, repo namespace notes, CI gotchas.
- `~/.grok/skills/terraform-provider/SKILL.md` — General Terraform provider patterns (resource implementation, testing, CI, releases).
- `~/.grok/skills/self-hosted-runner/SKILL.md` — CI runner setup, Trivy DB mirrors, tool installation gotchas.

## Project

Terraform provider for [Coolify](https://coolify.io/), the open-source self-hosted PaaS.
Built with Go 1.26, Terraform Plugin Framework v1.19, and GoReleaser for releases.
25 resources, 44 data sources, 570+ tests (unit + acceptance), 13 CI jobs.
8 ACME Corp scenario examples with `terraform test` integration tests.

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
- **Note**: `make ci` does NOT include acceptance tests. Run acc tests for changed packages: `TF_ACC=1 go test -race -v -count=1 -timeout=10m ./internal/service/<changed-package>/`
- Use `make testacc` for the full suite when changing shared code (client, provider, flex, validate)
- Build: `make build`
- Test (all, with race detector): `make test`
- Test (single package): `go test -race -count=1 -timeout=5m ./internal/service/project/`
- Acceptance tests (needs running Coolify): `make testacc`
- Lint: `make lint`
- Format: `make fmt`
- Generate docs: `make docs`
- Validate HCL examples: `make validate`
- Install locally: `make install`
- Spec compliance: `make spec-check`
- API coverage doc: `make api-coverage`
- Extract contract from Coolify source: `make contract-extract VERSION=v4.0.1`
- Verify client structs cover contract: `make contract-check`
- Regenerate OpenAPI spec from contract: `make spec-generate`

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
- Mark passwords and keys as `Sensitive: true`
- Wrap all client errors: `fmt.Errorf("getting project %s: %w", uuid, err)`

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
  funlen (150 lines/80 statements), godox (no FIXME/HACK/XXX), dupl (200 tokens),
  gocognit (complexity 20), nestif (depth 5), forbidigo (no fmt.Print), gocritic, dupword
- errcheck excluded from test files
- No em dashes in human-facing text

## Testing

- Framework: `hashicorp/terraform-plugin-testing` with `httptest` mock servers
- 570+ tests (unit + acceptance)
- Acceptance tests are skipped unless `TF_ACC=1` is set
- Run `make ci && make testacc` before pushing (ci = build, lint, test, validate, docs-check, api-coverage-check, vulncheck; testacc = acceptance tests against real Coolify)
- Before adding a test function, grep for its name to avoid duplicates

## CI

13 GitHub Actions jobs on push to main and PRs (self-hosted runner):
Detect Changes, Test, Lint, Validate Examples, Docs, Govulncheck,
Trivy, Gitleaks, GoReleaser Check, Scenario Tests, Acceptance Tests,
Spec Freshness (weekly only), CI (gate).
A separate Dependabot Auto-Merge workflow auto-merges minor/patch PRs.
Format check (gofmt) is included in the Lint job via golangci-lint.
Scenario Tests run `terraform test` against real Coolify (requires secrets).

## Safety

- Never commit API tokens or real credentials (test files use `"test-token"`)
- `.gitleaks.toml` allowlists test files and examples for false positive suppression
- Passwords in example HCL must be obvious placeholders (e.g., `"change-me-in-production"`)
- Run `make ci && make testacc` locally before pushing
