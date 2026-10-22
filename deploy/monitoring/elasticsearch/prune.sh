#!/bin/sh
# Manual cleanup (make mon-prune days=N): deletes log + trace documents older than N days. Never runs automatically.
# Env: DAYS, ELASTIC_PASSWORD, ES_URL. Runs inside the es-init image (curl only).
set -eu
ES_URL="${ES_URL:-http://elasticsearch:9200}"
: "${DAYS:?}" "${ELASTIC_PASSWORD:?}"
case "$DAYS" in ''|*[!0-9]*) echo "DAYS must be a number" >&2; exit 1 ;; esac

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM
esc="$(printf '%s' "$ELASTIC_PASSWORD" | sed 's/\\/\\\\/g; s/"/\\"/g')"
printf 'user = "elastic:%s"\n' "$esc" > "$tmp/curl.cfg"
chmod 600 "$tmp/curl.cfg"
q="{\"query\":{\"range\":{\"@timestamp\":{\"lt\":\"now-${DAYS}d\"}}}}"
opts="ignore_unavailable=true&allow_no_indices=true"

for pat in 'filebeat-*' 'traces-apm*' 'logs-apm*' 'metrics-apm*'; do
	n=$(curl -sS -K "$tmp/curl.cfg" -H 'Content-Type: application/json' -d "$q" "$ES_URL/$pat/_count?$opts" \
		| sed -n 's/.*"count":\([0-9]*\).*/\1/p')
	d=$(curl -sS -K "$tmp/curl.cfg" -H 'Content-Type: application/json' -d "$q" \
		"$ES_URL/$pat/_delete_by_query?conflicts=proceed&wait_for_completion=true&$opts" \
		| sed -n 's/.*"deleted":\([0-9]*\).*/\1/p')
	echo "$pat: older than ${DAYS}d = ${n:-0}, deleted = ${d:-0}"
	curl -sS -K "$tmp/curl.cfg" -X POST "$ES_URL/$pat/_forcemerge?only_expunge_deletes=true&$opts" >/dev/null || true
done
echo "mon-prune done"
