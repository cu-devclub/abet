# Service diagrams

Back to the [documentation index](../../README.md).

## (a) Containers and routes

The browser only ever talks to the gateway. In production **only the gateway port is published** (`GATEWAY_HTTP_PORT`, container port 9080);
the MinIO console (`MINIO_CONSOLE_HOST_PORT`, container 9001) is published too (dev and prod; the S3 API on 9000 never in prod); in dev Postgres (`POSTGRES_HOST_PORT`) and the RabbitMQ management UI (`RABBITMQ_MGMT_HOST_PORT`) are published as well.

```mermaid
flowchart LR
    Browser([Browser])

    subgraph docker["docker compose network"]
        GW["api-gateway<br/>APISIX :9080"]
        FE["frontend<br/>nginx :80 (prod) / vite :5173 (dev)"]
        AUTH["auth-service :8080<br/>Google login, sessions"]
        MAIN["main-service :8080<br/>domain API, validation,<br/>queue producer + consumer"]
        PORT["portfolio-service<br/>Python worker (LaTeX)<br/>no port"]
        PG[("postgres :5432<br/>auth_db + main_db")]
        MQ[["rabbitmq :5672<br/>course_build_queue<br/>course_status_queue"]]
        S3[("minio :9000 (S3, internal)<br/>bucket abet<br/>console :9001")]
    end

    Google([Google OIDC])

    Browser -->|"HTTP :GATEWAY_HTTP_PORT"| GW
    GW -->|"/* (priority 0)"| FE
    GW -->|"/public/ allowlist: google login,<br/>google callback, logout (prefix stripped)<br/>everything else 404 at the gateway"| AUTH
    GW -->|"/auth/api/* (prefix stripped)<br/>forward-auth first"| MAIN
    GW -.->|"forward-auth: GET /internal/v1/auth/verify<br/>forwards Cookie only (internal route)"| AUTH
    AUTH -->|"HTTPS: code exchange, JWKS"| Google
    AUTH -->|"POST /internal/v1/auth/login-sync<br/>X-Internal-Token"| MAIN
    AUTH --> PG
    MAIN --> PG
    MAIN -->|"publish job"| MQ
    MQ -->|"consume job"| PORT
    PORT -->|"publish result"| MQ
    MQ -->|"consume status"| MAIN
    MAIN -->|"S3: put/get/copy/delete"| S3
    PORT -->|"S3: read zip + cover,<br/>write outputs"| S3
    Browser -.->|"console :MINIO_CONSOLE_HOST_PORT<br/>(root login)"| S3
```

**Optional monitoring profile** (see [monitoring.md](../../monitoring.md)): `/grafana/*` and `/kibana/*` are extra gateway routes (cookies `abet_session` / `abet_oauth` and `X-User-*` removed); the services export OpenTelemetry traces to `apm-server:8200`; Prometheus scrapes the services' `/metrics`, the gateway, RabbitMQ, MinIO, Postgres exporter, cAdvisor and node-exporter; Filebeat ships all container logs to Elasticsearch.

```mermaid
flowchart LR
    GW2[api-gateway] -.->|"/grafana"| GRAF[grafana]
    GW2 -.->|"/kibana"| KIB[kibana]
    GW2 & AUTH2[auth-service] & MAIN2[main-service] & PORT2[portfolio worker] -.->|"OTLP/HTTP + token"| APM[apm-server]
    APM --> ES[(elasticsearch)]
    FB[filebeat] --> ES
    KIB --> ES
    PROM[prometheus] -.->|scrape| AUTH2 & MAIN2 & GW2
    GRAF --> PROM
```

| Browser path | Goes to | Auth |
|---|---|---|
| `/public/api/v1/auth/google/login`, `/public/api/v1/auth/google/callback`, `/public/api/v1/auth/logout` | auth-service (`/public` removed) | none |
| any other `/public/*` | answered by the gateway: `404 {"error":"not found"}` | none |
| `/auth/api/*` | main-service (`/auth` removed) | forward-auth against auth-service; gateway sets `X-User-Id`, `X-User-Email` |
| everything else | frontend (SPA, static assets) | none (the SPA itself redirects to `/login`) |

