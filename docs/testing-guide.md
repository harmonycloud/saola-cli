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
| `internal/client` | Yes | Kubernetes client wrapper |
| `internal/app` | Yes | Root command registration |
| `internal/consts` | Yes | Constants |
| `internal/packages` | Yes | Package definitions |
| `internal/cmd/build` | Yes | Build command |
| `internal/cmd/delete` | Yes | Delete command |
| `internal/cmd/describe` | Yes | Describe command |
| `internal/cmd/get` | Yes | Get command |
| `internal/cmd/inspect` | Yes | Inspect command |
| `internal/cmd/install` | Yes | Install command |
| `internal/cmd/uninstall` | Yes | Uninstall command |
| `internal/cmd/upgrade` | Yes | Upgrade command |
| `internal/cmd/resource` | Yes | Resource command |
| `internal/cmd/run` | Yes | Run command |
| `internal/cmd/version` | Yes | Version command |
| `cmd/saola` | No | Main entrypoint |

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

### Automatic Operator Rebuild

The E2E script can automatically rebuild and redeploy the opensaola operator before running tests. Set `OPENSAOLA_DIR` to the opensaola project path:

```bash
OPENSAOLA_DIR=../opensaola PKG_DIR=../dataservice-baseline/clickhouse ./scripts/e2e-test.sh
```

If `OPENSAOLA_DIR` is not set or the directory doesn't exist, the script will skip operator rebuild and assume it's already deployed.

Environment variables:
| Variable | Default | Description |
|----------|---------|-------------|
| `OPENSAOLA_DIR` | `../opensaola` | Path to opensaola project (for auto-rebuild) |
| `OPERATOR_IMG` | `opensaola:latest` | Operator image tag |
| `PKG_DIR` | (required) | Middleware package directory |
| `PKG_NS` | `middleware-operator` | Package namespace |
| `NS` | `e2e-test` | Test namespace |

### Prerequisites

1. **Deploy opensaola operator**:
   ```bash
   cd ../opensaola
   make docker-build IMG=opensaola:test
   make install
   make deploy IMG=opensaola:test
   kubectl wait --for=condition=available --timeout=120s deploy/opensaola-controller-manager -n opensaola-system
   ```
   For detailed instructions, see the [opensaola Testing Guide](https://gitee.com/opensaola/opensaola/blob/master/docs/testing-guide.md#build-and-deploy-for-testing).

2. **A middleware package directory** (e.g., `../dataservice-baseline/clickhouse`)

3. **kubectl** configured with cluster access

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

### Step 11: Delete Lifecycle

```bash
# Delete middleware
$SAOLA delete middleware my-clickhouse -n $NS
# Wait and verify
sleep 30
kubectl get mid -n $NS   # should be empty
kubectl get pods -n $NS  # middleware pods should be gone

# Delete operator
$SAOLA delete operator clickhouse-operator -n $NS
sleep 15
kubectl get mo -n $NS    # should be empty

# Uninstall package
$SAOLA uninstall $PKG_NAME --pkg-namespace $PKG_NS
kubectl get secrets -n $PKG_NS -l middleware.cn/project=opensaola  # should show uninstall annotation

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
