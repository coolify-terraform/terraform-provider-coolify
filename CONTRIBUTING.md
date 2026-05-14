# Contributing to terraform-provider-coolify

## Development Setup

### Prerequisites

- [Go](https://golang.org/doc/install) >= 1.26
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [golangci-lint](https://golangci-lint.run/welcome/install/) v2.12.2 exactly (must match CI)

### Clone and Build

```bash
git clone https://github.com/SebTardifLabs/terraform-provider-coolify.git
cd terraform-provider-coolify
make build
```

### Running Tests

Unit tests (no Coolify instance required):

```bash
make test
```

Acceptance tests (requires a running Coolify instance):

```bash
export COOLIFY_ENDPOINT="http://localhost:8000"
export COOLIFY_TOKEN="your-api-token"
make testacc
```

**Note**: See [TESTING.md](TESTING.md) for the full local Coolify installation
procedure, API token creation, and server validation steps.

### Code Quality

Run all CI checks locally before pushing:

```bash
make ci      # Run the full local check suite
```

Or run individual checks:

```bash
make lint              # Run golangci-lint (requires v2.12.2 exactly)
make fmt               # Format code (gofmt + go mod tidy)
make docs              # Generate documentation
make validate          # Check HCL formatting in examples/
make goreleaser-check  # Validate .goreleaser.yml (requires goreleaser v2.x)
```

**Required local tools** (match the versions pinned in CI):

- `golangci-lint` v2.12.2 exactly (match CI: `curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b ~/.local/bin v2.12.2`)
- `terraform` >= 1.6 (for `terraform fmt` on examples)
- `goreleaser` v2.x (match CI major version: `go install github.com/goreleaser/goreleaser/v2@latest`)
- `tfplugindocs` (`cd tools && go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs`)

## Project Structure

```
internal/
  client/       API client (HTTP methods for each resource type)
  provider/     Provider configuration and resource registration
  service/      One subpackage per resource type
    project/    coolify_project resource + data sources + tests
    server/     coolify_server resource + data sources + tests
    ...
  flex/         Type conversion helpers
  acctest/      Shared test utilities
```

## Adding a New Resource

1. Create a new subpackage under `internal/service/`
2. Implement the resource (resource.go) following existing patterns
3. Add data source(s) if applicable
4. Write unit tests with httptest mock servers (minimum: Create, Update, Import, Disappears)
5. Write acceptance tests in `resource_acc_test.go` (see [TESTING.md](TESTING.md))
6. Add client methods in `internal/client/`
7. Register the resource in `internal/provider/provider.go`
8. Add examples in `examples/resources/coolify_<type>/`: both `resource.tf` and `import.sh`
9. Add endpoint(s) to `coveredEndpoints()` in `internal/spectest/coverage_test.go`
10. Run `make api-coverage` to regenerate API_COVERAGE.md
11. Update resource/data source/test counts in AGENTS.md and README.md
12. Run `make docs` to generate documentation

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

3. Run Terraform without `terraform init` (dev_overrides skip the registry):
   ```bash
   export COOLIFY_ENDPOINT="http://localhost:8000"
   export COOLIFY_TOKEN="your-token"
   terraform plan
   terraform apply
   ```

4. Start a local Coolify instance for testing (see [TESTING.md](TESTING.md)
   for the full setup procedure).

## Pull Requests

- Run `make ci` before submitting (runs all checks except trivy/gitleaks security scans)
- Include tests for new functionality
- Keep PRs focused on a single change