#!/bin/bash
set -euo pipefail

# saola-cli E2E Test Script
# Prerequisites: opensaola operator deployed, kubectl configured

SAOLA=${SAOLA:-./bin/saola}
PKG_DIR=${PKG_DIR:-}
PKG_NS=${PKG_NS:-middleware-operator}
NS=${NS:-e2e-test}
SAMPLES=${SAMPLES:-./docs/e2e-samples}
OPENSAOLA_DIR=${OPENSAOLA_DIR:-../opensaola}
OPERATOR_IMG=${OPERATOR_IMG:-opensaola:latest}

if [ -z "$PKG_DIR" ]; then
  echo "Usage: PKG_DIR=/path/to/middleware/package ./scripts/e2e-test.sh"
  echo ""
  echo "Example:"
  echo "  PKG_DIR=../dataservice-baseline/clickhouse ./scripts/e2e-test.sh"
  exit 1
fi

# Prerequisite checks
command -v kubectl >/dev/null 2>&1 || { echo "ERROR: kubectl not found"; exit 1; }
kubectl cluster-info >/dev/null 2>&1 || { echo "ERROR: cluster not reachable"; exit 1; }

echo "========================================="
echo "  saola-cli E2E Test"
echo "========================================="
echo "Package: $PKG_DIR"
echo "Package NS: $PKG_NS"
echo "Test NS: $NS"
echo ""

# Cleanup on exit — preserves original exit code
cleanup() {
  local exit_code=$?
  echo ""
  echo "=== Cleanup (exit code: $exit_code) ==="
  kubectl delete -f "$SAMPLES/clickhouse-middleware.yaml" --timeout=60s 2>/dev/null || true
  kubectl delete -f "$SAMPLES/clickhouse-operator.yaml" --timeout=60s 2>/dev/null || true
  kubectl wait --for=delete middleware --all -n "$NS" --timeout=60s 2>/dev/null || true
  "$SAOLA" uninstall "$PKG_NAME" --pkg-namespace "$PKG_NS" 2>/dev/null || true
  exit $exit_code
}
trap cleanup EXIT

# Build
echo "=== Step 1: Build saola ==="
make build
"$SAOLA" version

# Create namespaces
echo ""
echo "=== Step 2: Create namespaces ==="
kubectl create namespace "$PKG_NS" --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -

# Phase 0: Rebuild and redeploy opensaola operator
echo ""
echo "=== Phase 0: Rebuild and Redeploy OpenSaola Operator ==="

if [ ! -d "$OPENSAOLA_DIR" ]; then
  echo "SKIP: opensaola directory not found at $OPENSAOLA_DIR"
  echo "Set OPENSAOLA_DIR to the opensaola project path, or deploy operator manually."
else
  echo "--- 0.1: Build operator image ---"
  (cd "$OPENSAOLA_DIR" && make docker-build IMG="$OPERATOR_IMG") 2>&1 | tail -5

  echo "--- 0.2: Install CRDs ---"
  (cd "$OPENSAOLA_DIR" && make install) 2>&1 | tail -3

  echo "--- 0.3: Deploy operator ---"
  (cd "$OPENSAOLA_DIR" && make deploy IMG="$OPERATOR_IMG") 2>&1 | tail -3

  echo "--- 0.4: Wait for operator ready ---"
  kubectl wait --for=condition=available --timeout=120s \
    deploy/opensaola-controller-manager -n opensaola-system

  echo "--- 0.5: Verify operator ---"
  kubectl get pods -n opensaola-system
  kubectl logs -n opensaola-system deploy/opensaola-controller-manager --tail=5 2>&1 | grep -i 'error\|panic' || echo "NO ERRORS"
fi

# Phase A: Package management
echo ""
echo "=== Phase A: Package Management ==="
echo "--- A1: Install package ---"
"$SAOLA" install "$PKG_DIR" --pkg-namespace "$PKG_NS"

