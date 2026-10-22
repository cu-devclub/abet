#!/bin/sh
# 1. Computes the gateway request-body limit from the same .env values the backend uses (config.yaml reads it as
#    ${{GATEWAY_MAX_BODY_MB}}, APISIX env substitution).
#    GATEWAY_MAX_BODY_MB = max(MAX_UPLOAD_MB, MAX_COVER_MB, MAX_IMPORT_MB) + 20 (multipart overhead headroom).
# 2. Renders the APISIX configuration from the read-only sources in /gateway-src into /usr/local/apisix/conf:
#    config.yaml as is; apisix.yaml with the monitoring parts (/grafana + /kibana routes, opentelemetry global rule and
#    exporter metadata) inserted ONLY when MONITORING_ENABLED=true, otherwise removed (then /grafana* is the 404 JSON).
# Then hands over to the image's original entrypoint.
set -eu

to_int() {
	case "${1:-}" in
		""|*[!0-9]*) echo "$2" ;;
		*) echo "$1" ;;
	esac
}

upload=$(to_int "${MAX_UPLOAD_MB:-}" 300)
cover=$(to_int "${MAX_COVER_MB:-}" 20)
import=$(to_int "${MAX_IMPORT_MB:-}" 5)

max=$upload
[ "$cover" -gt "$max" ] && max=$cover
[ "$import" -gt "$max" ] && max=$import

GATEWAY_MAX_BODY_MB=$((max + 20))
export GATEWAY_MAX_BODY_MB
echo "api-gateway: client_max_body_size = ${GATEWAY_MAX_BODY_MB}m (upload=${upload} cover=${cover} import=${import})"

SRC=${GATEWAY_SRC:-/gateway-src}
DST=${GATEWAY_DST:-/usr/local/apisix/conf}

# Writes apisix.yaml to stdout. The @@MON_*@@ marker lines are replaced by the monitoring snippets (when $1 = true)
# or dropped. Inside the snippets @@KEY@@ is replaced literally (no regex) by the value from the environment;
# backslashes and double quotes are escaped because the templates put the values in double-quoted YAML scalars.
render_apisix() {
	awk -v routes="$SRC/monitoring/routes.yaml" -v grules="$SRC/monitoring/global-rules.yaml" \
		-v meta="$SRC/monitoring/plugin-metadata.yaml" -v on="$1" '
	function esc(v) { gsub(/\\/, "&&", v); gsub(/"/, "\\\\&", v); return v }
	function lit(line, key, val,   out, i) {
		out = ""
		while ((i = index(line, key)) > 0) { out = out substr(line, 1, i - 1) val; line = substr(line, i + length(key)) }
		return out line
	}
	function include(f,   l) {
		while ((getline l < f) > 0) {
			l = lit(l, "@@OTEL_SAMPLE_RATIO@@", ENVIRON["OTEL_RATIO"])
			l = lit(l, "@@DEPLOY_ENV@@", esc(ENVIRON["DEPLOY_ENV"]))
			l = lit(l, "@@APM_SECRET_TOKEN@@", esc(ENVIRON["APM_SECRET_TOKEN"]))
			print l
		}
		close(f)
	}
	/@@MON_ROUTES@@/ { if (on == "true") include(routes); next }
	/@@MON_GLOBAL_RULES@@/ { if (on == "true") include(grules); next }
	/@@MON_METADATA@@/ { if (on == "true") include(meta); next }
	{ print }' "$SRC/apisix.yaml"
}

case "${MONITORING_ENABLED:-false}" in true|TRUE|True|1) MON=true ;; *) MON=false ;; esac
# trace_id_ratio fraction: a plain number 0..1, default 1.0
case "${OTEL_SAMPLE_RATIO:-}" in ""|*[!0-9.]*) OTEL_RATIO=1.0 ;; *) OTEL_RATIO=$OTEL_SAMPLE_RATIO ;; esac
DEPLOY_ENV="${DEPLOY_ENV:-dev}"
APM_SECRET_TOKEN="${APM_SECRET_TOKEN:-}"
export OTEL_RATIO DEPLOY_ENV APM_SECRET_TOKEN

cp "$SRC/config.yaml" "$DST/config.yaml"
# the opentelemetry plugin is not in APISIX's default plugin list: enable it (full list) only with monitoring
if [ "$MON" = "true" ]; then
	printf '\n' >> "$DST/config.yaml"
	cat "$SRC/monitoring/config-plugins.yaml" >> "$DST/config.yaml"
fi
render_apisix "$MON" > "$DST/apisix.yaml"
echo "api-gateway: monitoring=$MON env=$DEPLOY_ENV, rendered $DST/apisix.yaml"

# GATEWAY_RENDER_ONLY=1: render and stop (used to test the rendering without starting APISIX)
[ -z "${GATEWAY_RENDER_ONLY:-}" ] || exit 0

exec /docker-entrypoint.sh "$@"
