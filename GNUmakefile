GOLANGCI_LINT_VERSION := 2.12.2
GORELEASER_MAJOR := 2

default: help

build: ## Compile the provider
	go build -v ./...

test: ## Run unit tests (race detector, coverage)
	go test -race -cover -count=1 -p 10 -timeout=10m ./...

testacc: ## Run acceptance tests (needs COOLIFY_ENDPOINT + COOLIFY_TOKEN)
	TF_ACC=1 go test -race -v -cover -count=1 -timeout=120m -p 1 -run 'TestAcc' ./...

lint: check-golangci-lint-version ## Run golangci-lint + go mod tidy check (CI-pinned golangci-lint)
	golangci-lint run ./...
	@go mod tidy && git diff --exit-code go.mod go.sum || (echo "go mod tidy produced changes"; exit 1)

fmt: ## Format code (gofmt + go mod tidy)
	gofmt -s -w .
	go mod tidy

docs: ## Regenerate documentation via tfplugindocs
	go generate ./...

validate: ## Check HCL formatting in examples/
	terraform fmt -check -recursive examples/

install: ## Install provider to local Go bin
	go install .

spec-update: ## Download latest Coolify OpenAPI spec
	curl -sL https://raw.githubusercontent.com/coollabsio/coolify/v4.x/openapi.json \
		-o testdata/specs/coolify-v4.json
	@echo "Updated testdata/specs/coolify-v4.json"

spec-check: ## Run OpenAPI spec compliance tests
	go test -race -count=1 -run 'TestClientEndpoints_SpecCompliance' ./internal/spectest/ -v

contract-extract: ## Extract contract from Coolify source (usage: make contract-extract VERSION=v4.0.1)
	scripts/extract-contract.sh $(or $(VERSION),latest)

contract-check: ## Verify client structs cover all contract fields
	go test -race -count=1 -run 'TestContractCoverage' ./internal/spectest/ -v

contract-matrix: ## Generate API contract accuracy matrix page
	python3 scripts/generate-contract-matrix.py

spec-generate: ## Regenerate OpenAPI spec from contract (idempotent: always patches from original)
	python3 scripts/generate-openapi.py \
		--contract testdata/contracts/coolify-v4.json \
		--spec testdata/specs/coolify-v4.original.json \
		--output testdata/specs/coolify-v4.json

api-coverage: ## Regenerate API_COVERAGE.md from coverage registry
	GENERATE_COVERAGE_DOC=1 go test -count=1 -run TestSpecCoverage_GenerateDoc ./internal/spectest/ -v

ci: build lint test validate docs-check api-coverage-check counts-check vulncheck goreleaser-check modverify ## Run all checks (CI also runs trivy + gitleaks security scans)

modverify: ## Verify module cache integrity against go.sum
	go mod verify

docs-check: ## Check generated docs are up to date
	@go generate ./... && git diff --exit-code || (echo "docs/ out of date: run 'make docs' and commit"; exit 1)

counts-check: ## Verify AGENTS.md resource/data source counts match provider.go
	@r_actual=$$(sed -n '/func.*Resources.*\[\]func.*resource\.Resource/,/^}/p' internal/provider/provider.go | grep -o 'New[A-Za-z]*' | wc -l | tr -d ' '); \
	d_actual=$$(sed -n '/func.*DataSources.*\[\]func.*datasource\.DataSource/,/^}/p' internal/provider/provider.go | grep -o 'New[A-Za-z]*' | wc -l | tr -d ' '); \
	r_doc=$$(grep -Eo '^[0-9]+ resources' AGENTS.md | grep -Eo '^[0-9]+'); \
	d_doc=$$(grep -Eo '[0-9]+ data sources' AGENTS.md | grep -Eo '^[0-9]+'); \
	ok=true; \
	if [ "$$r_actual" != "$$r_doc" ]; then echo "AGENTS.md says $$r_doc resources but provider.go has $$r_actual"; ok=false; fi; \
	if [ "$$d_actual" != "$$d_doc" ]; then echo "AGENTS.md says $$d_doc data sources but provider.go has $$d_actual"; ok=false; fi; \
	$$ok || (echo "Run: update AGENTS.md counts to match provider.go"; exit 1); \
	echo "Counts OK: $$r_actual resources, $$d_actual data sources"

api-coverage-check: ## Check API_COVERAGE.md is up to date
	@$(MAKE) api-coverage && git diff --exit-code API_COVERAGE.md || (echo "API_COVERAGE.md out of date: run 'make api-coverage' and commit"; exit 1)

check-golangci-lint-version: ## Verify golangci-lint version matches CI
	@version="$$(golangci-lint version 2>/dev/null || true)"; \
	if ! printf '%s\n' "$$version" | grep -q "version $(GOLANGCI_LINT_VERSION) "; then \
		echo "ERROR: golangci-lint $(GOLANGCI_LINT_VERSION) required to match CI."; \
		if [ -n "$$version" ]; then echo "Installed: $$version"; else echo "Installed: not found"; fi; \
		exit 1; \
	fi

check-goreleaser-version: ## Verify goreleaser major version matches CI
	@raw_version="$$(goreleaser --version 2>/dev/null || true)"; \
	version="$$(printf '%s\n' "$$raw_version" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)"; \
	if [ "$${version%%.*}" != "$(GORELEASER_MAJOR)" ]; then \
		echo "ERROR: goreleaser v$(GORELEASER_MAJOR).x required to match CI."; \
		if [ -n "$$version" ]; then echo "Installed: $$version"; else echo "Installed: not found"; fi; \
		exit 1; \
	fi

goreleaser-check: check-goreleaser-version ## Validate .goreleaser.yml with CI-compatible goreleaser
	goreleaser check

vulncheck: ## Run govulncheck for known vulnerabilities
	go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: build test testacc lint fmt docs docs-check api-coverage-check counts-check validate install spec-update spec-check spec-generate api-coverage contract-extract contract-check contract-matrix vulncheck check-golangci-lint-version check-goreleaser-version goreleaser-check modverify ci help