.DEFAULT_GOAL := help

# Windows: the recipes need a POSIX shell. When make is started from PowerShell/cmd it would otherwise use cmd.exe,
# so use Git for Windows' sh (8.3 path: make's wildcard cannot handle the space in "Program Files").
ifeq ($(OS),Windows_NT)
GIT_SH := $(firstword $(wildcard C:/Progra~1/Git/bin/sh.exe C:/Progra~1/Git/usr/bin/sh.exe C:/Progra~2/Git/bin/sh.exe))
ifneq ($(GIT_SH),)
SHELL := $(GIT_SH)
# make runs simple recipe lines without the shell, so the POSIX tools (printf, sed, grep ...) must be on PATH too
export PATH := $(dir $(GIT_SH))../usr/bin;$(PATH)
else
$(warning Git for Windows sh.exe not found - run make from Git Bash)
endif
endif
.PHONY: help env dev dev-logs down prod prod-down logs ps restart worker-restart build test lint db-shell rabbit-ui minio-console clean config dev-session seed e2e mon-urls mon-prune

# ENV selects the stack for every ops target: `make logs ENV=prod s=main-service`. dev (default) | prod.
# dev and prod are separate compose projects (<name>-dev / <name>), so they never share containers or volumes.
ENV ?= dev

# Only a few keys are read from .env here (a plain sed instead of `-include .env`: passwords may contain
# characters make would mangle, and exporting them would override what compose reads from .env itself).
env_value = $(shell sed -n 's/^$(1)=//p' .env 2>/dev/null | tail -n 1 | tr -d '\r"'"'"' ')
COMPOSE_PROJECT_NAME ?= $(or $(call env_value,COMPOSE_PROJECT_NAME),abet)

ifeq ($(ENV),dev)
COMPOSE_FILES := -f compose.yaml -f compose.dev.yaml
PROJECT := $(COMPOSE_PROJECT_NAME)-dev
else ifeq ($(ENV),prod)
COMPOSE_FILES := -f compose.yaml -f compose.prod.yaml
PROJECT := $(COMPOSE_PROJECT_NAME)
else
$(error ENV must be 'dev' or 'prod' (got '$(ENV)'))
endif

# Monitoring (optional compose profile "monitoring"): on when MON=1 on the command line or MONITORING_ENABLED=true in .env.
# MON=0 forces it off. The resolved value is exported so compose (gateway, OTEL_ENABLED of the services) sees it.
ifeq ($(MON),1)
MONITORING_ENABLED := true
else ifeq ($(MON),0)
MONITORING_ENABLED := false
else ifeq ($(call env_value,MONITORING_ENABLED),true)
MONITORING_ENABLED := true
else
MONITORING_ENABLED := false
endif
export MONITORING_ENABLED
# Consumed by compose.yaml: environment label (dev|prod) and the compose project (Filebeat only ships this project's logs).
export DEPLOY_ENV := $(ENV)
export ABET_PROJECT := $(PROJECT)

MON_SERVICES := elasticsearch es-init kibana kibana-init apm-server filebeat prometheus cadvisor node-exporter postgres-mon-init postgres-exporter grafana
MON_KEYS := ELASTIC_PASSWORD KIBANA_SYSTEM_PASSWORD ES_INGEST_PASSWORD ES_READER_PASSWORD KIBANA_ENCRYPTION_KEY APM_SECRET_TOKEN GRAFANA_ADMIN_PASSWORD POSTGRES_MONITOR_PASSWORD

# DC honours the monitoring switch; DC_ALL always includes the profile (down/clean/ps/logs never leave monitoring containers behind).
DC_ALL = docker compose -p $(PROJECT) $(COMPOSE_FILES) --profile monitoring
ifeq ($(MONITORING_ENABLED),true)
DC = $(DC_ALL)
else
DC = docker compose -p $(PROJECT) $(COMPOSE_FILES)
endif

# Usage: $(call require_dev) as the first recipe line of a dev-only target.
define require_dev
@if [ "$(ENV)" != "dev" ]; then echo "make $@ is DEV ONLY - refusing to run with ENV=$(ENV)" >&2; exit 1; fi
endef

help: ## Show this help
	@printf '\nUsage:\n  make \033[36m<target>\033[0m [ENV=dev|prod] [MON=1] [s=<service>]\n\n'
	@printf 'ENV selects the stack (default dev): dev -> project "%s-dev", prod -> project "%s".\n' '$(COMPOSE_PROJECT_NAME)' '$(COMPOSE_PROJECT_NAME)'
	@printf 'Example: make logs ENV=prod s=main-service   |   shortcuts: dev, dev-logs, prod, prod-down\n'
	@printf 'Monitoring (Grafana + Kibana): MON=1 for one run (make dev MON=1) or MONITORING_ENABLED=true in .env; MON=0 forces it off.\n\nTargets:\n'
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

