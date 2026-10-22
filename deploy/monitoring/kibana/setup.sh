#!/bin/sh
# Creates (or overwrites) the ABET Kibana assets: data view "ABET logs" (default) and two saved searches.
# Idempotent: every object has a fixed id. Needs only curl.
# Env: ELASTIC_PASSWORD (user "elastic"), KIBANA_URL (default http://kibana:5601/kibana),
#      KIBANA_WAIT_SECONDS (default 300). Secrets are never printed or put on a command line.
set -eu

KIBANA_URL="${KIBANA_URL:-http://kibana:5601/kibana}"
KIBANA_URL="${KIBANA_URL%/}"
KIBANA_WAIT_SECONDS="${KIBANA_WAIT_SECONDS:-300}"
: "${ELASTIC_PASSWORD:?ELASTIC_PASSWORD is required}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

# curl reads the credentials from a config file so they never show up in the process list or logs.
esc="$(printf '%s' "$ELASTIC_PASSWORD" | sed 's/\\/\\\\/g; s/"/\\"/g')"
printf 'user = "elastic:%s"\n' "$esc" > "$tmp/curl.cfg"
chmod 600 "$tmp/curl.cfg"
unset esc

# kcurl <method> <path> [extra curl args...]  -> body in $tmp/out, prints the HTTP status code
kcurl() {
  method="$1"; path="$2"; shift 2
  curl -sS -K "$tmp/curl.cfg" -X "$method" -H 'kbn-xsrf: abet-setup' \
    -o "$tmp/out" -w '%{http_code}' --connect-timeout 5 --max-time 60 "$@" "$KIBANA_URL$path"
}

fail() {
  echo "kibana-setup: $1" >&2
  [ -s "$tmp/out" ] && head -c 600 "$tmp/out" >&2 && echo >&2
  exit 1
}

echo "kibana-setup: waiting for Kibana at $KIBANA_URL (up to ${KIBANA_WAIT_SECONDS}s)"
waited=0
while :; do
  code="$(kcurl GET /api/status 2>/dev/null || true)"
  if [ "$code" = "200" ] && grep -q '"overall":{"level":"available"' "$tmp/out"; then break; fi
  if [ "$code" = "401" ]; then fail "Kibana rejected the elastic credentials (HTTP 401)"; fi
  if [ "$waited" -ge "$KIBANA_WAIT_SECONDS" ]; then fail "Kibana not available after ${KIBANA_WAIT_SECONDS}s (last HTTP status: ${code:-none})"; fi
  sleep 5
  waited=$((waited + 5))
done
echo "kibana-setup: Kibana is available"

# 1. data view (override=true replaces an existing one with the same id)
code="$(kcurl POST /api/data_views/data_view -H 'Content-Type: application/json' -d '{
  "override": true,
  "data_view": {"id": "abet-logs", "name": "ABET logs", "title": "filebeat-*", "timeFieldName": "@timestamp"}
}')"
[ "$code" = "200" ] || fail "creating data view abet-logs failed (HTTP $code)"
echo "kibana-setup: data view 'ABET logs' (abet-logs) ready"

# 2. make it the default data view
code="$(kcurl POST /api/data_views/default -H 'Content-Type: application/json' \
  -d '{"data_view_id": "abet-logs", "force": true}')"
[ "$code" = "200" ] || fail "setting the default data view failed (HTTP $code)"

# 3. saved searches, imported with fixed ids (overwrite=true makes re-runs a no-op)
ref='"references":[{"name":"kibanaSavedObjectMeta.searchSourceJSON.index","type":"index-pattern","id":"abet-logs"}]'
src() { printf '{\\"query\\":{\\"query\\":\\"%s\\",\\"language\\":\\"kuery\\"},\\"filter\\":[],\\"indexRefName\\":\\"kibanaSavedObjectMeta.searchSourceJSON.index\\"}' "$1"; }
cols='["service.name","log.level","message","trace.id"]'
sort='[["@timestamp","desc"]]'
{
  printf '{"type":"search","id":"abet-errors","attributes":{"title":"ABET errors","description":"Log records with log.level error from every ABET service","columns":%s,"sort":%s,"grid":{},"hideChart":false,"isTextBasedQuery":false,"kibanaSavedObjectMeta":{"searchSourceJSON":"%s"}},%s}\n' \
    "$cols" "$sort" "$(src 'log.level: error')" "$ref"
  printf '{"type":"search","id":"abet-by-service","attributes":{"title":"ABET by service","description":"All log records, grouped by service; filter with service.name","columns":%s,"sort":%s,"grid":{},"hideChart":false,"isTextBasedQuery":false,"kibanaSavedObjectMeta":{"searchSourceJSON":"%s"}},%s}\n' \
    "$cols" "$sort" "$(src '*')" "$ref"
} > "$tmp/searches.ndjson"

code="$(kcurl POST '/api/saved_objects/_import?overwrite=true' -F "file=@$tmp/searches.ndjson")"
[ "$code" = "200" ] || fail "importing saved searches failed (HTTP $code)"
grep -q '"success":true' "$tmp/out" || fail "saved search import reported errors"
echo "kibana-setup: saved searches 'ABET errors' and 'ABET by service' ready"
echo "kibana-setup: done"
