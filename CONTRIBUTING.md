# Contributing to terraform-provider-coolify

## Development Setup

### Prerequisites

- [Go](https://golang.org/doc/install) >= 1.24
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [golangci-lint](https://golangci-lint.run/welcome/install/) v2.x

### Clone and Build

```bash
git clone https://github.com/SebTardif/terraform-provider-coolify.git
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
cp .env.example .env
# Edit .env with your Coolify endpoint and token
source .env
make testacc
```

### Code Quality

```bash
make lint    # Run golangci-lint
make fmt     # Format code
make docs    # Generate documentation
```

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
4. Write unit tests with httptest mock servers
5. Add client methods in `internal/client/`
6. Register the resource in `internal/provider/provider.go`
7. Add an example in `examples/resources/`
8. Run `make docs` to generate documentation

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
       "SebTardif/coolify" = "/home/YOUR_USER/go/bin"
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

4. Start a local Coolify instance for testing:
   ```bash
   docker compose up -d
   # Wait for startup, then create an API token in the Coolify UI
   ```

## Pull Requests

- Run `make test` and `make lint` before submitting
- Include tests for new functionality
- Keep PRs focused on a single change