GOLANGCI_LINT_VERSION := 2.12.2
GORELEASER_MAJOR := 2
ACTIONLINT_VERSION := 1.7.12
BIN_DIR := $(CURDIR)/bin

export PATH := $(BIN_DIR):$(PATH)

default: help

build: ## Compile the provider
	go build -v ./...

test: ## Run unit tests (race detector, coverage)
	go test -race -cover -count=1 -p 4 -parallel=1 -timeout=15m ./...

testacc: ## Run acceptance tests (needs COOLIFY_ENDPOINT + COOLIFY_TOKEN)
	TF_ACC=1 go test -race -v -cover -count=1 -timeout=120m -p 1 -parallel=1 -run 'TestAcc' ./...

acc-bootstrap: ## Run the supported local Coolify acceptance-test bootstrap script
	./scripts/setup-coolify-test.sh

acc-preflight: ## Check required env, API reachability, and common acceptance fixtures
	@missing=0; \
	if [ -z "$$COOLIFY_ENDPOINT" ]; then echo "ERROR: COOLIFY_ENDPOINT is not set. Set it or run 'make acc-bootstrap'."; missing=1; fi; \
	if [ -z "$$COOLIFY_TOKEN" ]; then echo "ERROR: COOLIFY_TOKEN is not set. Set it or run 'make acc-bootstrap'."; missing=1; fi; \
	if [ "$$missing" -ne 0 ]; then exit 1; fi; \
	version=$$(curl -fsS -H "Authorization: Bearer $$COOLIFY_TOKEN" "$$COOLIFY_ENDPOINT/api/v1/version" 2>/dev/null || true); \
	if [ -z "$$version" ]; then echo "ERROR: Could not reach $$COOLIFY_ENDPOINT/api/v1/version with COOLIFY_TOKEN. Run 'make acc-bootstrap' or fix the local instance."; exit 1; fi; \
	echo "API OK: $$version"; \
	servers=$$(curl -fsS -H "Authorization: Bearer $$COOLIFY_TOKEN" "$$COOLIFY_ENDPOINT/api/v1/servers" 2>/dev/null || true); \
	visible_server_uuid_fields=$$(printf '%s\n' "$$servers" | grep -oE '"uuid"[[:space:]]*:[[:space:]]*"[^"]+"' | sed 's/[[:space:]]//g' || true); \
	has_visible_server=0; \
	if [ -n "$$visible_server_uuid_fields" ]; then echo "Server discovery OK: at least one server is visible."; has_visible_server=1; else echo "WARNING: No visible servers returned. Set COOLIFY_SERVER_UUID or run 'make acc-bootstrap' and validate a server."; fi; \
	if [ -n "$$COOLIFY_SERVER_UUID" ]; then \
		if [ "$$has_visible_server" -ne 1 ]; then echo "ERROR: COOLIFY_SERVER_UUID is set, but /api/v1/servers returned no visible servers."; exit 1; fi; \
		if printf '%s\n' "$$visible_server_uuid_fields" | grep -Fqx '"uuid":"'"$$COOLIFY_SERVER_UUID"'"'; then echo "Server fixture override OK: COOLIFY_SERVER_UUID is visible."; else echo "ERROR: COOLIFY_SERVER_UUID=$$COOLIFY_SERVER_UUID was not returned by /api/v1/servers. Fix the UUID or run 'make acc-bootstrap' to validate a local server."; exit 1; fi; \
	else echo "INFO: COOLIFY_SERVER_UUID not set. Acceptance helpers will auto-discover the first visible server."; fi; \
	if [ -z "$$COOLIFY_HETZNER_TOKEN" ]; then echo "WARNING: COOLIFY_HETZNER_TOKEN not set. Hetzner and cloud token acceptance packages will skip."; else echo "Hetzner fixture OK: COOLIFY_HETZNER_TOKEN is set."; fi; \
	if [ -z "$$COOLIFY_S3_STORAGE_UUID" ]; then echo "WARNING: COOLIFY_S3_STORAGE_UUID not set. S3 backup acceptance tests will skip."; else echo "S3 fixture OK: COOLIFY_S3_STORAGE_UUID is set."; fi; \
	if [ -n "$$COOLIFY_GITHUB_APP_APP_ID" ] && [ -n "$$COOLIFY_GITHUB_APP_INSTALLATION_ID" ] && [ -n "$$COOLIFY_GITHUB_APP_CLIENT_ID" ] && [ -n "$$COOLIFY_GITHUB_APP_CLIENT_SECRET" ] && [ -n "$$COOLIFY_GITHUB_APP_PRIVATE_KEY_FILE" ] && [ -n "$$COOLIFY_GITHUB_APP_REPOSITORY" ]; then echo "GitHub App fixtures OK: COOLIFY_GITHUB_APP_* is configured."; else echo "WARNING: COOLIFY_GITHUB_APP_* fixtures are incomplete. GitHub App application acceptance will skip."; fi

