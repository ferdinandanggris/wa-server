#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# Billing E2E Test Script
# Tests billing API endpoints against a running server.
# Prerequisites: Server running on localhost:9090, seed data applied
# ============================================================

BASE_URL="${BASE_URL:-http://localhost:9090}"
COMPANY_ID="${COMPANY_ID:-00000000-0000-0000-0000-000000000001}"
PASS=0
FAIL=0

check() {
    local label="$1"
    local status="$2"
    if [ "$status" -ge 200 ] && [ "$status" -lt 300 ]; then
        echo "  ✅ $label"
        PASS=$((PASS + 1))
    else
        echo "  ❌ $label (HTTP $status)"
        FAIL=$((FAIL + 1))
    fi
}

echo "============================================"
echo " Billing E2E Tests"
echo " Base URL: $BASE_URL"
echo " Company:  $COMPANY_ID"
echo "============================================"

# 1. GET /api/v1/billing/quota
echo ""
echo "1) GET /api/v1/billing/quota"
status=$(curl -s -o /tmp/billing_quota.json -w "%{http_code}" "$BASE_URL/api/v1/billing/quota?company_id=$COMPANY_ID")
check "quota" "$status"
if [ "$status" -eq 200 ]; then
    cat /tmp/billing_quota.json | python3 -m json.tool --no-ensure-ascii 2>/dev/null || cat /tmp/billing_quota.json
fi

# 2. GET /api/v1/billing/quota (missing company_id → 400)
echo ""
echo "2) GET /api/v1/billing/quota (no company_id)"
status=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/billing/quota")
if [ "$status" -eq 400 ]; then
    echo "  ✅ missing company_id → 400"
    PASS=$((PASS + 1))
else
    echo "  ❌ missing company_id → $status (expected 400)"
    FAIL=$((FAIL + 1))
fi

# 3. GET /api/v1/billing/usage
echo ""
echo "3) GET /api/v1/billing/usage"
status=$(curl -s -o /tmp/billing_usage.json -w "%{http_code}" "$BASE_URL/api/v1/billing/usage?company_id=$COMPANY_ID")
check "usage" "$status"
if [ "$status" -eq 200 ]; then
    cat /tmp/billing_usage.json | python3 -m json.tool --no-ensure-ascii 2>/dev/null || cat /tmp/billing_usage.json
fi

# 4. GET /api/v1/billing/cost-summary
echo ""
echo "4) GET /api/v1/billing/cost-summary"
status=$(curl -s -o /tmp/billing_summary.json -w "%{http_code}" "$BASE_URL/api/v1/billing/cost-summary")
check "cost-summary" "$status"
if [ "$status" -eq 200 ]; then
    cat /tmp/billing_summary.json | python3 -m json.tool --no-ensure-ascii 2>/dev/null || cat /tmp/billing_summary.json
fi

# 5. POST /api/v1/billing/sync-costs
echo ""
echo "5) POST /api/v1/billing/sync-costs"
status=$(curl -s -o /tmp/billing_sync.json -w "%{http_code}" -X POST "$BASE_URL/api/v1/billing/sync-costs")
check "sync-costs" "$status"
if [ "$status" -eq 200 ]; then
    cat /tmp/billing_sync.json | python3 -m json.tool --no-ensure-ascii 2>/dev/null || cat /tmp/billing_sync.json
fi

# 6. GET /api/v1/billing/sync-costs (wrong method → 405)
echo ""
echo "6) GET /api/v1/billing/sync-costs (wrong method)"
status=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/billing/sync-costs")
if [ "$status" -eq 405 ]; then
    echo "  ✅ wrong method → 405"
    PASS=$((PASS + 1))
else
    echo "  ❌ wrong method → $status (expected 405)"
    FAIL=$((FAIL + 1))
fi

# 7. GET /api/v1/billing/usage with date range
echo ""
echo "7) GET /api/v1/billing/usage with date range"
START=$(date -u -d '-7 days' +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -v-7d +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "7daysago")
END=$(date -u +%Y-%m-%dT%H:%M:%SZ)
status=$(curl -s -o /tmp/billing_usage_range.json -w "%{http_code}" \
    "$BASE_URL/api/v1/billing/usage?company_id=$COMPANY_ID&start=$START&end=$END")
check "usage with date range" "$status"

# Summary
echo ""
echo "============================================"
echo " Results: $PASS passed, $FAIL failed"
echo "============================================"

exit $FAIL