env: ## Create .env from .env.template if it does not exist (then edit the secrets!)
	@if [ -f .env ]; then echo ".env already exists - not touching it"; else cp .env.template .env && echo "Created .env from .env.template - replace every 'change-me' and fill GOOGLE_CLIENT_ID/SECRET, SUPER_ADMIN_EMAILS"; fi

dev: ## Start the DEV stack (hot reload) in the background: http://localhost:$GATEWAY_HTTP_PORT  (MON=1 adds monitoring)
	$(MAKE) --no-print-directory ENV=dev _up

prod: ## Build and start the PRODUCTION stack in the background  (MON=1 adds monitoring)
	$(MAKE) --no-print-directory ENV=prod _up

# Host paths bind-mounted into containers that run as a non-root user (curl_user, grafana 472, apisix, postgres ...).
MOUNTED_DIRS := deploy api-gateway portfolio-service/templates

.PHONY: _up _preflight
# Linux: a copy that did not come from git clone (zip, scp, WinSCP ...) can leave directories at 700, which the
# non-root containers cannot enter ("can't open '/init/es-init.sh': Permission denied"); add world read/traverse.
# Windows (Docker Desktop) ignores host modes, so the chmod is skipped there. CRLF breaks /bin/sh on every OS.
_preflight:
ifneq ($(OS),Windows_NT)
	@chmod -R a+rX $(MOUNTED_DIRS)
endif
	@crlf=$$(grep -rlI "$$(printf '\r')" --include='*.sh' $(MOUNTED_DIRS) 2>/dev/null); \
	if [ -n "$$crlf" ]; then printf '%s\n' "these shell scripts have CRLF line endings, convert them to LF (e.g. sed -i 's/\r$$//' <file>):" $$crlf >&2; exit 1; fi

_up: _preflight
ifeq ($(MONITORING_ENABLED),true)
	@missing=""; for k in $(MON_KEYS); do v=$$(sed -n "s/^$$k=//p" .env 2>/dev/null | tail -n 1); [ -n "$$v" ] || missing="$$missing $$k"; done; \
	if [ -n "$$missing" ]; then echo "monitoring is on but these keys are missing/empty in .env:$$missing (see .env.template)" >&2; exit 1; fi
else
	@# monitoring off: stop + remove monitoring containers of this stack left from an earlier MON=1 run (volumes are kept)
	@$(DC_ALL) rm -sf $(MON_SERVICES) >/dev/null 2>&1 || true
endif
	$(DC) up -d --build

dev-logs: ## Follow dev logs (all services)
	$(MAKE) --no-print-directory ENV=dev logs

prod-down: ## Stop the production stack (keeps volumes)
	$(MAKE) --no-print-directory ENV=prod down

down: ## Stop the stack for ENV incl. monitoring containers (keeps volumes)
	$(DC_ALL) down --remove-orphans

logs: ## Follow logs (ENV stack, incl. monitoring containers): make logs [s=main-service]
	$(DC_ALL) logs -f --tail=200 $(s)

ps: ## Show service status (ENV stack, incl. monitoring containers)
	$(DC_ALL) ps -a

restart: ## Restart one service (ENV stack): make restart s=auth-service
	@test -n "$(s)" || (echo "usage: make restart s=<service> [ENV=dev|prod]" && exit 1)
	$(DC_ALL) restart $(s)

worker-restart: ## Restart all portfolio-service workers (ENV stack) - e.g. after a hung build
	$(DC) restart portfolio-service

build: ## Build all images (ENV files; MON=1 includes the monitoring images)
	$(DC) build

test: ## Run unit tests of auth-service, main-service and frontend (skips what is missing)
	@for d in auth-service main-service; do \
		if [ -f $$d/go.mod ]; then echo "== go test $$d"; (cd $$d && go test ./... -count=1) || exit 1; else echo "== skip $$d (no go.mod)"; fi; \
	done
	@if [ -f frontend/package.json ]; then echo "== npm test frontend"; (cd frontend && npm test --if-present) || exit 1; else echo "== skip frontend (no package.json)"; fi

lint: ## go vet (Go services) and npm run lint (frontend); skips what is missing
	@for d in auth-service main-service; do \
		if [ -f $$d/go.mod ]; then echo "== go vet $$d"; (cd $$d && go vet ./...) || exit 1; else echo "== skip $$d (no go.mod)"; fi; \
	done
	@if [ -f frontend/package.json ]; then echo "== npm run lint frontend"; (cd frontend && npm run lint --if-present) || exit 1; else echo "== skip frontend (no package.json)"; fi

