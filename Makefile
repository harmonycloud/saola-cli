VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MODULE     := gitea.com/middleware-management/saola-cli
LDFLAGS    := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.GitCommit=$(GIT_COMMIT) \
	-X $(MODULE)/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: build clean tidy lint test

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
