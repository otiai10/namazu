#!/bin/bash
#
# E2E Test Script for namazu
#
# This script can be run by Claude Code autonomously to verify functionality.
# It supports two modes:
#   1. Emulator mode (preferred): Uses Firebase Emulator for isolated testing
#   2. Real Firestore mode: Uses real Firestore with --test-mode (auth disabled)
#
# Usage:
#   ./scripts/e2e-test.sh           # Auto-detect mode
#   ./scripts/e2e-test.sh emulator  # Force emulator mode
#   ./scripts/e2e-test.sh real      # Force real Firestore mode
#

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
API_PORT=18080
WEBHOOK_PORT=19090
EMULATOR_UI_PORT=4000
FIRESTORE_PORT=8080
AUTH_PORT=9099
NAMAZU_PID=""
WEBHOOK_PID=""

cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    [ -n "$NAMAZU_PID" ] && kill $NAMAZU_PID 2>/dev/null || true
    [ -n "$WEBHOOK_PID" ] && kill $WEBHOOK_PID 2>/dev/null || true
    # Stop Firebase Emulator (docker compose)
    if [ "$USE_EMULATOR" = true ]; then
        docker compose down 2>/dev/null || true
    fi
    # Kill any remaining processes on our ports
    lsof -ti:$API_PORT | xargs kill 2>/dev/null || true
    lsof -ti:$WEBHOOK_PORT | xargs kill 2>/dev/null || true
    rm -f /tmp/namazu-e2e.log /tmp/webhook-e2e.log /tmp/emulator-e2e.log
}

trap cleanup EXIT

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

wait_for_port() {
    local port=$1
    local timeout=${2:-30}
    local count=0
    while ! nc -z localhost $port 2>/dev/null; do
        sleep 1
        count=$((count + 1))
        if [ $count -ge $timeout ]; then
            log_error "Timeout waiting for port $port"
            return 1
        fi
    done
    return 0
}

# Determine mode
MODE="${1:-auto}"
USE_EMULATOR=false

if [ "$MODE" = "emulator" ]; then
    USE_EMULATOR=true
elif [ "$MODE" = "real" ]; then
    USE_EMULATOR=false
elif [ "$MODE" = "auto" ]; then
    # Check if docker compose is available
    if command -v docker &> /dev/null && docker compose version &> /dev/null; then
        USE_EMULATOR=true
    else
        log_warn "Docker Compose not found, using real Firestore with --test-mode"
        USE_EMULATOR=false
    fi
fi

echo "========================================"
echo "  namazu E2E Test"
echo "========================================"
echo "Mode: $([ "$USE_EMULATOR" = true ] && echo "Emulator" || echo "Real Firestore")"
echo ""

# 1. Start Firebase Emulators via Docker Compose (if using emulator mode)
if [ "$USE_EMULATOR" = true ]; then
    log_info "Starting Firebase Emulators via Docker Compose..."
    docker compose up -d firebase-emulators > /tmp/emulator-e2e.log 2>&1

    # Wait for all emulator ports to be ready
    log_info "Waiting for Emulator UI (port $EMULATOR_UI_PORT)..."
    if ! wait_for_port $EMULATOR_UI_PORT 60; then
        log_error "Emulator UI failed to start"
        docker compose logs firebase-emulators
        exit 1
    fi

    log_info "Waiting for Firestore (port $FIRESTORE_PORT)..."
    if ! wait_for_port $FIRESTORE_PORT 30; then
        log_error "Firestore Emulator failed to start"
        docker compose logs firebase-emulators
        exit 1
    fi

    log_info "Waiting for Auth (port $AUTH_PORT)..."
    if ! wait_for_port $AUTH_PORT 30; then
        log_error "Auth Emulator failed to start"
        docker compose logs firebase-emulators
        exit 1
    fi

    log_info "Firebase Emulators started"
    export FIREBASE_AUTH_EMULATOR_HOST="localhost:$AUTH_PORT"
    export FIRESTORE_EMULATOR_HOST="localhost:$FIRESTORE_PORT"
fi

# 2. Start dummy webhook receiver
log_info "Starting webhook receiver on port $WEBHOOK_PORT..."
PORT=$WEBHOOK_PORT go run ./cmd/dummysubscriber/ > /tmp/webhook-e2e.log 2>&1 &
WEBHOOK_PID=$!
sleep 2

if ! wait_for_port $WEBHOOK_PORT 10; then
    log_error "Webhook receiver failed to start"
    cat /tmp/webhook-e2e.log
    exit 1
fi
log_info "Webhook receiver started"

# 3. Start namazu server
log_info "Starting namazu server on port $API_PORT..."
if [ "$USE_EMULATOR" = true ]; then
    FIREBASE_AUTH_EMULATOR_HOST="localhost:$AUTH_PORT" \
    FIRESTORE_EMULATOR_HOST="localhost:$FIRESTORE_PORT" \
    NAMAZU_SOURCE_ENDPOINT=wss://api-realtime-sandbox.p2pquake.net/v2/ws \
    NAMAZU_STORE_PROJECT_ID=namazu-test \
    NAMAZU_STORE_DATABASE="(default)" \
    NAMAZU_API_ADDR=":$API_PORT" \
    go run ./cmd/namazu/ --test-mode > /tmp/namazu-e2e.log 2>&1 &
else
    # Use real Firestore from .env.localdev
    source .env.localdev 2>/dev/null || true
    NAMAZU_API_ADDR=":$API_PORT" \
    go run ./cmd/namazu/ --test-mode > /tmp/namazu-e2e.log 2>&1 &
fi
NAMAZU_PID=$!
sleep 3

