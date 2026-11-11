#!/usr/bin/env bash
# Exercise the seeded demo: a read is allowed, a drop is denied, a delete waits
# for approval. Requires curl and python3. The proxy must already be serving.
set -euo pipefail

base="${KEYWARDEN_URL:-http://127.0.0.1:8080}"
: "${KEYWARDEN_ADMIN_TOKEN:?set KEYWARDEN_ADMIN_TOKEN}"

ready=0
for _ in $(seq 1 30); do
  if curl -sf "$base/healthz" >/dev/null && curl -sf "$base/readyz" >/dev/null; then
    ready=1
    break
  fi
  sleep 1
done
if [[ "$ready" != 1 ]]; then
  echo "proxy did not become ready" >&2
  exit 1
fi
meta=$(curl -sf "$base/v1/meta")
python3 -c 'import json,sys; meta=json.load(sys.stdin); assert meta.get("demo") is True, meta; assert meta.get("version"), meta' <<<"$meta"

auth=(-H "Authorization: Bearer $KEYWARDEN_ADMIN_TOKEN" -H "Content-Type: application/json")
task=$(curl -sf "${auth[@]}" -d '{"agent_id":"support","ttl_seconds":900}' "$base/v1/tasks")
token=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])' <<<"$task")
task_id=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["task"]["id"])' <<<"$task")

call() {
  local sql=$1
  curl -s -H "Authorization: Bearer $token" -H "Content-Type: application/json" \
    -d "$(python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"orders__query","arguments":{"sql":sys.argv[1]}}}))' "$sql")" \
    "$base/mcp"
  echo
}

echo "allow"
allow=$(call "SELECT status, customer FROM orders WHERE id = 1042")
echo "$allow"
python3 -c 'import json,sys; body=json.load(sys.stdin); text=body["result"]["content"][0]["text"]; assert body["result"]["isError"] is False, body; assert "Ada Lovelace" in text, text' <<<"$allow"
echo "deny"
deny=$(call "DROP TABLE orders")
echo "$deny"
python3 -c 'import json,sys; body=json.load(sys.stdin); assert body["result"]["isError"] is True, body; assert "denied" in body["result"]["content"][0]["text"]' <<<"$deny"

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
verify=$(curl -sf -H "Authorization: Bearer $KEYWARDEN_ADMIN_TOKEN" "$base/v1/audit/verify")
echo "$verify"
python3 -c 'import json,sys; body=json.load(sys.stdin); assert body.get("ok") is True, body' <<<"$verify"

echo "revoke"
curl -sf -X POST "${auth[@]}" "$base/v1/tasks/$task_id/revoke" >/dev/null
revoked=$(curl -s -o /tmp/keywarden-revoked.json -w '%{http_code}' -H "Authorization: Bearer $token" -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' "$base/mcp")
if [[ "$revoked" != "401" ]]; then
  echo "revoked task returned $revoked" >&2
  cat /tmp/keywarden-revoked.json >&2
  exit 1
fi

echo "org isolation"
suffix=$(python3 -c 'import secrets; print(secrets.token_hex(3))')
org_a=$(curl -sf "${auth[@]}" -d "{\"slug\":\"qa${suffix}a\",\"name\":\"QA A\"}" "$base/v1/orgs")
org_b=$(curl -sf "${auth[@]}" -d "{\"slug\":\"qa${suffix}b\",\"name\":\"QA B\"}" "$base/v1/orgs")
token_a=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])' <<<"$org_a")
token_b=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])' <<<"$org_b")
curl -sf -H "Authorization: Bearer $token_a" -H "Content-Type: application/json" \
  -d "{\"id\":\"probe${suffix}\",\"name\":\"Probe\"}" "$base/v1/agents" >/dev/null
agents_b=$(curl -sf -H "Authorization: Bearer $token_b" "$base/v1/agents")
python3 -c 'import json,sys; ids=[a["id"] for a in json.load(sys.stdin)["agents"]]; assert not any(i.startswith("probe") for i in ids), ids' <<<"$agents_b"
cross=$(curl -s -o /tmp/keywarden-cross.json -w '%{http_code}' -H "Authorization: Bearer $token_b" -H "Content-Type: application/json" \
  -d '{"decision":"deny","comment":"cross"}' "$base/v1/approvals/00000000-0000-4000-8000-000000000099")
if [[ "$cross" == "200" ]]; then
  echo "org B decided an approval it does not own" >&2
  cat /tmp/keywarden-cross.json >&2
  exit 1
fi
echo "orgs isolated"
