#!/usr/bin/env bash
set -euo pipefail

# Task Center minimal smoke test:
# 1) login
# 2) create task
# 3) execute task
# 4) get execution detail
# 5) optionally cancel (when running/pending)
# 6) optionally retry (when failed/partial_fail/canceled)
#
# Usage:
# BASE_URL="http://127.0.0.1:8080" USERNAME="e2e_admin" PASSWORD="Admin123" HOST_IPS="127.0.0.1" ./backend/scripts/task_center_smoke.sh

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
USERNAME="${USERNAME:-e2e_admin}"
PASSWORD="${PASSWORD:-Admin123}"
TOKEN="${TOKEN:-}"
HOST_IPS="${HOST_IPS:-127.0.0.1}"
API_PREFIX="${API_PREFIX:-/api/v1}"
LOGIN_PATH="${LOGIN_PATH:-/auth/login}"
AUTH_HEADER_PREFIX="${AUTH_HEADER_PREFIX:-Bearer}"
MODE="${MODE:-full}" # quick/full
DO_CANCEL="${DO_CANCEL:-1}"
DO_RETRY="${DO_RETRY:-1}"
RETRY_SCOPE="${RETRY_SCOPE:-failed}" # failed/all
STEP="init"

if [[ "$MODE" == "quick" ]]; then
  DO_CANCEL=0
  DO_RETRY=0
fi

log() {
  printf '[task-smoke] %s\n' "$*"
}

fatal() {
  echo "[task-smoke][FAILED][$STEP] $*" >&2
  exit 1
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing command: $1" >&2
    exit 1
  fi
}

json_get() {
  local key="$1"
  python3 -c 'import json,sys; d=json.load(sys.stdin); print(d'"$key"')'
}

api_json() {
  local method="$1"
  local path="$2"
  local data="${3:-}"
  local token="${4:-}"

  local url="${BASE_URL}${API_PREFIX}${path}"
  local headers=(-H "Content-Type: application/json")
  if [[ -n "$token" ]]; then
    headers+=(-H "Authorization: ${AUTH_HEADER_PREFIX} ${token}")
  fi

  if [[ -n "$data" ]]; then
    curl -sS -X "$method" "$url" "${headers[@]}" -d "$data"
  else
    curl -sS -X "$method" "$url" "${headers[@]}"
  fi
}

assert_business_ok() {
  local body="$1"
  local code
  code="$(printf '%s' "$body" | json_get "['code']")"
  if [[ "$code" != "0" ]]; then
    fatal "api business error, code=${code}, body=${body}"
  fi
}

require_cmd curl
require_cmd python3

if [[ -n "$TOKEN" ]]; then
  log "skip login: using TOKEN from env"
else
  STEP="login"
  log "login: ${USERNAME}"
  LOGIN_BODY="$(api_json POST "${LOGIN_PATH}" "{\"username\":\"${USERNAME}\",\"password\":\"${PASSWORD}\"}")"
  assert_business_ok "$LOGIN_BODY"
  TOKEN="$(printf '%s' "$LOGIN_BODY" | json_get "['data']['token']")"
  if [[ -z "$TOKEN" || "$TOKEN" == "None" ]]; then
    fatal "login token empty, body=${LOGIN_BODY}"
  fi
fi

TASK_NAME="smoke_task_$(date +%s)"
SCRIPT_CONTENT='echo "smoke-start"; echo "smoke-err" 1>&2; echo "smoke-done"'
CREATE_PAYLOAD="$(TASK_NAME="$TASK_NAME" SCRIPT_CONTENT="$SCRIPT_CONTENT" python3 - <<'PY'
import json, os
print(json.dumps({
    "name": os.environ["TASK_NAME"],
    "task_type": "shell",
    "script_type": "bash",
    "script_content": os.environ["SCRIPT_CONTENT"],
    "timeout": 60,
    "run_as_user": "root",
    "description": "task center smoke",
}, ensure_ascii=False))
PY
)"

STEP="create_task"
log "create task: ${TASK_NAME}"
CREATE_BODY="$(api_json POST "/tasks" "${CREATE_PAYLOAD}" "$TOKEN")"
assert_business_ok "$CREATE_BODY"
TASK_ID="$(printf '%s' "$CREATE_BODY" | json_get "['data']['id']")"
if [[ -z "$TASK_ID" || "$TASK_ID" == "None" ]]; then
  fatal "create task failed, body=${CREATE_BODY}"
fi

IFS=',' read -r -a host_array <<< "$HOST_IPS"
hosts_json="["
for ip in "${host_array[@]}"; do
  ip_trimmed="$(echo "$ip" | xargs)"
  if [[ -n "$ip_trimmed" ]]; then
    if [[ "$hosts_json" != "[" ]]; then
      hosts_json+=","
    fi
    hosts_json+="\"${ip_trimmed}\""
  fi
done
hosts_json+="]"
EXEC_PAYLOAD="{\"host_ips\":${hosts_json}}"

STEP="execute_task"
log "execute task: task_id=${TASK_ID}, hosts=${HOST_IPS}"
EXEC_BODY="$(api_json POST "/tasks/${TASK_ID}/execute" "${EXEC_PAYLOAD}" "$TOKEN")"
assert_business_ok "$EXEC_BODY"
EXEC_ID="$(printf '%s' "$EXEC_BODY" | json_get "['data']['id']")"
if [[ -z "$EXEC_ID" || "$EXEC_ID" == "None" ]]; then
  fatal "execute task failed, body=${EXEC_BODY}"
fi

sleep 1
STEP="get_execution_detail"
DETAIL_BODY="$(api_json GET "/task-executions/${EXEC_ID}" "" "$TOKEN")"
assert_business_ok "$DETAIL_BODY"
STATUS="$(printf '%s' "$DETAIL_BODY" | json_get "['data']['status']")"
SUCCESS_COUNT="$(printf '%s' "$DETAIL_BODY" | json_get "['data']['success_count']")"
FAIL_COUNT="$(printf '%s' "$DETAIL_BODY" | json_get "['data']['fail_count']")"
log "execution detail: id=${EXEC_ID}, status=${STATUS}, success=${SUCCESS_COUNT}, fail=${FAIL_COUNT}"

if [[ "$DO_CANCEL" == "1" && ( "$STATUS" == "pending" || "$STATUS" == "running" ) ]]; then
  STEP="cancel_execution"
  log "cancel execution: ${EXEC_ID}"
  CANCEL_BODY="$(api_json POST "/task-executions/${EXEC_ID}/cancel" "" "$TOKEN")"
  assert_business_ok "$CANCEL_BODY"
  STATUS="$(printf '%s' "$CANCEL_BODY" | json_get "['data']['status']")"
  log "cancel done, status=${STATUS}"
fi

if [[ "$DO_RETRY" == "1" && ( "$STATUS" == "failed" || "$STATUS" == "partial_fail" || "$STATUS" == "canceled" ) ]]; then
  STEP="retry_execution"
  log "retry execution: ${EXEC_ID}"
  RETRY_BODY="$(api_json POST "/task-executions/${EXEC_ID}/retry?scope=${RETRY_SCOPE}" "" "$TOKEN")"
  assert_business_ok "$RETRY_BODY"
  NEW_EXEC_ID="$(printf '%s' "$RETRY_BODY" | json_get "['data']['id']")"
  log "retry done, new_execution_id=${NEW_EXEC_ID}"
fi

STEP="done"
log "smoke completed successfully (mode=${MODE})"