if ! wait_for_port $API_PORT 30; then
    log_error "namazu server failed to start"
    cat /tmp/namazu-e2e.log
    exit 1
fi
log_info "namazu server started"

# 4. Run API tests
API_BASE="http://localhost:$API_PORT"
PASSED=0
FAILED=0

run_test() {
    local name="$1"
    local result="$2"
    if [ "$result" = "0" ]; then
        echo -e "  ${GREEN}✓${NC} $name"
        PASSED=$((PASSED + 1))
    else
        echo -e "  ${RED}✗${NC} $name"
        FAILED=$((FAILED + 1))
    fi
}

echo ""
log_info "Running API tests..."
echo ""

# Test: Health check
HEALTH=$(curl -s "$API_BASE/health" | jq -r '.status' 2>/dev/null)
run_test "Health check returns 'ok'" "$([ "$HEALTH" = "ok" ] && echo 0 || echo 1)"

# Test: Create subscription without filter
SUB1_ID=$(curl -s -X POST "$API_BASE/api/subscriptions" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"E2E Test - No Filter\",
        \"delivery\": {
            \"type\": \"webhook\",
            \"url\": \"http://localhost:$WEBHOOK_PORT/webhook\",
            \"secret\": \"test-secret-key\"
        }
    }" | jq -r '.id' 2>/dev/null)
run_test "Create subscription without filter" "$([ -n "$SUB1_ID" ] && [ "$SUB1_ID" != "null" ] && echo 0 || echo 1)"

# Test: Create subscription with MinScale filter
SUB2_ID=$(curl -s -X POST "$API_BASE/api/subscriptions" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"E2E Test - Scale 4+\",
        \"delivery\": {
            \"type\": \"webhook\",
            \"url\": \"http://localhost:$WEBHOOK_PORT/webhook\",
            \"secret\": \"test-secret-key\"
        },
        \"filter\": {
            \"min_scale\": 40
        }
    }" | jq -r '.id' 2>/dev/null)
run_test "Create subscription with MinScale filter" "$([ -n "$SUB2_ID" ] && [ "$SUB2_ID" != "null" ] && echo 0 || echo 1)"

# Test: Create subscription with Prefecture filter
SUB3_ID=$(curl -s -X POST "$API_BASE/api/subscriptions" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"E2E Test - Tokyo Only\",
        \"delivery\": {
            \"type\": \"webhook\",
            \"url\": \"http://localhost:$WEBHOOK_PORT/webhook\",
            \"secret\": \"test-secret-key\"
        },
        \"filter\": {
            \"prefectures\": [\"東京都\"]
        }
    }" | jq -r '.id' 2>/dev/null)
run_test "Create subscription with Prefecture filter" "$([ -n "$SUB3_ID" ] && [ "$SUB3_ID" != "null" ] && echo 0 || echo 1)"

# Test: List subscriptions
SUB_COUNT=$(curl -s "$API_BASE/api/subscriptions" | jq 'length' 2>/dev/null)
run_test "List subscriptions (count >= 3)" "$([ "$SUB_COUNT" -ge 3 ] && echo 0 || echo 1)"

# Test: Get subscription by ID
if [ -n "$SUB1_ID" ] && [ "$SUB1_ID" != "null" ]; then
    SUB1_NAME=$(curl -s "$API_BASE/api/subscriptions/$SUB1_ID" | jq -r '.name' 2>/dev/null)
    run_test "Get subscription by ID" "$([ "$SUB1_NAME" = "E2E Test - No Filter" ] && echo 0 || echo 1)"
else
    run_test "Get subscription by ID" "1"
fi

# Test: Update subscription
if [ -n "$SUB1_ID" ] && [ "$SUB1_ID" != "null" ]; then
    UPDATE_RESULT=$(curl -s -X PUT "$API_BASE/api/subscriptions/$SUB1_ID" \
        -H "Content-Type: application/json" \
        -d "{
            \"name\": \"E2E Test - Updated\",
            \"delivery\": {
                \"type\": \"webhook\",
                \"url\": \"http://localhost:$WEBHOOK_PORT/webhook\",
                \"secret\": \"test-secret-key\"
            }
        }" | jq -r '.name' 2>/dev/null)
    run_test "Update subscription" "$([ "$UPDATE_RESULT" = "E2E Test - Updated" ] && echo 0 || echo 1)"
else
    run_test "Update subscription" "1"
fi

# Test: Delete subscription
if [ -n "$SUB1_ID" ] && [ "$SUB1_ID" != "null" ]; then
    DELETE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API_BASE/api/subscriptions/$SUB1_ID")
    run_test "Delete subscription" "$([ "$DELETE_STATUS" = "204" ] && echo 0 || echo 1)"
else
    run_test "Delete subscription" "1"
fi

# Cleanup remaining test subscriptions
[ -n "$SUB2_ID" ] && [ "$SUB2_ID" != "null" ] && curl -s -X DELETE "$API_BASE/api/subscriptions/$SUB2_ID" > /dev/null
[ -n "$SUB3_ID" ] && [ "$SUB3_ID" != "null" ] && curl -s -X DELETE "$API_BASE/api/subscriptions/$SUB3_ID" > /dev/null

# 5. Print results
echo ""
echo "========================================"
echo "  Test Results"
echo "========================================"
echo -e "  Passed: ${GREEN}$PASSED${NC}"
echo -e "  Failed: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    log_info "All tests passed!"
    exit 0
else
    log_error "Some tests failed"
    echo ""
    echo "namazu logs:"
    tail -20 /tmp/namazu-e2e.log
    exit 1
fi
