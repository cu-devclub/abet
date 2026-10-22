#!/bin/sh
# Creates the per-service databases on first start of the postgres container.
# Runs only when the data directory is empty (docker-entrypoint-initdb.d semantics).
set -eu

for db in "${AUTH_DB_NAME:?AUTH_DB_NAME is required}" "${MAIN_DB_NAME:?MAIN_DB_NAME is required}"; do
  echo "init-databases: creating database ${db}"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-SQL
	SELECT 'CREATE DATABASE "${db}" OWNER "${POSTGRES_USER}"'
	WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '${db}')\gexec
SQL
done
