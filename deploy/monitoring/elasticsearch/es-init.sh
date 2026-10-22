#!/bin/sh
# One-shot, idempotent Elasticsearch bootstrap (run by the es-init compose service, every `up`):
#   1. kibana_system password            2. role + user abet_ingest (Filebeat + APM Server writers)
#   3. role + user abet_reader (Grafana, read-only)   4. retention = forever (D26): no delete phase anywhere
# Env: ELASTIC_PASSWORD, KIBANA_SYSTEM_PASSWORD, ES_INGEST_PASSWORD, ES_READER_PASSWORD, ES_URL. Secrets never printed.
set -eu

ES_URL="${ES_URL:-http://elasticsearch:9200}"
: "${ELASTIC_PASSWORD:?}" "${KIBANA_SYSTEM_PASSWORD:?}" "${ES_INGEST_PASSWORD:?}" "${ES_READER_PASSWORD:?}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

# credentials go through a curl config file (not the command line / process list)
jesc() { printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'; }
printf 'user = "elastic:%s"\n' "$(jesc "$ELASTIC_PASSWORD")" > "$tmp/curl.cfg"
chmod 600 "$tmp/curl.cfg"

# es <method> <path> [json-body]  -> prints HTTP status; body in $tmp/out
es() {
	if [ $# -ge 3 ]; then
		printf '%s' "$3" > "$tmp/body"
		curl -sS -K "$tmp/curl.cfg" -X "$1" -H 'Content-Type: application/json' --data-binary "@$tmp/body" \
			-o "$tmp/out" -w '%{http_code}' --connect-timeout 5 --max-time 60 "$ES_URL$2"
	else
		curl -sS -K "$tmp/curl.cfg" -X "$1" -o "$tmp/out" -w '%{http_code}' --connect-timeout 5 --max-time 60 "$ES_URL$2"
	fi
}
must() { # must <label> <status> [accepted statuses...]
	label="$1"; code="$2"; shift 2
	for ok in 200 201 "$@"; do [ "$code" = "$ok" ] && { echo "es-init: $label ok ($code)"; return 0; }; done
	echo "es-init: $label FAILED ($code): $(head -c 400 "$tmp/out")" >&2
	exit 1
}

# wait for the cluster (the compose healthcheck already waits; this keeps `docker compose run` safe)
i=0
until [ "$(es GET '/_cluster/health?wait_for_status=yellow&timeout=5s')" = "200" ]; do
	i=$((i + 1)); [ $i -gt 60 ] && { echo "es-init: elasticsearch not ready" >&2; exit 1; }
	sleep 2
done

# 1. kibana_system password
must "kibana_system password" "$(es POST /_security/user/kibana_system/_password "{\"password\":\"$(jesc "$KIBANA_SYSTEM_PASSWORD")\"}")"

# 2. abet_ingest: Filebeat (data stream filebeat-*, ILM/template setup, ingest pipelines) + APM Server (traces/logs/metrics-apm*)
must "role abet_ingest" "$(es PUT /_security/role/abet_ingest '{
  "cluster": ["monitor", "manage_ilm", "manage_index_templates", "read_ilm", "manage_pipeline", "read_pipeline", "manage_ingest_pipelines"],
  "indices": [
    {"names": ["filebeat-*", "logs-*", "traces-*", "metrics-*"],
     "privileges": ["auto_configure", "create_doc", "create_index", "view_index_metadata", "read"]},
    {"names": [".apm-agent-configuration", ".apm-source-map"], "privileges": ["read", "view_index_metadata"], "allow_restricted_indices": true}
  ]
}')"
must "user abet_ingest" "$(es PUT /_security/user/abet_ingest "{\"password\":\"$(jesc "$ES_INGEST_PASSWORD")\",\"roles\":[\"abet_ingest\"],\"full_name\":\"ABET log/trace writer\"}")"

# 3. abet_reader: Grafana (read only)
must "role abet_reader" "$(es PUT /_security/role/abet_reader '{
  "cluster": ["monitor"],
  "indices": [
    {"names": ["filebeat-*", "traces-apm*", "logs-apm*"], "privileges": ["read", "view_index_metadata"]}
  ]
}')"
must "user abet_reader" "$(es PUT /_security/user/abet_reader "{\"password\":\"$(jesc "$ES_READER_PASSWORD")\",\"roles\":[\"abet_reader\"],\"full_name\":\"ABET Grafana reader\"}")"

# 4. Retention: forever. One ILM policy without a delete phase, referenced from the @custom component templates of the
#    APM data streams (they are composed last, so they override the built-in policies that delete after 10 days / 1 year).
must "ilm policy abet-keep-forever" "$(es PUT /_ilm/policy/abet-keep-forever '{
  "policy": {"phases": {"hot": {"min_age": "0ms", "actions": {"rollover": {"max_age": "30d", "max_primary_shard_size": "50gb"}}}}}
}')"
#    8.19 also puts a data stream lifecycle (data_retention 10d/90d/180d/390d) in the apm-*d@lifecycle templates:
#    the @custom template switches it off (enabled=false, data_retention=null) so only the delete-free ILM policy remains.
#    The @custom templates are taken from the cluster (all *apm*@custom, created by the apm-data plugin) so patch
#    releases that add streams are covered on the next run.
#    On a fresh cluster the apm-data plugin installs its templates a little after the cluster turns yellow: wait for the
#    index templates (installed last) before touching the @custom component templates.
i=0
until [ "$(es GET /_index_template/traces-apm@template)" = "200" ] && [ "$(es GET /_index_template/metrics-apm.internal@template)" = "200" ]; do
	i=$((i + 1)); [ $i -gt 90 ] && { echo "es-init: apm-data index templates did not appear" >&2; exit 1; }
	sleep 2
done
# apply_customs: PUT the keep-forever settings into every apm @custom template that exists right now.
apply_customs() {
	# the @custom component templates are optional hooks referenced by the apm index templates (they do not exist until created)
	must "list apm index templates" "$(es GET '/_index_template/*apm*?filter_path=index_templates.index_template.composed_of')"
	customs=$(grep -o '"[^"]*apm[^"]*@custom"' "$tmp/out" | tr -d '"' | sort -u)
	[ -n "$customs" ] || { echo "es-init: no apm @custom component templates found (apm-data plugin not ready?)" >&2; exit 1; }
	for ct in $customs; do
		must "component template $ct" "$(es PUT "/_component_template/$ct" \
			'{"template": {"settings": {"index.lifecycle.name": "abet-keep-forever"}, "lifecycle": {"enabled": false, "data_retention": null}}, "_meta": {"description": "ABET: keep APM data forever (D26)"}}')"
	done
}
apply_customs
# second pass after a short pause: picks up templates the plugin was still installing during the first pass
sleep 5
apply_customs
echo "es-init done"
