# Contributing to terraform-provider-coolify

## Development Setup

### Prerequisites

- [Go](https://golang.org/doc/install) >= 1.26
- [Python](https://www.python.org/downloads/) >= 3.9 (required for script unit tests run by `make ci`)
- [Terraform](https://www.terraform.io/downloads.html) >= 1.6 (required for `terraform fmt` validation)
- [golangci-lint](https://golangci-lint.run/welcome/install/) v2.12.2 exactly (must match CI)

### Clone and Build

```bash
git clone https://github.com/coolify-terraform/terraform-provider-coolify.git
cd terraform-provider-coolify
make build
```

If you plan to run `make ci`, install the required local tools first:

```bash
make tools
```

This installs `golangci-lint`, `goreleaser`, and `tfplugindocs` in one step. Python remains a separate prerequisite for `make ci`, `make python-test`, and the other Python-backed tooling targets. On a fresh clone, `make ci` fails early if a required tool is missing. Run `make help` to see every supported local target from [GNUmakefile](GNUmakefile).

### Running Tests

Unit tests (no Coolify instance required):

```bash
make test
```

Acceptance tests (requires a running local Coolify instance):

```bash
make acc-bootstrap
# Copy the printed COOLIFY_* exports into your shell, then:
make acc-preflight
make testacc
make testacc-pkg PKG=./internal/service/application/
```

`make acc-bootstrap` builds on the local instance setup documented in
[TESTING.md](TESTING.md). It does not install Coolify itself. It finishes the
repo-supported local fixture setup by enabling the API, generating a token,
validating the default server, and creating the local S3 backup fixture.

`make acc-preflight` requires `COOLIFY_ENDPOINT` and `COOLIFY_TOKEN`,
auto-discovers a visible server unless `COOLIFY_SERVER_UUID` is set, and fails
if that override UUID is not returned by `/api/v1/servers`. It also warns when
optional fixtures are missing. `COOLIFY_HETZNER_TOKEN` gates the cloud token
and Hetzner acceptance packages. `COOLIFY_S3_STORAGE_UUID` gates S3 backup
coverage. `COOLIFY_GITHUB_APP_*` gates the GitHub App application acceptance
test.

**Note**: See [TESTING.md](TESTING.md) for the full local Coolify installation
procedure, API token creation, and server validation steps.

### Code Quality

Run the aggregate local checks before pushing:

```bash
make ci
```

From [GNUmakefile](GNUmakefile), `make ci` runs these local targets:
- `build`
- `lint`
- `test`
- `validate`
- `actionlint-check`
- `python-test`
- `docs-check`
- `api-coverage-check`
- `counts-check`
- `contract-compat`
- `vulncheck`
- `goreleaser-check`
- `modverify`

`make ci` does not run acceptance tests or the CI-only security scanners
(`trivy` and `gitleaks`) from the GitHub Actions pipeline. If your change
touches real Coolify API behavior, also run `make acc-preflight`, then
`make testacc` or `make testacc-pkg PKG=./internal/service/<package>/`.
Both Make targets serialize acceptance execution to avoid API overload and
macOS TSAN/fork crashes. `make testacc` serializes packages and in-package
execution; `make testacc-pkg` serializes in-package execution for one package.
The lower-level `TF_ACC=1 go test ...` equivalents are documented in
[TESTING.md](TESTING.md).

Or run individual checks:

```bash
make lint              # Run golangci-lint (requires v2.12.2 exactly)
make fmt               # Format code (gofmt + go mod tidy)
make docs              # Generate documentation
make validate          # Check HCL formatting in examples/
make goreleaser-check  # Validate .goreleaser.yml (requires goreleaser v2.x)
```

**Required local tools:** Run `make tools` to install `golangci-lint`, `goreleaser`, and `tfplugindocs` automatically. You also need `terraform` >= 1.6 (for `terraform fmt` on examples).

## Project Structure

```
internal/
  client/       API client (HTTP methods for each resource type)
  provider/     Provider configuration and resource registration
  service/      One subpackage per resource type
    project/    coolify_project resource + data sources + tests
    server/     coolify_server resource + data sources + tests
    ...
  flex/         Type conversion helpers (incl. Configure helpers)
  acctest/      Shared test utilities
  validate/     Input validators (UUID format, FQDN URL, cron)
  filter/       Generic data source filtering helpers
  spectest/     OpenAPI spec compliance and contract coverage tests
```

## Adding a New Resource

Use the scaffold script to generate boilerplate:

```bash
make scaffold NAME=webhook
```

This creates the resource, data source, client methods, unit tests,
acceptance tests, and examples with TODO placeholders. Then complete the
remaining manual steps printed by the script: register in provider.go,
fill in TODOs, customize the acceptance tests in `resource_acc_test.go`,
and run `make ci`.

<details>
<summary>Manual steps (if not using the scaffold)</summary>

1. Create a new subpackage under `internal/service/`
2. Implement the resource (resource.go) following existing patterns
3. Add data source(s) if applicable
4. Write unit tests with httptest mock servers (minimum: Create, Update, Import, Disappears)
5. Write acceptance tests in `resource_acc_test.go` and, for data sources,
   `data_source_acc_test.go` (see [TESTING.md](TESTING.md))
6. Add client methods in `internal/client/`
7. Register the resource in `internal/provider/provider.go`
8. Add examples in `examples/resources/coolify_<type>/`: both `resource.tf` and `import.sh`
9. Add endpoint(s) to `coveredEndpoints()` in `internal/spectest/coverage_test.go`
10. Run `make api-coverage` to regenerate API_COVERAGE.md
11. Update resource/data source/test counts in AGENTS.md and README.md
12. Run `make docs` to generate documentation

</details>

## Style Guide

- Use `terraform-plugin-framework` (not SDK v2)
- One file per resource/data source
- Co-locate tests with implementation (`_test.go`)
- Wrap all client errors with context: `fmt.Errorf("verb resource %s: %w", id, err)`
- Handle 404 gracefully in Read (remove from state) and Delete (silently succeed)
- Use `MarkdownDescription` on all schema attributes
- Implement `ImportState` for all resources

## Local Provider Testing (dev_overrides)

To test the provider against a real Coolify instance without publishing:

1. Build and install locally:
   ```bash
   go install .
   ```

2. Create or edit `~/.terraformrc`:
   ```hcl
   provider_installation {
     dev_overrides {
       "coolify-terraform/coolify" = "/home/YOUR_USER/go/bin"
     }
     direct {}
   }
   ```
   Replace `/home/YOUR_USER/go/bin` with your `$GOPATH/bin` (run `go env GOPATH` to find it).

3. Run Terraform commands with the local override in place:
   ```bash
   export COOLIFY_ENDPOINT="http://localhost:8000"
   export COOLIFY_TOKEN="your-token"
   terraform plan
   terraform apply
   ```

   If the configuration uses local modules, run `terraform get` first.
   `terraform init` still tries to resolve providers even with `dev_overrides`.

4. Start a local Coolify instance for testing (see [TESTING.md](TESTING.md)
   for the full setup procedure).

## DCO Sign-Off

All commits must carry a `Signed-off-by` trailer (the
[Developer Certificate of Origin](https://developercertificate.org/)). This is
enforced by CI on every pull request.

Add the sign-off automatically with the `-s` flag:

```bash
git commit -s -m "Add widget resource"
```

If you forgot on an existing commit, amend it:

```bash
git commit --amend -s --no-edit
```

For multiple unsigned commits on a branch:

```bash
git rebase --signoff HEAD~N   # N = number of commits to fix
```

## Issue Triage

Issues are triaged using three labels:

| Label | Meaning |
|-------|---------|
| `needs-triage` | Unreviewed, waiting for maintainer review |
| `ready` | Accepted and available for implementation |
| `needs-info` | Blocked on additional information from the reporter |

A GitHub Actions workflow automatically labels new issues:
- Issues from OWNER / MEMBER / COLLABORATOR authors get `ready` immediately.
- Issues from external authors get `needs-triage` and wait for maintainer review.

Maintainers accept an issue by replacing `needs-triage` with `ready`:

```bash
gh issue edit N --add-label ready --remove-label needs-triage
```

### What gets implemented

Only issues labeled `ready` (or legacy unlabeled issues) are in scope for
implementation. Issues labeled `needs-triage` or `needs-info` are never
auto-implemented.

When working on a `ready` issue, scope is limited to the issue title, body,
and comments from the issue creator and maintainers. Comments from other
users do not expand the PR scope (they may be used as hints for the same
bug, but additional feature requests should be filed as separate issues).

## Pull Requests

- Run `make ci` before submitting, and add `make testacc` or targeted `TF_ACC=1 go test ...` commands when your change touches real Coolify API behavior (`make ci` still skips trivy, gitleaks, and acceptance tests)
- Include tests for new functionality
- Keep PRs focused on a single change
- All commits must have a DCO sign-off (see above)

## Coding Standards

This project follows idiomatic Go conventions enforced by automated tooling:

- **Formatting**: `gofmt -s` (enforced by CI via golangci-lint)
- **Linting**: golangci-lint v2 with 20 linters including errcheck, govet, staticcheck,
  funlen (150 lines / 80 statements), gocognit (complexity 20), nestif (depth 5),
  dupl (250 tokens), and forbidigo (no fmt.Print)
- **Error handling**: Wrap all errors with context using `fmt.Errorf("verb resource %s: %w", id, err)`
- **Framework**: Use `terraform-plugin-framework` (not SDK v2)
- **File organization**: One file per resource/data source, co-locate tests
- **Documentation**: Use `MarkdownDescription` on all schema attributes
- **Security**: Mark passwords and keys as `Sensitive: true`, never commit real credentials

## Test Policy

All new features and bug fixes require tests:

- **Unit tests** use `httptest` mock servers (no Coolify instance needed)
- **Acceptance tests** run against a real Coolify instance (`TF_ACC=1`)
- **Minimum test coverage**: Create, Update, Import, and Disappears tests for every resource
- **Race detection**: All tests run with the `-race` flag
- **CI enforcement**: Tests must pass in CI before merge; coverage is reported via
  GitHub native code coverage