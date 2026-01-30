#!/bin/bash
# Test authentication with Firebase Identity Platform
# Usage: ./scripts/test-auth.sh
#
# Requires either:
#   1. .env.test file in project root
#   2. FIREBASE_API_KEY environment variable

set -e

# Load .env.test if it exists
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
if [ -f "$PROJECT_ROOT/.env.test" ]; then
    echo "Loading $PROJECT_ROOT/.env.test"
    set -a
    source "$PROJECT_ROOT/.env.test"
    set +a
fi

# Configuration
API_KEY="${FIREBASE_API_KEY:-}"
TENANT_ID="${NAMAZU_AUTH_TENANT_ID:-test-hwuh8}"
EMAIL="${TEST_EMAIL:-otiai10+namazutest001@gmail.com}"
PASSWORD="${TEST_PASSWORD:-testtesttest}"
API_BASE="${API_BASE:-http://localhost:8080}"

if [ -z "$API_KEY" ]; then
    echo "Error: FIREBASE_API_KEY is required"
    echo ""
    echo "Options:"
    echo "  1. Create .env.test with FIREBASE_API_KEY=xxx"
    echo "  2. Run: FIREBASE_API_KEY=xxx ./scripts/test-auth.sh"
    exit 1
fi

echo "=== Step 1: Sign in with Email/Password ==="
echo "Email: $EMAIL"
echo "Tenant: $TENANT_ID"

# Sign in to get ID token (with tenant)
SIGNIN_RESPONSE=$(curl -s -X POST \
    "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=${API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"${EMAIL}\",
        \"password\": \"${PASSWORD}\",
        \"returnSecureToken\": true,
        \"tenantId\": \"${TENANT_ID}\"
    }")

# Extract ID token
ID_TOKEN=$(echo "$SIGNIN_RESPONSE" | jq -r '.idToken')

if [ "$ID_TOKEN" == "null" ] || [ -z "$ID_TOKEN" ]; then
    echo "Failed to get ID token:"
    echo "$SIGNIN_RESPONSE" | jq .
    exit 1
fi

echo "Got ID token (first 50 chars): ${ID_TOKEN:0:50}..."
echo ""

echo "=== Step 2: Test /api/me endpoint ==="
ME_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer ${ID_TOKEN}" \
    "${API_BASE}/api/me")

HTTP_CODE=$(echo "$ME_RESPONSE" | tail -1)
BODY=$(echo "$ME_RESPONSE" | sed '$d')

echo "HTTP Status: $HTTP_CODE"
echo "Response:"
echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
echo ""

echo "=== Step 3: Test /api/me/providers endpoint ==="
PROVIDERS_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer ${ID_TOKEN}" \
    "${API_BASE}/api/me/providers")

HTTP_CODE=$(echo "$PROVIDERS_RESPONSE" | tail -1)
BODY=$(echo "$PROVIDERS_RESPONSE" | sed '$d')

echo "HTTP Status: $HTTP_CODE"
echo "Response:"
echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
echo ""

echo "=== Step 4: Create a test subscription ==="
CREATE_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST \
    -H "Authorization: Bearer ${ID_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "Test Subscription",
        "delivery": {
            "type": "webhook",
            "url": "https://example.com/webhook",
            "secret": "test-secret-123"
        }
    }' \
    "${API_BASE}/api/subscriptions")

HTTP_CODE=$(echo "$CREATE_RESPONSE" | tail -1)
BODY=$(echo "$CREATE_RESPONSE" | sed '$d')

echo "HTTP Status: $HTTP_CODE"
echo "Response:"
echo "$BODY" | jq . 2>/dev/null || echo "$BODY"

# Extract subscription ID for cleanup
SUB_ID=$(echo "$BODY" | jq -r '.id' 2>/dev/null)
echo ""

echo "=== Step 5: List my subscriptions ==="
LIST_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer ${ID_TOKEN}" \
    "${API_BASE}/api/subscriptions")

HTTP_CODE=$(echo "$LIST_RESPONSE" | tail -1)
BODY=$(echo "$LIST_RESPONSE" | sed '$d')

echo "HTTP Status: $HTTP_CODE"
echo "Response:"
echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
echo ""

if [ "$SUB_ID" != "null" ] && [ -n "$SUB_ID" ]; then
    echo "=== Step 6: Delete test subscription ==="
    DELETE_RESPONSE=$(curl -s -w "\n%{http_code}" \
        -X DELETE \
        -H "Authorization: Bearer ${ID_TOKEN}" \
        "${API_BASE}/api/subscriptions/${SUB_ID}")

    HTTP_CODE=$(echo "$DELETE_RESPONSE" | tail -1)
    echo "HTTP Status: $HTTP_CODE"
    echo "Deleted subscription: $SUB_ID"
fi

echo ""
echo "=== Test Complete ==="
