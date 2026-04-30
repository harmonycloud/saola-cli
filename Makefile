VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MODULE     := github.com/harmonycloud/saola-cli
LDFLAGS    := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.GitCommit=$(GIT_COMMIT) \
	-X $(MODULE)/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: build clean tidy lint test test-e2e fmt help

## build: compile saola binary into bin/
build:
	go build -ldflags "$(LDFLAGS)" -o bin/saola ./cmd/saola/

## clean: remove build artifacts
clean:
	rm -rf bin/

## tidy: tidy go modules
tidy:
	go mod tidy

## lint: run go vet
lint:
	go vet ./...

## test: run unit tests
test:
	go test ./... -count=1

## test-e2e: run E2E tests (requires PKG_DIR, e.g., PKG_DIR=../dataservice-baseline/clickhouse make test-e2e)
test-e2e:
	@test -n "$(PKG_DIR)" || { echo "PKG_DIR is required. Example: PKG_DIR=../dataservice-baseline/clickhouse make test-e2e"; exit 1; }
	./scripts/e2e-test.sh

## fmt: format Go source files
fmt:
	gofmt -s -w .

## help: show available targets
help:
	@grep -E '^## ' Makefile | sed 's/^## //'
