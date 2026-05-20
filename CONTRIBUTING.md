# Contributing to terraform-provider-coolify

## Development Setup

### Prerequisites

- [Go](https://golang.org/doc/install) >= 1.26
- [Python](https://www.python.org/downloads/) >= 3.9 (required for script unit tests run by `make ci`)
- [Terraform](https://www.terraform.io/downloads.html) >= 1.6 (required for `terraform fmt` validation)
- [golangci-lint](https://golangci-lint.run/welcome/install/) v2.12.2 exactly (must match CI)

### Clone and Build

```bash
git clone https://github.com/SebTardifLabs/terraform-provider-coolify.git
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
- `python-test`
- `docs-check`
- `api-coverage-check`
- `counts-check`
- `vulncheck`
- `goreleaser-check`
- `modverify`

`make ci` does not run acceptance tests or the CI-only security scanners
(`trivy` and `gitleaks`) from the GitHub Actions pipeline. If your change
touches real Coolify API behavior, also run `make acc-preflight`, then
`make testacc` or `make testacc-pkg PKG=./internal/service/<package>/`.
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

This creates the resource, data source, client methods, unit tests, and
examples with TODO placeholders. Then complete the remaining manual steps
printed by the script, especially adding `resource_acc_test.go` and, when
needed, `data_source_acc_test.go`, registering in provider.go, filling in
TODOs, and running `make ci`.

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
       "SebTardifLabs/coolify" = "/home/YOUR_USER/go/bin"
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

## Pull Requests

- Run `make ci` before submitting, and add `make testacc` or targeted `TF_ACC=1 go test ...` commands when your change touches real Coolify API behavior (`make ci` still skips trivy, gitleaks, and acceptance tests)
- Include tests for new functionality
- Keep PRs focused on a single change