default: build

build:
	go build -v ./...

test:
	go test -v -cover -timeout=2m -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout=120m ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .
	goimports -w -local github.com/SebTardif/terraform-provider-coolify .
	go mod tidy

docs:
	go generate ./...

install:
	go install .

.PHONY: build test testacc lint fmt docs install