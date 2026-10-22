# api-gateway (Apache APISIX 3.13.0, standalone)

Single entry point of the system. APISIX runs in **standalone / data-plane** mode: routes are declared in a
YAML file (`apisix.dev.yaml` or `apisix.prod.yaml`), there is no etcd and no Admin API.

| file | purpose |
|---|---|
| `config.yaml` | APISIX runtime config: `data_plane` role, `yaml` provider, `client_max_body_size ${{GATEWAY_MAX_BODY_MB}}m` (= max(MAX_UPLOAD_MB, MAX_COVER_MB, MAX_IMPORT_MB) + 20 MB, computed from `.env` at container start), 300 s proxy timeouts |
| `../deploy/api-gateway/entrypoint.sh` | wrapper entrypoint: computes `GATEWAY_MAX_BODY_MB` from the size limits and `exec`s the image's `/docker-entrypoint.sh docker-start` (compose sets `entrypoint` + `command: [docker-start]`) |
| `apisix.dev.yaml` | routes for `compose.dev.yaml` (frontend = Vite dev server `frontend:5173`, websocket enabled for HMR) |
| `apisix.prod.yaml` | routes for `compose.prod.yaml` (frontend = nginx `frontend:80`) |

The compose override mounts the right file as `/usr/local/apisix/conf/apisix.yaml`. Both files must end with `#END`
(APISIX uses it to detect a fully written file). APISIX reloads the file automatically when it changes; no restart is needed.

## Routes

| # | browser path | priority | what happens | upstream |
|---|---|---|---|---|
| 1 | `/public/api/v1/auth/google/login`, `/public/api/v1/auth/google/callback` (GET), `/public/api/v1/auth/logout` (POST) | 100 | **allowlist**, no authentication; `proxy-rewrite` strips `/public` | `auth-service:8080` |
| 1b | `/public`, `/public/*`, `/auth`, `/auth/*` (everything not matched above/below) | 50 | `fault-injection` abort: **404** `{"error":"not found"}` (JSON, never the SPA) | – |
| 2 | `/auth/api/*` | 100 | **forward-auth** against `http://auth-service:8080/internal/v1/auth/verify` (forwards only the `Cookie` header). On 200 the response headers `X-User-Id`, `X-User-Email` are set on the upstream request; any other status (normally 401) is returned to the browser. Then `proxy-rewrite` strips `/auth` (`/auth/api/v1/me` → `/api/v1/me`) | `main-service:8080` |
| 3 | `/*` | 0 | everything else (SPA, assets) | `frontend` |

Not routed (reachable only inside the docker network): main-service `/internal/*`, `/healthz`, `/readyz`, `/metrics`, swagger.
`/auth/internal/...`, `/auth/metrics`, `/auth/healthz` and `/public/healthz|readyz|metrics|health|api/v1/auth/verify` answer 404 JSON from the gateway (route 1b).

## Identity header spoofing (verified)

* `forward-auth` copies `upstream_headers` from the auth response with `core.request.set_header` (overwrite semantics, checked in
  `apisix/plugins/forward-auth.lua` of the 3.13.0 image). auth-service **always** sets both headers on 200, so a client-supplied
  `X-User-Id` / `X-User-Email` is always overwritten on authenticated requests; unauthenticated requests never reach the upstream.
* Belt and braces: route 2 also runs `serverless-pre-function` (rewrite phase, priority 10000 — runs **before** forward-auth at 2002)
  which clears `X-User-Id` / `X-User-Email` from the client request.
  `proxy-rewrite.headers.remove` must NOT be used for this: `proxy-rewrite` (priority 1008) runs *after* forward-auth and would delete the
  verified headers again.
* Tested with curl against a fake upstream: request with a valid cookie and spoofed `X-User-Id: spoofed` reached the upstream with the real user id.

`client_headers: ["Content-Type"]` is set on forward-auth so that the 401 body from auth-service keeps `Content-Type: application/json`.

## Uploads

`client_max_body_size` (nginx, `config.yaml`) is `${{GATEWAY_MAX_BODY_MB}}m`, computed by `deploy/api-gateway/entrypoint.sh` as
`max(MAX_UPLOAD_MB, MAX_COVER_MB, MAX_IMPORT_MB) + 20` from `.env` — nothing to edit here, just `make dev`/`make prod` after changing `.env`.
Per-upstream timeouts: `connect 60 s / send 300 s / read 300 s`.

## Changing routes / debugging

Edit the YAML, keep `#END` last. Logs: `docker compose logs -f api-gateway`. The `forward-auth uri with no TLS` warning at startup is
expected (service-to-service traffic inside the private docker network).

## Monitoring routes and tracing (optional, `MONITORING_ENABLED=true`)

`deploy/api-gateway/entrypoint.sh` renders `/usr/local/apisix/conf/apisix.yaml` from the mounted per-env file
(`/gateway-src/apisix.yaml`) and the snippets in `api-gateway/monitoring/`:

| snippet | rendered when | content |
|---|---|---|
| `routes.yaml` | monitoring on | `/grafana`, `/grafana/*` -> grafana:3000 and `/kibana`, `/kibana/*` -> kibana:5601 (priority 100, websocket), `/grafana/metrics` -> 404. A serverless pre-function removes the `abet_session` / `abet_oauth` cookies and every `X-User-*` header before proxying. |
| `global-rules.yaml`, `plugin-metadata.yaml`, `config-plugins.yaml` | monitoring on | `opentelemetry` global rule (parent-based, `trace_id_ratio` = `OTEL_SAMPLE_RATIO`), exporter -> `apm-server:8200` with the APM bearer token, resource `service.name=api-gateway`. The plugin is not in APISIX's default plugin list, so the full list is appended to `config.yaml`. |

Always on: the `prometheus` global rule (exporter on `:9091`, never routed/published) and `monitoring-not-found`
(priority 50), which answers `/grafana*` and `/kibana*` with the 404 JSON when monitoring is off.
`forward-auth` forwards `traceparent`/`tracestate` to auth-service so `/internal/v1/auth/verify` joins the request trace.