check-pkg: ## Verify PKG is set for package-scoped test targets
	@test -n "$(PKG)" || (echo "ERROR: PKG is required, example: make test-pkg PKG=./internal/service/project/"; exit 1)

test-pkg: check-pkg ## Run unit tests for one package (usage: make test-pkg PKG=./internal/service/project/)
	go test -race -cover -count=1 -parallel=1 -timeout=$(or $(TIMEOUT),15m) $(PKG)

testacc-pkg: check-pkg ## Run acceptance tests for one package with serialized execution (usage: make testacc-pkg PKG=./internal/service/project/ [RUN=TestAcc])
	TF_ACC=1 go test -race -v -cover -count=1 -parallel=1 -timeout=$(or $(TIMEOUT),30m) -run '$(or $(RUN),TestAcc)' $(PKG)

lint: check-golangci-lint-version ## Run golangci-lint + go mod tidy check (CI-pinned golangci-lint)
	golangci-lint run ./...
	@go mod tidy && git diff --exit-code go.mod go.sum || (echo "go mod tidy produced changes"; exit 1)

fmt: ## Format code (gofmt + go mod tidy)
	gofmt -s -w .
	go mod tidy

docs: check-tfplugindocs ## Regenerate documentation via tfplugindocs
	go generate ./...

validate: ## Check HCL formatting in examples/
	terraform fmt -check -recursive examples/

python-test: check-python3 ## Run Python unit tests for scripts/
	python3 -m unittest discover -s scripts -p '*test*.py' -v

install: ## Install provider to local Go bin
	go install .

spec-update: ## Download latest Coolify OpenAPI spec
	curl -sL https://raw.githubusercontent.com/coollabsio/coolify/v4.x/openapi.json \
		-o testdata/specs/coolify-v4.json
	@echo "Updated testdata/specs/coolify-v4.json"

spec-check: ## Run OpenAPI spec compliance tests
	go test -race -count=1 -run 'TestClientEndpoints_SpecCompliance' ./internal/spectest/ -v

contract-extract: check-python3 ## Extract contract from Coolify source (usage: make contract-extract VERSION=v4.0.1)
	scripts/extract-contract.sh $(or $(VERSION),latest)

contract-check: ## Verify client structs cover all contract fields
	go test -race -count=1 -run 'TestContractCoverage' ./internal/spectest/ -v

contract-matrix: check-python3 ## Generate API contract accuracy matrix page
	python3 scripts/generate-contract-matrix.py

spec-generate: check-python3 ## Regenerate OpenAPI spec from contract (idempotent: always patches from original)
	python3 scripts/generate-openapi.py \
		--contract testdata/contracts/coolify-v4.json \
		--spec testdata/specs/coolify-v4.original.json \
		--output testdata/specs/coolify-v4.json

api-coverage: ## Regenerate API_COVERAGE.md from coverage registry
	GENERATE_COVERAGE_DOC=1 go test -count=1 -run TestSpecCoverage_GenerateDoc ./internal/spectest/ -v

test-import-gen: ## Test terraform plan -generate-config-out compatibility (needs Coolify + TF 1.5+)
	scripts/test-import-generation.sh $(or $(TYPE),all)

scaffold: ## Scaffold a new resource (usage: make scaffold NAME=webhook)
	@./scripts/new-resource.sh $(NAME)

ci: build lint test validate actionlint-check python-test docs-check api-coverage-check counts-check vulncheck goreleaser-check modverify ## Run all checks (CI also runs trivy + gitleaks security scans)

modverify: ## Verify module cache integrity against go.sum
	go mod verify

docs-check: check-tfplugindocs ## Check generated docs are up to date
	@before=$$(mktemp); after=$$(mktemp); \
	trap 'rm -f "$$before" "$$after"' EXIT; \
	git diff -- docs/ > "$$before"; \
	go generate ./...; \
	git diff -- docs/ > "$$after"; \
	if ! cmp -s "$$before" "$$after"; then \
		echo "docs/ out of date: run 'make docs' and commit"; \
		exit 1; \
	fi

