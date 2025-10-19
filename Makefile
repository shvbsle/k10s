BINARY:=k10s
PKG:=./...
GO:=go

.PHONY: all build run test lint fmt vet snapshot release version

all: build

version:
	@cat VERSION

build:
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
	golangci-lint run

snapshot:
	goreleaser release --snapshot --clean

release:
	goreleaser release --clean
