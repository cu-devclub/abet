import json
import logging
import os
from pathlib import Path

# Base Paths - points to ABET_Processor directory
BASE_DIR = Path(__file__).resolve().parent
SETTINGS_JSON_PATH = BASE_DIR / "settings.json"

# Logging setup for config loading
logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] [Config] %(message)s")

# Initial MariaDB Credentials (from Environment Variables)
MARIADB_HOST = os.getenv("MARIADB_HOST", "abet-mariadb")
MARIADB_PORT = int(os.getenv("MARIADB_PORT", "3306"))
MARIADB_USER = os.getenv("MARIADB_USER", "root")
MARIADB_PASS = os.getenv("MARIADB_PASS", "password")
MARIADB_DB = os.getenv("MARIADB_DB", "abet_db")
MARIADB_TABLE = os.getenv("MARIADB_TABLE", "app_settings")


def load_settings_from_json() -> dict:
    """Fallback: Load configuration settings from settings.json."""
    if SETTINGS_JSON_PATH.exists():
        try:
            with open(SETTINGS_JSON_PATH, "r", encoding="utf-8") as f:
                data = json.load(f)
                logging.info(f"Loaded fallback configuration from {SETTINGS_JSON_PATH.name}")
                return data
        except Exception as exc:
            logging.warning(f"Failed to read {SETTINGS_JSON_PATH.name}: {exc}")
    return {}


def load_settings_from_mariadb() -> dict:
    """Primary: Attempt to load configuration directly from MariaDB database."""
    try:
        import pymysql

        logging.info(f"Connecting to MariaDB at {MARIADB_HOST}:{MARIADB_PORT} (DB: '{MARIADB_DB}')...")
        conn = pymysql.connect(
            host=MARIADB_HOST,
            port=MARIADB_PORT,
            user=MARIADB_USER,
            password=MARIADB_PASS,
            database=MARIADB_DB,
            connect_timeout=3,
            cursorclass=pymysql.cursors.DictCursor,
        )

        settings = {}
        with conn.cursor() as cursor:
            # Query key-value format (setting_key, setting_value)
            try:
                cursor.execute(f"SELECT setting_key, setting_value FROM {MARIADB_TABLE}")
                rows = cursor.fetchall()
                for row in rows:
                    k = row.get("setting_key")
                    v = row.get("setting_value")
                    if k:
                        settings[k] = v
            except Exception:
                # If key-value query fails, try single-row column query (SELECT * FROM table LIMIT 1)
                cursor.execute(f"SELECT * FROM {MARIADB_TABLE} LIMIT 1")
                row = cursor.fetchone()
                if row:
                    settings = {k: v for k, v in row.items() if v is not None}

        conn.close()
        if settings:
            logging.info(f"Successfully loaded {len(settings)} settings from MariaDB database '{MARIADB_DB}'!")
            return settings

    except ImportError:
        logging.warning("pymysql package not installed. Skipping MariaDB config lookup.")
    except Exception as exc:
        logging.warning(f"MariaDB unavailable ({exc}). Falling back to settings.json...")

    return {}


# 1. Try loading from MariaDB database
effective_config = load_settings_from_mariadb()

# 2. Fallback to settings.json if MariaDB configuration is not available
if not effective_config:
    effective_config = load_settings_from_json()

# Overwrite DB connection parameters from settings.json if present
MARIADB_HOST = effective_config.get("MARIADB_HOST", MARIADB_HOST)
MARIADB_PORT = int(effective_config.get("MARIADB_PORT", MARIADB_PORT))
MARIADB_USER = effective_config.get("MARIADB_USER", MARIADB_USER)
MARIADB_PASS = effective_config.get("MARIADB_PASS", MARIADB_PASS)
MARIADB_DB = effective_config.get("MARIADB_DB", MARIADB_DB)
MARIADB_TABLE = effective_config.get("MARIADB_TABLE", MARIADB_TABLE)

# RabbitMQ Settings
RABBITMQ_HOST = effective_config.get("RABBITMQ_HOST") or os.getenv("RABBITMQ_HOST", "abet-rabbitmq")
RABBITMQ_PORT = int(effective_config.get("RABBITMQ_PORT") or os.getenv("RABBITMQ_PORT", "5672"))
RABBITMQ_USER = effective_config.get("RABBITMQ_USER") or os.getenv("RABBITMQ_USER", "guest")
RABBITMQ_PASS = effective_config.get("RABBITMQ_PASS") or os.getenv("RABBITMQ_PASS", "guest")
RABBITMQ_VHOST = effective_config.get("RABBITMQ_VHOST") or os.getenv("RABBITMQ_VHOST", "/")
QUEUE_NAME = effective_config.get("RABBITMQ_QUEUE") or os.getenv("RABBITMQ_QUEUE", "course_build_queue")
STATUS_QUEUE_NAME = effective_config.get("RABBITMQ_STATUS_QUEUE") or os.getenv("RABBITMQ_STATUS_QUEUE", "course_status_queue")

# Output and Processing Defaults
DEFAULT_OUTPUT_DIR = Path(
    effective_config.get("DEFAULT_OUTPUT_DIR") or os.getenv("DEFAULT_OUTPUT_DIR", "/var/storage/output_pdfs")
).resolve()

raw_delete_zip = effective_config.get("DELETE_ZIP_DEFAULT")
if raw_delete_zip is not None:
    DELETE_ZIP_DEFAULT = str(raw_delete_zip).lower() in ("true", "1", "yes")
else:
    DELETE_ZIP_DEFAULT = os.getenv("DELETE_ZIP_DEFAULT", "true").lower() in ("true", "1", "yes")

# Logging Level
LOG_LEVEL = str(effective_config.get("LOG_LEVEL") or os.getenv("LOG_LEVEL", "INFO")).upper()