counts-check: ## Verify AGENTS.md and README.md resource/data source/test counts
	@r_actual=$$(sed -n '/func.*Resources.*\[\]func.*resource\.Resource/,/^}/p' internal/provider/provider.go | grep -o 'New[A-Za-z]*' | wc -l | tr -d ' '); \
	d_actual=$$(sed -n '/func.*DataSources.*\[\]func.*datasource\.DataSource/,/^}/p' internal/provider/provider.go | grep -o 'New[A-Za-z]*' | wc -l | tr -d ' '); \
	r_doc=$$(grep -Eo '^[0-9]+ resources' AGENTS.md | grep -Eo '^[0-9]+'); \
	d_doc=$$(grep -Eo '[0-9]+ data sources' AGENTS.md | grep -Eo '^[0-9]+'); \
	t_actual=$$(grep -r 'func Test' --include='*_test.go' . | wc -l | tr -d ' '); \
	t_floor=$$(( (t_actual / 10) * 10 )); \
	t_agents=$$(grep -Eo '[0-9]+\+ tests' AGENTS.md | head -1 | grep -Eo '^[0-9]+'); \
	t_readme=$$(grep -Eo '[0-9]+\+ tests' README.md | head -1 | grep -Eo '^[0-9]+'); \
	ok=true; \
	if [ "$$r_actual" != "$$r_doc" ]; then echo "AGENTS.md says $$r_doc resources but provider.go has $$r_actual"; ok=false; fi; \
	if [ "$$d_actual" != "$$d_doc" ]; then echo "AGENTS.md says $$d_doc data sources but provider.go has $$d_actual"; ok=false; fi; \
	if [ -n "$$t_agents" ] && [ "$$t_agents" -gt "$$t_floor" ]; then echo "AGENTS.md says $$t_agents+ tests but actual is $$t_actual (floor $$t_floor)"; ok=false; fi; \
	if [ -n "$$t_readme" ] && [ "$$t_readme" -gt "$$t_floor" ]; then echo "README.md says $$t_readme+ tests but actual is $$t_actual (floor $$t_floor)"; ok=false; fi; \
	if ! $$ok; then echo "Run: update test counts (actual: $$t_actual, floor: $$t_floor+)"; exit 1; fi; \
	echo "Counts OK: $$r_actual resources, $$d_actual data sources, $$t_actual tests ($$t_floor+ documented)"

api-coverage-check: ## Check API_COVERAGE.md is up to date
	@before=$$(mktemp); after=$$(mktemp); \
	trap 'rm -f "$$before" "$$after"' EXIT; \
	git diff -- API_COVERAGE.md > "$$before"; \
	$(MAKE) api-coverage; \
	git diff -- API_COVERAGE.md > "$$after"; \
	if ! cmp -s "$$before" "$$after"; then \
		echo "API_COVERAGE.md out of date: run 'make api-coverage' and commit"; \
		exit 1; \
	fi

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

check-python3: ## Verify Python 3 is installed for Python-backed tooling
	@command -v python3 >/dev/null 2>&1 || (echo "ERROR: python3 is required for Python-backed Make targets in this repo. Install Python 3.9+ and re-run."; exit 1)

check-actionlint-version: ## Verify actionlint version matches CI
	@version="$$(actionlint --version 2>/dev/null | awk 'NR == 1 {print $$1}')"; \
	if [ "$$version" != "$(ACTIONLINT_VERSION)" ]; then \
		echo "ERROR: actionlint $(ACTIONLINT_VERSION) required to match CI. Install with: make tools"; \
		if [ -n "$$version" ]; then echo "Installed: $$version"; else echo "Installed: not found"; fi; \
		exit 1; \
	fi

actionlint-check: check-actionlint-version ## Lint GitHub Actions workflows
	actionlint

check-tfplugindocs: ## Verify tfplugindocs is installed for docs generation
	@command -v tfplugindocs >/dev/null 2>&1 || (echo "ERROR: tfplugindocs is required for docs generation. Install with: make tools"; exit 1)

goreleaser-check: check-goreleaser-version ## Validate .goreleaser.yml with CI-compatible goreleaser
	goreleaser check

vulncheck: ## Run govulncheck for known vulnerabilities
	go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...

tools: ## Install all required development tools
	@mkdir -p "$(BIN_DIR)"
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) to $(BIN_DIR)..."
	@GOBIN="$(BIN_DIR)" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GOLANGCI_LINT_VERSION)
	@echo "Installing goreleaser to $(BIN_DIR)..."
	@GOBIN="$(BIN_DIR)" go install github.com/goreleaser/goreleaser/v$(GORELEASER_MAJOR)@latest
	@echo "Installing actionlint $(ACTIONLINT_VERSION) to $(BIN_DIR)..."
	@GOBIN="$(BIN_DIR)" go install github.com/rhysd/actionlint/cmd/actionlint@v$(ACTIONLINT_VERSION)
	@echo "Installing tfplugindocs to $(BIN_DIR)..."
	@cd tools && GOBIN="$(BIN_DIR)" go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
	@echo "All tools installed to $(BIN_DIR)."

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: build test testacc acc-bootstrap acc-preflight check-pkg test-pkg testacc-pkg lint fmt docs docs-check api-coverage-check counts-check validate python-test install spec-update spec-check spec-generate api-coverage contract-extract contract-check contract-matrix vulncheck check-golangci-lint-version check-goreleaser-version check-python3 check-actionlint-version check-tfplugindocs actionlint-check goreleaser-check modverify ci scaffold test-import-gen tools help