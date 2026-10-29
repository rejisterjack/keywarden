#!/usr/bin/env bash
# Exercise the seeded demo: a read is allowed, a drop is denied, a delete waits
# for approval. Requires curl and python3. The proxy must already be serving.
set -euo pipefail

base="${KEYWARDEN_URL:-http://127.0.0.1:8080}"
: "${KEYWARDEN_ADMIN_TOKEN:?set KEYWARDEN_ADMIN_TOKEN}"

for _ in $(seq 1 30); do
  if curl -sf "$base/healthz" >/dev/null; then
    break
  fi
  sleep 1
done

auth=(-H "Authorization: Bearer $KEYWARDEN_ADMIN_TOKEN" -H "Content-Type: application/json")
task=$(curl -sf "${auth[@]}" -d '{"agent_id":"support","ttl_seconds":900}' "$base/v1/tasks")
token=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])' <<<"$task")

call() {
  local sql=$1
  curl -s -H "Authorization: Bearer $token" -H "Content-Type: application/json" \
    -d "$(python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"orders__query","arguments":{"sql":sys.argv[1]}}}))' "$sql")" \
    "$base/mcp"
  echo
}

echo "allow"
call "SELECT status, customer FROM orders WHERE id = 1042"
echo "deny"
call "DROP TABLE orders"

echo "approval"
call "DELETE FROM orders WHERE id = 1042" >/tmp/keywarden-delete.json &
pid=$!
approval=""
for _ in $(seq 1 50); do
  pending=$(curl -sf -H "Authorization: Bearer $KEYWARDEN_ADMIN_TOKEN" "$base/v1/approvals?status=pending")
  approval=$(python3 -c 'import json,sys; rows=json.load(sys.stdin)["approvals"]; print(rows[0]["id"] if rows else "")' <<<"$pending")
  if [[ -n "$approval" ]]; then
    break
  fi
  sleep 0.1
done
if [[ -z "$approval" ]]; then
  echo "no pending approval appeared" >&2
  exit 1
fi
curl -sf "${auth[@]}" -d '{"decision":"approve","comment":"quickstart"}' "$base/v1/approvals/$approval" >/dev/null
wait "$pid"
echo "approved"
cat /tmp/keywarden-delete.json
echo
curl -sf -H "Authorization: Bearer $KEYWARDEN_ADMIN_TOKEN" "$base/v1/audit/verify"
echo
