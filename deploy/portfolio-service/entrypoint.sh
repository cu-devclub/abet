#!/bin/sh
# Deploy shim for portfolio-service (decision A6).
# settings.json values override environment variables inside config.py, so rewrite the
# relevant keys of /app/settings.json from the container environment, then start the worker.
# Other keys (e.g. LAYOUT_PRESETS) are preserved. The edit lives only inside the container.
set -eu

cd /app

python3 - <<'PY'
import json
import os

path = "settings.json"
with open(path, "r", encoding="utf-8") as f:
    settings = json.load(f)

overrides = {
    "RABBITMQ_HOST": os.environ.get("RABBITMQ_HOST"),
    "RABBITMQ_PORT": os.environ.get("RABBITMQ_PORT"),
    "RABBITMQ_USER": os.environ.get("RABBITMQ_USER"),
    "RABBITMQ_PASS": os.environ.get("RABBITMQ_PASS"),
    "RABBITMQ_VHOST": os.environ.get("RABBITMQ_VHOST"),
    "RABBITMQ_QUEUE": os.environ.get("RABBITMQ_QUEUE"),
    "RABBITMQ_STATUS_QUEUE": os.environ.get("RABBITMQ_STATUS_QUEUE"),
    "DEFAULT_OUTPUT_DIR": os.environ.get("DEFAULT_OUTPUT_DIR", "/var/storage/output_pdfs"),
    "DELETE_ZIP_DEFAULT": os.environ.get("DELETE_ZIP_DEFAULT"),
    "LOG_LEVEL": os.environ.get("LOG_LEVEL"),
}

for key, value in overrides.items():
    if value is None or value == "":
        continue
    if key == "RABBITMQ_PORT":
        value = int(value)
    elif key == "DELETE_ZIP_DEFAULT":
        value = value.strip().lower() in ("true", "1", "yes")
    elif key == "LOG_LEVEL":
        value = value.upper()
    settings[key] = value

with open(path, "w", encoding="utf-8") as f:
    json.dump(settings, f, indent=2)
print("entrypoint: settings.json rewritten from environment", flush=True)
PY

exec python3 -u worker.py
