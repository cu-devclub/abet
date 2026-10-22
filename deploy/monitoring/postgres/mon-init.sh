#!/bin/sh
# One-shot, idempotent: least-privilege Postgres role for postgres-exporter (pg_monitor only).
# Env: PGHOST, PGUSER, PGPASSWORD (superuser, libpq reads them), POSTGRES_MONITOR_PASSWORD. No password in a URL.
set -eu
: "${POSTGRES_MONITOR_PASSWORD:?POSTGRES_MONITOR_PASSWORD is required}"
psql -v ON_ERROR_STOP=1 -q -d postgres -v mon_pw="$POSTGRES_MONITOR_PASSWORD" <<'SQL'
SELECT format('CREATE ROLE abet_monitor LOGIN PASSWORD %L', :'mon_pw')
 WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'abet_monitor') \gexec
SELECT format('ALTER ROLE abet_monitor LOGIN PASSWORD %L', :'mon_pw') \gexec
GRANT pg_monitor TO abet_monitor;
SQL
echo "postgres-mon-init done"
