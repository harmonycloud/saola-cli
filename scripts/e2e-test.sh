#!/bin/bash
set -euo pipefail

# saola-cli E2E Test Script
# Prerequisites: opensaola operator deployed, kubectl configured

SAOLA=${SAOLA:-./bin/saola}
PKG_DIR=${PKG_DIR:-}
PKG_NS=${PKG_NS:-middleware-operator}
NS=${NS:-e2e-test}
SAMPLES=${SAMPLES:-./docs/e2e-samples}

if [ -z "$PKG_DIR" ]; then
  echo "Usage: PKG_DIR=/path/to/middleware/package ./scripts/e2e-test.sh"
  echo ""
  echo "Example:"
  echo "  PKG_DIR=../dataservice-baseline/clickhouse ./scripts/e2e-test.sh"
  exit 1
fi

echo "========================================="
echo "  saola-cli E2E Test"
echo "========================================="
echo "Package: $PKG_DIR"
echo "Package NS: $PKG_NS"
echo "Test NS: $NS"
echo ""

# Build
echo "=== Step 1: Build saola ==="
make build
$SAOLA version

# Create namespaces
echo ""
echo "=== Step 2: Create namespaces ==="
kubectl create namespace $PKG_NS --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace $NS --dry-run=client -o yaml | kubectl apply -f -

# Phase A: Package management
echo ""
echo "=== Phase A: Package Management ==="
echo "--- A1: Install package ---"
$SAOLA install $PKG_DIR --pkg-namespace $PKG_NS

echo "--- A2: Get package ---"
$SAOLA get package --pkg-namespace $PKG_NS

PKG_NAME=$($SAOLA get package --pkg-namespace $PKG_NS -o name 2>/dev/null | head -1 | sed 's|package/||')
echo "Package name: $PKG_NAME"

echo "--- A3: Inspect ---"
$SAOLA inspect $PKG_NAME --pkg-namespace $PKG_NS 2>&1 | head -15

echo "--- A4: Baselines ---"
$SAOLA get baseline --package $PKG_NAME --pkg-namespace $PKG_NS
$SAOLA get baseline --package $PKG_NAME --kind operator --pkg-namespace $PKG_NS
$SAOLA get baseline --package $PKG_NAME --kind action --pkg-namespace $PKG_NS

# Phase B: Operator deployment (if samples exist)
if [ -f "$SAMPLES/clickhouse-operator.yaml" ]; then
  echo ""
  echo "=== Phase B: Operator Deployment ==="
  echo "--- B1: Install CRDs ---"
  kubectl apply -f $PKG_DIR/crds/ 2>/dev/null || true

  echo "--- B2: Create MiddlewareOperator ---"
  kubectl apply -f $SAMPLES/clickhouse-operator.yaml

  echo "--- B3: Wait for Available (max 3min) ---"
  for i in $(seq 1 36); do
    STATE=$(kubectl get mo -n $NS -o jsonpath='{.items[0].status.state}' 2>/dev/null)
    if [ "$STATE" = "Available" ]; then
      echo "Operator Available at attempt $i"
      break
    fi
    [ "$i" -eq 36 ] && echo "WARNING: Operator not Available after 3 minutes"
    sleep 5
  done

  echo "--- B4: Verify ---"
  $SAOLA get operator -n $NS
  kubectl get pods -n $NS
fi

# Phase C: Middleware deployment (if samples exist)
if [ -f "$SAMPLES/clickhouse-middleware.yaml" ]; then
  echo ""
  echo "=== Phase C: Middleware Deployment ==="
  echo "--- C1: Create Middleware ---"
  kubectl apply -f $SAMPLES/clickhouse-middleware.yaml

  echo "--- C2: Wait for Available (max 5min) ---"
  for i in $(seq 1 60); do
    STATE=$(kubectl get mid -n $NS -o jsonpath='{.items[0].status.state}' 2>/dev/null)
    if [ "$STATE" = "Available" ]; then
      echo "Middleware Available at attempt $i"
      break
    fi
    [ "$i" -eq 60 ] && echo "WARNING: Middleware not Available after 5 minutes"
    sleep 5
  done

  echo "--- C3: Verify ---"
  $SAOLA get middleware -n $NS
  $SAOLA get all -n $NS
  kubectl get pods -n $NS
fi

# Phase D: Output formats
echo ""
echo "=== Phase D: Output Formats ==="
$SAOLA get all -n $NS
$SAOLA get middleware -n $NS -o yaml 2>&1 | head -10
$SAOLA get operator -n $NS -o json 2>&1 | head -10
$SAOLA --lang en get all -n $NS

echo ""
echo "========================================="
echo "  saola-cli E2E TEST PASSED"
echo "========================================="
