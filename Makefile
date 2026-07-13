VERSION           ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GIT_COMMIT        ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
SOURCE_DATE_EPOCH ?= $(shell git show -s --format=%ct HEAD 2>/dev/null || echo 0)
BUILD_DATE        ?= $(shell date -u -d "@$(SOURCE_DATE_EPOCH)" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -r "$(SOURCE_DATE_EPOCH)" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown)
DIST_DIR          ?= dist
MODULE     := github.com/harmonycloud/saola-cli
LDFLAGS    := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.GitCommit=$(GIT_COMMIT) \
	-X $(MODULE)/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: build release-build release-checksums clean tidy lint test test-e2e fmt help

## build: compile saola binary into bin/
build:
	go build -ldflags "$(LDFLAGS)" -o bin/saola ./cmd/saola/

## release-build: reproducibly build static Linux binaries for amd64 and arm64 into dist/
release-build:
	@test "$(GIT_COMMIT)" != "unknown" || { echo "GIT_COMMIT must be a full commit SHA" >&2; exit 1; }
	@printf '%s\n' "$(GIT_COMMIT)" | grep -Eq '^[0-9a-f]{40}$$' || { echo "GIT_COMMIT must be a full lowercase 40-character SHA" >&2; exit 1; }
	@test "$(BUILD_DATE)" != "unknown" || { echo "BUILD_DATE could not be derived from SOURCE_DATE_EPOCH" >&2; exit 1; }
	rm -rf "$(DIST_DIR)"
	mkdir -p "$(DIST_DIR)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags "$(LDFLAGS)" -o "$(DIST_DIR)/saola-linux-amd64" ./cmd/saola/
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -buildvcs=false -ldflags "$(LDFLAGS)" -o "$(DIST_DIR)/saola-linux-arm64" ./cmd/saola/

## release-checksums: write sorted SHA-256 checksums for both release binaries
release-checksums:
	@test -f "$(DIST_DIR)/saola-linux-amd64" -a -f "$(DIST_DIR)/saola-linux-arm64" || { echo "run make release-build first" >&2; exit 1; }
	@cd "$(DIST_DIR)" && if command -v sha256sum >/dev/null 2>&1; then \
		sha256sum saola-linux-amd64 saola-linux-arm64 > SHA256SUMS; \
	else \
		shasum -a 256 saola-linux-amd64 saola-linux-arm64 > SHA256SUMS; \
	fi

## clean: remove build artifacts
clean:
	rm -rf bin/ "$(DIST_DIR)/"

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
