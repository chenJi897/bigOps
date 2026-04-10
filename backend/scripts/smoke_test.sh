#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
USERNAME="${USERNAME:-admin}"
PASSWORD="${PASSWORD:-admin123}"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'
PASS=0
FAIL=0

check() {
  local desc="$1" code="$2" body="$3"
  if [ "$code" -ge 200 ] && [ "$code" -lt 300 ]; then
    echo -e "${GREEN}[PASS]${NC} $desc (HTTP $code)"
    PASS=$((PASS + 1))
  elif echo "$body" | grep -q '"code":0'; then
    echo -e "${GREEN}[PASS]${NC} $desc (HTTP $code, biz code 0)"
    PASS=$((PASS + 1))
  else
    echo -e "${RED}[FAIL]${NC} $desc (HTTP $code)"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== BigOps Smoke Test ==="
echo "Target: $BASE_URL"
echo ""

# Health check
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health")
BODY=$(curl -s "$BASE_URL/health")
check "Health check" "$CODE" "$BODY"

# Login
LOGIN_RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")
LOGIN_CODE=$(echo "$LOGIN_RESP" | tail -1)
LOGIN_BODY=$(echo "$LOGIN_RESP" | head -n -1)
check "Login" "$LOGIN_CODE" "$LOGIN_BODY"

TOKEN=$(echo "$LOGIN_BODY" | grep -o '"token":"[^"]*"' | head -1 | sed 's/"token":"//;s/"//')
if [ -z "$TOKEN" ]; then
  echo -e "${RED}Cannot extract token, aborting.${NC}"
  exit 1
fi
AUTH="Authorization: Bearer $TOKEN"

# Task list
RESP=$(curl -s -w "\n%{http_code}" -H "$AUTH" "$BASE_URL/api/v1/tasks?page=1&size=5")
CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
check "Task list" "$CODE" "$BODY"

# Create task
CREATE_RESP=$(curl -s -w "\n%{http_code}" -X POST -H "$AUTH" -H "Content-Type: application/json" \
  "$BASE_URL/api/v1/tasks" \
  -d '{"name":"smoke_test_task","task_type":"script","script_type":"bash","script_content":"echo hello","timeout":30}')
CREATE_CODE=$(echo "$CREATE_RESP" | tail -1)
CREATE_BODY=$(echo "$CREATE_RESP" | head -n -1)
check "Create task" "$CREATE_CODE" "$CREATE_BODY"

TASK_ID=$(echo "$CREATE_BODY" | grep -o '"id":[0-9]*' | head -1 | sed 's/"id"://')

# Get task by ID
if [ -n "$TASK_ID" ]; then
  RESP=$(curl -s -w "\n%{http_code}" -H "$AUTH" "$BASE_URL/api/v1/tasks/$TASK_ID")
  CODE=$(echo "$RESP" | tail -1)
  BODY=$(echo "$RESP" | head -n -1)
  check "Get task by ID" "$CODE" "$BODY"

  # Delete task
  RESP=$(curl -s -w "\n%{http_code}" -X POST -H "$AUTH" "$BASE_URL/api/v1/tasks/$TASK_ID/delete")
  CODE=$(echo "$RESP" | tail -1)
  BODY=$(echo "$RESP" | head -n -1)
  check "Delete task" "$CODE" "$BODY"
fi

# Alert rules list
RESP=$(curl -s -w "\n%{http_code}" -H "$AUTH" "$BASE_URL/api/v1/alert-rules?page=1&size=5")
CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
check "Alert rules list" "$CODE" "$BODY"

# Alert events list
RESP=$(curl -s -w "\n%{http_code}" -H "$AUTH" "$BASE_URL/api/v1/alert-events?page=1&size=5")
CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
check "Alert events list" "$CODE" "$BODY"

# Monitor summary
RESP=$(curl -s -w "\n%{http_code}" -H "$AUTH" "$BASE_URL/api/v1/monitor/summary")
CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
check "Monitor summary" "$CODE" "$BODY"

# Golden signals
RESP=$(curl -s -w "\n%{http_code}" -H "$AUTH" "$BASE_URL/api/v1/monitor/golden-signals?minutes=60")
CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
check "Golden signals" "$CODE" "$BODY"

# Inspection templates
RESP=$(curl -s -w "\n%{http_code}" -H "$AUTH" "$BASE_URL/api/v1/inspection/templates?page=1&size=5")
CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
check "Inspection templates" "$CODE" "$BODY"

# Pending approvals
RESP=$(curl -s -w "\n%{http_code}" -H "$AUTH" "$BASE_URL/api/v1/task-approvals/pending?page=1&size=5")
CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
check "Pending task approvals" "$CODE" "$BODY"

# Logout
RESP=$(curl -s -w "\n%{http_code}" -X POST -H "$AUTH" "$BASE_URL/api/v1/auth/logout")
CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
check "Logout" "$CODE" "$BODY"

echo ""
echo "=== Results: ${GREEN}${PASS} passed${NC}, ${RED}${FAIL} failed${NC} ==="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
