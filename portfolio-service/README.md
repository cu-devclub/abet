# ABET Processor - Headless Microservice Worker

Automated, headless RabbitMQ microservice worker that processes instructor course submission `.zip` files, compiles ABET-compliant course display binder PDFs via LaTeX, exports generated PDFs to a designated directory, and deletes processed `.zip` files automatically with **zero human intervention**.

---

## 📁 Directory Structure

```text
ABET_Processor/
├── Dockerfile              # Docker container setup (Python + TeX Live + Poppler)
├── config.py               # Microservice configuration & environment variables
├── worker.py               # Main RabbitMQ consumer loop
├── test_producer.py        # Helper CLI script to publish test messages
├── requirements.txt        # Python dependencies for microservice
├── .env.example            # Environment variables template
└── README.md               # Documentation
```

---

## ⚡ Quick Start with Docker Compose (Recommended)

To launch both **RabbitMQ** and the **ABET Microservice Worker** from the root project directory:

```bash
docker compose up -d --build
```

- **RabbitMQ Broker**: Port `5672`
- **RabbitMQ Management Dashboard**: `http://localhost:15672` (Credentials: `guest` / `guest`)
- **Shared Storage Folders**:
  - Uploads: `./storage/uploads`
  - Completed PDFs: `./storage/output_pdfs`

To stop the containers:
```bash
docker compose down
```

---

## 📨 RabbitMQ Message Payload Format

Your web application backend (Node.js, Go, Python, Java, etc.) publishes a JSON message to queue **`course_build_queue`**:

```json
{
  "job_id": "job-20260812-001",
  "program_name": "aero-ay2025",
  "zip_path": "/var/storage/uploads/2145211-Introduction-to-Aerospace-Engineering.zip",
  "output_dir": "/var/storage/output_pdfs",
  "delete_zip_on_completion": true,
  "callback_url": "http://your-backend-api.internal/api/v1/jobs/callback"
}
```

### Payload v2 (MinIO / S3 storage)

When `"storage": "s3"` the worker reads its input from and writes its output to MinIO (bucket `MINIO_BUCKET`); no filesystem paths are involved:

```json
{
  "job_id": "job-20260812-001",
  "program_name": "aero-ay2025",
  "storage": "s3",
  "zip_key": "uploads/<submission_id>.zip",
  "cover_key": "covers/<curriculum_id>.pdf",
  "output_prefix": "outputs/<submission_id>/",
  "delete_zip_on_completion": true
}
```

| Field | Type | Description |
| :--- | :--- | :--- |
| `storage` | `string` | `"s3"` selects v2. Anything else (or missing) = the path-based payload below, unchanged. |
| `zip_key` | `string` | Object key of the instructor `.zip` (downloaded to a temp dir). |
| `cover_key` | `string` *(optional)* | Object key of a cover PDF (same effect as `cover_pdf_path`). |
| `output_prefix` | `string` *(optional)* | Key prefix the generated PDFs and reports are uploaded to (default `outputs/<job_id>/`). |
| `delete_zip_on_completion` | `boolean` *(optional)* | Removes the `zip_key` object after a successful upload of the outputs. |

The success status event additionally carries `exported_pdf_keys` and `exported_report_keys` (object keys). `exported_pdfs` / `exported_reports` then point at the wiped temp workspace and must be ignored. Download or upload failures produce a normal `failed` event whose `error` starts with `Storage error:`.

### Key Parameters (path-based payload):

