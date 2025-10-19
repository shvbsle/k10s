BINARY:=k10s
PKG:=./...
GO:=go

.PHONY: all build run test lint fmt vet check snapshot release version

all: build

version:
	@cat VERSION

# Run quality checks before building
check: fmt vet lint

build: check
	$(GO) build -trimpath -ldflags="-s -w -X $(MOD)/internal/core.Version=$$(git describe --tags --always --dirty)" -o bin/$(BINARY) ./cmd/k10s

run: build
	./bin/$(BINARY)

test:
	$(GO) test $(PKG) -cover

fmt:
	$(GO) fmt $(PKG)

vet:
	$(GO) vet $(PKG)

lint:
	@which golangci-lint > /dev/null || (echo "Error: golangci-lint not found. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

snapshot:
	goreleaser release --snapshot --clean

release:
	goreleaser release --clean
