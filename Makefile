BINARY:=k10s
PKG:=./...
GO:=go

.PHONY: all build run test lint fmt vet check snapshot release version tidy help

all: build

version: ## Display the current version
	@cat VERSION

check: tidy fmt vet lint ## Run quality checks

build: check test ## Build k10s
	$(GO) build -trimpath -ldflags="-s -w -X $(MOD)/internal/core.Version=$$(git describe --tags --always --dirty)" -o bin/$(BINARY) ./cmd/k10s

run: build ## Run k10s
	./bin/$(BINARY)

test: ## Test k10s
	$(GO) test -race $(PKG) -cover

fmt: ## Format code
	$(GO) fmt $(PKG)

vet: ## Vet code
	$(GO) vet $(PKG)

tidy: ## Tidy Go modules
	$(GO) mod tidy -e

lint: ## Lint code
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	golangci-lint run

snapshot: ## Create a snapshot release
	goreleaser release --snapshot --clean

release: ## Create a release
	goreleaser release --clean

help: ## Display help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[\.a-zA-Z_0-9\-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