echo "--- A2: Get package ---"
"$SAOLA" get package --pkg-namespace "$PKG_NS"

PKG_NAME=$("$SAOLA" get package --pkg-namespace "$PKG_NS" -o name 2>/dev/null | head -1 | sed 's|package/||')
if [ -z "$PKG_NAME" ]; then
  echo "FAIL: Could not determine package name"
  exit 1
fi
echo "Package name: $PKG_NAME"

echo "--- A3: Inspect ---"
"$SAOLA" inspect "$PKG_NAME" --pkg-namespace "$PKG_NS" 2>&1 | head -15

echo "--- A4: Baselines ---"
"$SAOLA" get baseline --package "$PKG_NAME" --pkg-namespace "$PKG_NS"
"$SAOLA" get baseline --package "$PKG_NAME" --kind operator --pkg-namespace "$PKG_NS"
"$SAOLA" get baseline --package "$PKG_NAME" --kind action --pkg-namespace "$PKG_NS"

# Phase B: Operator deployment (if samples exist)
if [ -f "$SAMPLES/clickhouse-operator.yaml" ]; then
  echo ""
  echo "=== Phase B: Operator Deployment ==="
  echo "--- B1: Install CRDs ---"
  kubectl apply -f "$PKG_DIR/crds/" || { echo "FAIL: CRD installation failed"; exit 1; }

  echo "--- B2: Create MiddlewareOperator ---"
  kubectl apply -f "$SAMPLES/clickhouse-operator.yaml"

  echo "--- B3: Wait for Available (max 3min) ---"
  for i in $(seq 1 36); do
    STATE=$(kubectl get mo -n "$NS" -o jsonpath='{.items[0].status.state}' 2>/dev/null)
    if [ "$STATE" = "Available" ]; then
      echo "Operator Available at attempt $i"
      break
    fi
    if [ "$i" -eq 36 ]; then
      echo "FAIL: Operator not Available after 3 minutes"
      kubectl get mo -n "$NS" -o yaml 2>/dev/null || true
      exit 1
    fi
    sleep 5
  done

  echo "--- B4: Verify ---"
  "$SAOLA" get operator -n "$NS"
  kubectl get pods -n "$NS"
else
  echo "SKIP: Phase B — no operator sample at $SAMPLES/clickhouse-operator.yaml"
fi

# Phase C: Middleware deployment (if samples exist)
if [ -f "$SAMPLES/clickhouse-middleware.yaml" ]; then
  echo ""
  echo "=== Phase C: Middleware Deployment ==="
  echo "--- C1: Create Middleware ---"
  kubectl apply -f "$SAMPLES/clickhouse-middleware.yaml"

  echo "--- C2: Wait for Available (max 5min) ---"
  for i in $(seq 1 60); do
    STATE=$(kubectl get mid -n "$NS" -o jsonpath='{.items[0].status.state}' 2>/dev/null)
    if [ "$STATE" = "Available" ]; then
      echo "Middleware Available at attempt $i"
      break
    fi
    if [ "$i" -eq 60 ]; then
      echo "FAIL: Middleware not Available after 5 minutes"
      kubectl get mid -n "$NS" -o yaml 2>/dev/null || true
      exit 1
    fi
    sleep 5
  done

  echo "--- C3: Verify ---"
  "$SAOLA" get middleware -n "$NS"
  "$SAOLA" get all -n "$NS"
  kubectl get pods -n "$NS"
else
  echo "SKIP: Phase C — no middleware sample at $SAMPLES/clickhouse-middleware.yaml"
fi

# Phase D: Output formats
echo ""
echo "=== Phase D: Output Formats ==="
"$SAOLA" get all -n "$NS"
"$SAOLA" get middleware -n "$NS" -o yaml 2>&1 | head -10
"$SAOLA" get operator -n "$NS" -o json 2>&1 | head -10
"$SAOLA" --lang en get all -n "$NS"

echo ""
echo "========================================="
echo "  saola-cli E2E TEST PASSED"
echo "========================================="