| Field | Type | Description |
| :--- | :--- | :--- |
| `job_id` | `string` | Unique identifier for tracking job status. |
| `program_name` | `string` | Target program workspace (e.g. `aero-ay2025`). Will be created automatically if missing. |
| `zip_path` | `string` | Absolute path to the uploaded instructor `.zip` file on disk. |
| `output_dir` | `string` *(optional)* | Custom directory where the compiled `.pdf` will be copied. Defaults to `/var/storage/output_pdfs`. |
| `delete_zip_on_completion` | `boolean` *(optional)* | If `true`, deletes the `.zip` file after processing. Default is `true`. |
| `callback_url` | `string` *(optional)* | HTTP URL to receive a POST webhook event when processing succeeds or fails. |
| `cover_pdf_path` | `string` *(optional)* | Absolute path to a PDF used as the course cover page instead of the default `assets/default/cover.pdf`. Copied into the temporary program workspace as `config/assets/cover.pdf`; if the file is missing a warning is logged and the default cover is used. When a curriculum cover is given (this field or `cover_key`) it always wins: the worker empties `cover_image` in the course identity CSV of its temporary copy of the upload. |

---

## 🧪 Testing the Microservice

You can test the worker using the included `test_producer.py` script:

```bash
python ABET_Processor/test_producer.py /path/to/uploaded/course.zip aero-ay2025
```

### What Happens Behind the Scenes:
1. Message is published to RabbitMQ (`course_build_queue`).
2. `worker.py` consumes the message.
3. Initializes the workspace if it doesn't exist.
4. Runs `scripts/runtime_pipeline.py build-course` headlessly.
5. Copies output `.pdf` binder to destination folder.
6. Deletes `.zip` file upon completion.
7. Publishes a completion event to `course_status_queue` (and optional webhook callback).

---

## ⚙️ Configuration Loading Hierarchy (`config.py`)

The microservice dynamically resolves its configuration using a 3-tier fallback strategy:

1. **Primary**: **MariaDB Database** (`app_settings` table). Queries database settings at startup if reachable.
2. **Fallback 1**: **Local `settings.json`** (`ABET_Processor/settings.json`). Loaded if MariaDB is offline or unreachable.
3. **Fallback 2**: **Environment Variables / Defaults**.

### MariaDB Table Schema Options:
You can store settings in MariaDB either as key-value rows:
```sql
CREATE TABLE app_settings (
  setting_key VARCHAR(255) PRIMARY KEY,
  setting_value VARCHAR(255)
);

INSERT INTO app_settings VALUES ('RABBITMQ_HOST', '10.0.0.5'), ('LOG_LEVEL', 'DEBUG');
```
...or as a single-row column table (`SELECT * FROM app_settings LIMIT 1`).

### Config Environment Variables (`.env`):
| Variable | Default | Description |
| :--- | :--- | :--- |
| `MARIADB_HOST` | `abet-mariadb` | MariaDB database host |
| `MARIADB_PORT` | `3306` | MariaDB database port |
| `MARIADB_USER` | `root` | MariaDB database username |
| `MARIADB_PASS` | `password` | MariaDB database password |
| `MARIADB_DB` | `abet_db` | MariaDB database name |
| `MARIADB_TABLE` | `app_settings` | MariaDB settings table name |
| `RABBITMQ_HOST` | `abet-rabbitmq` | RabbitMQ server hostname |
| `RABBITMQ_PORT` | `5672` | RabbitMQ server port |
| `RABBITMQ_USER` | `guest` | RabbitMQ username |
| `RABBITMQ_PASS` | `guest` | RabbitMQ password |
| `RABBITMQ_QUEUE` | `course_build_queue` | Target queue for incoming jobs |
| `RABBITMQ_STATUS_QUEUE` | `course_status_queue` | Queue for job completion/failure events |
| `DEFAULT_OUTPUT_DIR` | `/var/storage/output_pdfs` | Default destination path for generated PDFs |
| `DELETE_ZIP_DEFAULT` | `true` | Auto-delete `.zip` files after completion |
| `MINIO_ENDPOINT` | `minio:9000` | MinIO S3 endpoint (read from the environment only, payload v2) |
| `MINIO_ACCESS_KEY` | *(empty)* | MinIO service account access key |
| `MINIO_SECRET_KEY` | *(empty)* | MinIO service account secret key |
| `MINIO_BUCKET` | `abet` | Bucket holding uploads/covers/outputs |
| `MINIO_USE_SSL` | `false` | Use HTTPS for the MinIO endpoint |
