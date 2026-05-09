default: build

build:
	go build -v ./...

test:
	go test -race -cover -count=1 -timeout=10m ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout=120m ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .
	go mod tidy

docs:
	go generate ./...

validate:
	terraform fmt -check -recursive examples/

install:
	go install .

.PHONY: build test testacc lint fmt docs validate install