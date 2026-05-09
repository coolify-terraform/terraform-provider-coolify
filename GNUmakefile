default: help

build: ## Compile the provider
	go build -v ./...

test: ## Run unit tests (race detector, coverage)
	go test -race -cover -count=1 -timeout=10m ./...

testacc: ## Run acceptance tests (needs COOLIFY_ENDPOINT + COOLIFY_TOKEN)
	TF_ACC=1 go test -v -cover -timeout=120m ./...

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

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: build test testacc lint fmt docs validate install help