Not routed from outside: auth-service `/internal/*`, `/healthz`, `/readyz`, `/metrics` and main-service `/internal/*`, `/healthz`, `/readyz`, `/metrics`, swagger (`/auth/...` paths other than `/auth/api/*` answer `404` JSON). The portfolio worker runs as `PORTFOLIO_WORKER_REPLICAS` (default 2) containers consuming the same queue. Routes: [`api-gateway/apisix.prod.yaml`](../../../api-gateway/apisix.prod.yaml),
[`apisix.dev.yaml`](../../../api-gateway/apisix.dev.yaml).

Object layout in the MinIO bucket (created by the one-shot `minio-init`, which also creates the bucket-limited service account used by main-service and the worker): `uploads/`, `outputs/`, `portfolios/`, `covers/` — see [operations.md](../../operations.md#storage-layout-and-cleanup). There is no shared volume between main-service and the worker.

## (b) Google login and forward-auth on every request

```mermaid
sequenceDiagram
    autonumber
    actor U as Browser
    participant GW as api-gateway
    participant A as auth-service
    participant G as Google
    participant M as main-service
    participant ADB as auth_db
    participant MDB as main_db

    U->>GW: GET /public/api/v1/auth/google/login?redirect=/portfolios
    GW->>A: GET /api/v1/auth/google/login
    A-->>U: 302 to Google (state + PKCE S256, hd=first allowed domain)<br/>Set-Cookie abet_oauth (HttpOnly, 10 min)
    U->>G: sign in, consent
    G-->>U: 302 /public/api/v1/auth/google/callback?code&state
    U->>GW: GET callback (cookie abet_oauth)
    GW->>A: GET /api/v1/auth/google/callback
    A->>G: exchange code (PKCE), verify ID token
    Note over A: email_verified=true and exact domain in ALLOWED_EMAIL_DOMAINS
    A->>M: POST /internal/v1/auth/login-sync (X-Internal-Token)
    M->>MDB: find user by email, update name/picture/last_login_at
    alt user is provisioned
        M-->>A: 200 user_id, email, role
        A->>ADB: INSERT session (id = JWT sid)
        A-->>U: 302 PUBLIC_BASE_URL + redirect<br/>Set-Cookie abet_session (HS256 JWT, HttpOnly, SameSite=Lax)
    else not provisioned or domain rejected
        M-->>A: 403 not provisioned
        A-->>U: 302 /login?error=access_denied
    end

    Note over U,MDB: every later API call
    U->>GW: GET /auth/api/v1/portfolios (Cookie abet_session)
    GW->>GW: clear client X-User-Id / X-User-Email
    GW->>A: forward-auth GET /internal/v1/auth/verify (Cookie)
    A->>ADB: JWT check + session row (not revoked, not expired)
    alt valid
        A-->>GW: 200 + X-User-Id, X-User-Email
        GW->>M: GET /api/v1/portfolios + X-User-Id, X-User-Email
        M->>MDB: load actor (role, scope) on every request
        M-->>U: 200 JSON
    else invalid
        A-->>GW: 401
        GW-->>U: 401 {"error":"unauthenticated"}
    end
```

Logout: `POST /public/api/v1/auth/logout` marks the session row revoked and clears the cookie, so a copied cookie stops working immediately.

## (c) Portfolio upload, validation, queue, worker, status, download

```mermaid
sequenceDiagram
    autonumber
    actor U as Browser
    participant GW as api-gateway
    participant M as main-service
    participant DB as main_db
    participant ST as minio (bucket)
    participant Q as rabbitmq
    participant W as portfolio-service

    U->>GW: POST /auth/api/v1/offerings/{id}/submissions (multipart zip)
    GW->>M: forward-auth OK, same request
    M->>M: authorize upload on the offering
    M->>M: stream body to a local temp file (UPLOAD_TMP_DIR, limit MAX_UPLOAD_MB)
    M->>M: zip opens? run immediate validator (SRS 7) on the temp file
    alt errors found
        M->>DB: INSERT submission (invalid) + issues
        M-->>U: 201 status=invalid + issues (nothing stored, no job queued)
    else no errors (warnings allowed)
        M->>ST: PutFile uploads/{submission_id}.zip
        M->>DB: one tx: INSERT submission (queued) + issues
        M->>Q: publish v2 job to course_build_queue (persistent)
        alt publish failed
            M->>ST: delete uploads/{submission_id}.zip
            M->>DB: status=failed, issue QUEUE_UNAVAILABLE
            M-->>U: 201 status=failed
        else published
            M-->>U: 201 status=queued (UI shows Processing)
        end
    end

    Q->>W: job {job_id, program_name, storage:s3, zip_key, cover_key?, output_prefix}
    W->>ST: download zip (and covers/{curriculum_id}.pdf) into a temp dir, build PDF
    W->>ST: upload outputs/{submission_id}/* , delete uploads/{submission_id}.zip
    W->>Q: publish to course_status_queue {status: success|failed, job_id, exported_pdf_keys, ...}
    Q->>M: status event (prefetch 10, manual ack)
    M->>ST: server-side copy first *-main.pdf key to portfolios/{submission_id}.pdf, stat, delete outputs/{submission_id}/,<br/>delete PDFs of older generated submissions
    M->>DB: status=generated (or failed + parsed issues), then ack

    loop UI polls every 5 s while queued
        U->>GW: GET /auth/api/v1/portfolios
        GW->>M: list (latest_submission.status)
    end
    U->>GW: GET /auth/api/v1/offerings/{id}/portfolio.pdf
    GW->>M: authorize download
    M->>ST: Get portfolios/{submission_id}.pdf (streamed)
    M-->>U: 200 application/pdf (attachment, Content-Length)
```

Bulk download: `POST /auth/api/v1/portfolios/download` streams one zip with every PDF the caller may download; missing or forbidden ones are listed in `MISSING.txt`.
Message formats: [operations.md](../../operations.md#queues-and-message-formats). Validation rules: [business-rules.md](../../business-rules.md#upload-validation).

## (d) Mapping CSV import: preview, then apply

```mermaid
sequenceDiagram
    autonumber
    actor A as Admin (browser)
    participant GW as api-gateway
    participant M as main-service
    participant DB as main_db
    participant ST as minio (bucket)

    Note over A: Courses page filters choose curriculum + academic year (the CSV has no year column)
    A->>GW: POST /auth/api/v1/offerings/import/preview (file, curriculum_id, academic_year)
    GW->>M: authenticated request
    M->>M: authorize: super_admin or admin with curriculum in scope
    M->>M: ParseCSV (all rows validated, row 1 = header)
    M->>DB: read state of the scope (offerings, access, users) - no writes
    M->>M: BuildImportPlan (pure function)
    M-->>A: 200 ImportPlan {applied:false, summary, courses, access, users, errors}
    Note over A: Apply button is disabled while errors exist

    A->>GW: POST /auth/api/v1/offerings/import/apply (same file + scope)
    GW->>M: authenticated request
    M->>M: ParseCSV again (never trusts the preview)
    alt row errors
        M-->>A: 422 ImportPlan with errors, nothing applied
    else clean
        M->>DB: BEGIN, lock scope offerings (SELECT FOR UPDATE)
        M->>DB: load state, BuildImportPlan, write plan (users, courses, offerings, access)
        M->>DB: COMMIT
        M->>ST: after commit, best effort: delete objects of deleted offerings
        M-->>A: 200 ImportPlan {applied:true}
    end
```

The file itself is never stored (D3). Rules (create / update / keep / delete / keep_has_uploads, access mirror, new users): [business-rules.md](../../business-rules.md#mapping-csv-import).
