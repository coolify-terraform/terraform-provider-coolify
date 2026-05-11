default: help

build: ## Compile the provider
	go build -v ./...

test: ## Run unit tests (race detector, coverage)
	go test -race -cover -count=1 -timeout=10m ./...

testacc: ## Run acceptance tests (needs COOLIFY_ENDPOINT + COOLIFY_TOKEN)
	TF_ACC=1 go test -v -cover -count=1 -timeout=120m -run 'TestAcc' ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

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

api-coverage: ## Regenerate API_COVERAGE.md from coverage registry
	GENERATE_COVERAGE_DOC=1 go test -count=1 -run TestSpecCoverage_GenerateDoc ./internal/spectest/ -v

ci: build fmt-check lint test validate docs-check vulncheck ## Run all checks (mirrors CI pipeline)

fmt-check: ## Check formatting (no changes)
	@output=$$(gofmt -s -l .); if [ -n "$$output" ]; then echo "gofmt needed on:"; echo "$$output"; exit 1; fi
	@go mod tidy && git diff --exit-code go.mod go.sum || (echo "go mod tidy produced changes"; exit 1)

docs-check: ## Check generated docs are up to date
	@go generate ./... && git diff --exit-code docs/ || (echo "docs/ out of date: run 'make docs' and commit"; exit 1)

vulncheck: ## Run govulncheck for known vulnerabilities
	govulncheck ./...

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: build test testacc vet lint fmt fmt-check docs docs-check validate install spec-update spec-check api-coverage vulncheck ci help