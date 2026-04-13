# Saola CLI Testing Guide

**English** | [中文](testing-guide_zh.md)

This guide covers unit testing, building, end-to-end (E2E) testing, and CI checks for saola-cli.

## 1. Unit Tests

```bash
# Run all unit tests
make test

# Run a specific package with verbose output
go test ./internal/packager/... -v
go test ./internal/tarutil/... -v -run TestReadTarInfo
go test ./internal/printer/... -v
go test ./internal/lang/... -v
```

### Package Coverage Matrix

| Package | Has Tests | Notes |
|---------|-----------|-------|
| `internal/packager` | Yes | TAR/zstd packaging logic |
| `internal/tarutil` | Yes | TAR read/write helpers |
| `internal/printer` | Yes | Output formatting (table/yaml/json) |
| `internal/lang` | Yes | Bilingual message lookup |
| `internal/config` | Yes | Configuration management |
| `internal/cmdutil` | Yes | Command utility helpers |
| `internal/k8s` | Yes | Kubernetes helpers |
| `internal/version` | Yes | Version info |
| `internal/waiter` | Yes | Async wait logic |
| `internal/cmd/pkgcmd` | Yes | Package management commands |
| `internal/cmd/operator` | Yes | Operator commands |
| `internal/cmd/middleware` | Yes | Middleware commands |
| `internal/cmd/create` | Yes | Resource creation |
| `internal/cmd/baseline` | Yes | Baseline queries |
| `internal/cmd/action` | Yes | Action commands |
| `cmd/saola` | No | Main entrypoint |
| `internal/client` | No | Kubernetes client wrapper (requires cluster) |
| `internal/app` | No | Root command registration |
| `internal/consts` | No | Constants only |
| `internal/packages` | No | Package definitions |
| `internal/cmd/build` | No | Build command (requires filesystem) |
| `internal/cmd/delete` | No | Delete command (requires cluster) |
| `internal/cmd/describe` | No | Describe command (requires cluster) |
| `internal/cmd/get` | No | Get command (requires cluster) |
| `internal/cmd/inspect` | No | Inspect command (requires cluster) |
| `internal/cmd/install` | No | Install command (requires cluster) |
| `internal/cmd/uninstall` | No | Uninstall command (requires cluster) |
| `internal/cmd/upgrade` | No | Upgrade command (requires cluster) |
| `internal/cmd/resource` | No | Resource command |
| `internal/cmd/run` | No | Run command (requires cluster) |
| `internal/cmd/version` | No | Version command |

## 2. Build

```bash
# Build the binary
make build

# Verify the build
./bin/saola version
```

Expected output:

```
Version: v0.1.0
Git Commit: <commit-hash>
Build Date: <timestamp>
```

## 3. E2E Test Procedure

### Prerequisites

- A running Kubernetes cluster with `kubectl` access
- OpenSaola operator deployed (see [OpenSaola docs](https://gitee.com/opensaola/opensaola))
- A ClickHouse package directory (from [dataservice-baseline](https://gitee.com/opensaola/dataservice-baseline))

### Environment Setup

```bash
SAOLA=./bin/saola
PKG_DIR=/path/to/dataservice-baseline/clickhouse
PKG_NS=middleware-operator
NS=e2e-test

# Build the CLI
make build
```

### Step 1: Create Namespaces

```bash
kubectl create namespace $PKG_NS --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace $NS --dry-run=client -o yaml | kubectl apply -f -
```

### Step 2: Build Package

```bash
$SAOLA build $PKG_DIR
```

### Step 3: Install Package

```bash
$SAOLA install $PKG_DIR --pkg-namespace $PKG_NS
```

### Step 4: Verify Package Installation

```bash
# List installed packages
$SAOLA get package --pkg-namespace $PKG_NS

# Inspect package contents
$SAOLA inspect <pkg-name> --pkg-namespace $PKG_NS

# List baselines from package
$SAOLA get baseline --package <pkg-name> --pkg-namespace $PKG_NS

# Filter baselines by kind
$SAOLA get baseline --package <pkg-name> --kind operator --pkg-namespace $PKG_NS
```

Replace `<pkg-name>` with the actual package name from `get package` output (e.g., `clickhouse-0.25.5-1.0.0`).

### Step 5: Install ClickHouse CRDs

```bash
kubectl apply -f $PKG_DIR/crds/
```

### Step 6: Create MiddlewareOperator

```bash
kubectl apply -f docs/e2e-samples/clickhouse-operator.yaml
```

### Step 7: Verify Operator

```bash
# Watch operator status
kubectl get mo -n $NS -w

# List operators via saola
$SAOLA get operator -n $NS

# Describe operator details
$SAOLA describe operator clickhouse-operator -n $NS

# Verify operator deployment and pods
kubectl get deploy -n $NS
kubectl get pods -n $NS
```

Wait until the operator reaches the `Running` state.

### Step 8: Create Middleware Instance

```bash
kubectl apply -f docs/e2e-samples/clickhouse-middleware.yaml
```

### Step 9: Verify Middleware

```bash
# Watch middleware status
kubectl get mid -n $NS -w

# List middleware via saola
$SAOLA get middleware -n $NS

# Describe middleware details
$SAOLA describe middleware my-clickhouse -n $NS

# Verify pods and ClickHouseInstallation CR
kubectl get pods -n $NS
kubectl get chi -n $NS
```

Wait until the middleware reaches the `Running` state.

### Step 10: Test Output Formats

```bash
# Aggregate view
$SAOLA get all -n $NS

# YAML output
$SAOLA get middleware -n $NS -o yaml

# JSON output
$SAOLA get middleware -n $NS -o json

# Wide output
$SAOLA get operator -n $NS -o wide
```

### Step 11: Cleanup

```bash
# Delete middleware instance
$SAOLA delete middleware my-clickhouse -n $NS

# Delete operator
$SAOLA delete operator clickhouse-operator -n $NS

# Uninstall package
$SAOLA uninstall <pkg-name> --pkg-namespace $PKG_NS

# Remove test namespace
kubectl delete namespace $NS
```

## 4. CI Checks

Run the following before submitting a PR:

```bash
make lint     # go vet (basic correctness checks)
make test     # unit tests
make build    # verify compilation
```

All three commands must pass with zero errors.

## 5. Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `make build` fails with missing module | Dependency not fetched | Run `make tidy` first |
| `saola get package` returns empty | Package not installed or wrong namespace | Check `--pkg-namespace` flag |
| Operator stuck in `Pending` | CRDs not installed | Run `kubectl apply -f $PKG_DIR/crds/` |
| Middleware stuck in `Pending` | Operator not ready | Wait for operator to reach `Running` state |
| `connection refused` errors | kubeconfig not set | Check `KUBECONFIG` env var or `--kubeconfig` flag |