db-shell: ## psql shell into main_db (ENV stack)
	$(DC) exec postgres sh -c 'PGPASSWORD="$$POSTGRES_PASSWORD" psql -U "$$POSTGRES_USER" -d "$$MAIN_DB_NAME"'

dev-session: ## DEV ONLY: print an `abet_session=<jwt>` cookie for a provisioned user: make dev-session email=admin1@chula.ac.th
	$(call require_dev)
	@test -n "$(email)" || (echo "usage: make dev-session email=<e-mail>" >&2 && exit 1)
	@$(DC) exec -T auth-service go run ./cmd/devsession "$(email)"

seed: ## DEV ONLY: load idempotent test data (departments, curricula, users of every role, offerings) into main_db
	$(call require_dev)
	$(DC) exec -T postgres sh -c 'PGPASSWORD="$$POSTGRES_PASSWORD" psql -v ON_ERROR_STOP=1 -q -U "$$POSTGRES_USER" -d "$$MAIN_DB_NAME"' < deploy/dev/seed.sql

e2e: ## DEV ONLY: run the Playwright suite against the running dev stack (BASE_URL, default http://localhost:8000)
	$(call require_dev)
	cd frontend && npm run e2e

minio-console: ## Print the MinIO console URL (published in dev and prod; login: MINIO_ROOT_USER / MINIO_ROOT_PASSWORD from .env)
	@port=$$(grep -E '^MINIO_CONSOLE_HOST_PORT=' .env 2>/dev/null | cut -d= -f2 | tr -d '"'); echo "MinIO console ($(ENV)): http://localhost:$${port:-9001}  (login: MINIO_ROOT_USER / MINIO_ROOT_PASSWORD from .env)"

rabbit-ui: ## Print the RabbitMQ management UI URL (dev stack publishes it)
	@port=$$(grep -E '^RABBITMQ_MGMT_HOST_PORT=' .env 2>/dev/null | cut -d= -f2); echo "RabbitMQ UI: http://localhost:$${port:-15672}  (login: RABBITMQ_USER / RABBITMQ_PASS from .env)"

mon-urls: ## Print the Grafana and Kibana URLs and login user names (never passwords)
	@base="$(call env_value,PUBLIC_BASE_URL)"; base=$${base:-http://localhost:8000}; guser="$(call env_value,GRAFANA_ADMIN_USER)"; \
	echo "Grafana (metrics):       $$base/grafana/   user: $${guser:-admin}   (password: GRAFANA_ADMIN_PASSWORD in .env)"; \
	echo "Kibana (logs + traces):  $$base/kibana/    user: elastic   (password: ELASTIC_PASSWORD in .env)"; \
	echo "Monitoring for this command: $(MONITORING_ENABLED)  (switch on: make $(if $(filter prod,$(ENV)),prod,dev) MON=1, or MONITORING_ENABLED=true in .env)"

mon-prune: ## Delete logs + traces older than N days from Elasticsearch (asks first): make mon-prune days=90 [ENV=prod]
	@case "$(days)" in ''|*[!0-9]*) echo "usage: make mon-prune days=<N> [ENV=dev|prod]" >&2; exit 1 ;; esac
	@printf "Delete ALL logs and traces older than %s days from Elasticsearch (%s stack, project '%s')? Type 'yes' to continue: " "$(days)" "$(ENV)" "$(PROJECT)"; read ans; \
	if [ "$$ans" = "yes" ]; then MSYS_NO_PATHCONV=1 $(DC_ALL) run --rm --no-deps -T -e DAYS=$(days) --entrypoint sh es-init /init/prune.sh; else echo "aborted"; fi

clean: ## DANGER: stop the ENV stack and DELETE its volumes (database, queues, files, monitoring data) - asks first
	@printf "This deletes ALL data volumes of the %s stack (project '%s': postgres, rabbitmq, minio, and the monitoring data: Elasticsearch logs/traces, Prometheus, Grafana). Type 'yes' to continue: " "$(ENV)" "$(PROJECT)"; read ans; \
	if [ "$$ans" = "yes" ]; then $(DC_ALL) down -v --remove-orphans; else echo "aborted"; fi

config: ## Validate the merged compose config of the ENV stack (ENV=prod, MON=1 supported)
	$(DC) config -q && echo "$(ENV) config OK (project $(PROJECT), monitoring $(MONITORING_ENABLED))"
