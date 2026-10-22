#!/usr/bin/env python3
"""
ABET Processor - Stateless RabbitMQ Consumer Microservice
Receives course submission jobs via RabbitMQ, compiles PDF binders headlessly in
an isolated temporary scratch directory, exports output PDFs/reports to destination,
and completely wipes all temporary processing data and input zip files.

Zero course data is stored inside the microservice.
"""

from __future__ import annotations

import contextlib
import json
import logging
import os
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import traceback
from pathlib import Path
import mimetypes
import pika
import requests

from config import (
    BASE_DIR,
    DEFAULT_OUTPUT_DIR,
    DELETE_ZIP_DEFAULT,
    LOG_LEVEL,
    QUEUE_NAME,
    RABBITMQ_HOST,
    RABBITMQ_PASS,
    RABBITMQ_PORT,
    RABBITMQ_USER,
    RABBITMQ_VHOST,
    STATUS_QUEUE_NAME,
)

# Configure logging
logging.basicConfig(
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    format="%(asctime)s [%(levelname)s] [StatelessWorker] %(message)s",
    handlers=[logging.StreamHandler(sys.stdout)],
)


class _TraceIdFilter(logging.Filter):
    """Appends ' trace_id=<hex>' to log lines written while a job span is active (the format is left untouched)."""

    def filter(self, record: logging.LogRecord) -> bool:
        if _trace_api is not None:
            try:
                ctx = _trace_api.get_current_span().get_span_context()
                if ctx.is_valid:
                    record.msg = f"{record.getMessage()} trace_id={ctx.trace_id:032x}"
                    record.args = ()
            except Exception:
                pass
        return True


for _handler in logging.getLogger().handlers:
    _handler.addFilter(_TraceIdFilter())

# Global connection reference for graceful shutdown
connection: pika.BlockingConnection | None = None
should_stop = False
_minio_client = None

# OpenTelemetry (only set up when OTEL_ENABLED=true; telemetry must never make a job fail)
_trace_api = None
_tracer = None
_propagate = None
_tracer_provider = None


def setup_tracing() -> None:
    """Configure the OTLP/HTTP exporter (APM Server). Does nothing unless OTEL_ENABLED=true."""
    global _trace_api, _tracer, _propagate, _tracer_provider
    if os.environ.get("OTEL_ENABLED", "false").strip().lower() not in ("true", "1", "yes"):
        return
    try:
        from opentelemetry import propagate, trace
        from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
        from opentelemetry.sdk.resources import Resource
        from opentelemetry.sdk.trace import TracerProvider
        from opentelemetry.sdk.trace.export import BatchSpanProcessor
        from opentelemetry.sdk.trace.sampling import ParentBased, TraceIdRatioBased

        attrs = {"service.name": os.environ.get("OTEL_SERVICE_NAME", "portfolio-service")}
        for item in os.environ.get("OTEL_RESOURCE_ATTRIBUTES", "").split(","):
            key, sep, value = item.strip().partition("=")
            if sep and key:
                attrs[key] = value
        endpoint = os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT", "http://apm-server:8200").rstrip("/")
        token = os.environ.get("APM_SECRET_TOKEN", "")
        exporter = OTLPSpanExporter(
            endpoint=endpoint + "/v1/traces",
            headers={"Authorization": f"Bearer {token}"} if token else None,
        )
        ratio = float(os.environ.get("OTEL_SAMPLE_RATIO", "1.0") or "1.0")
        provider = TracerProvider(resource=Resource.create(attrs), sampler=ParentBased(TraceIdRatioBased(ratio)))
        provider.add_span_processor(BatchSpanProcessor(exporter))
        _tracer_provider = provider
        _tracer = provider.get_tracer("portfolio-service")
        _propagate, _trace_api = propagate, trace
        logging.info(f"Tracing enabled (OTLP endpoint {endpoint})")
    except Exception as exc:
        _tracer = _trace_api = _propagate = None
        logging.warning(f"Tracing disabled, setup failed: {exc}")


def shutdown_tracing() -> None:
    """Flush pending spans (best effort)."""
    if _tracer_provider is not None:
        try:
            _tracer_provider.shutdown()
        except Exception:
            pass


def span(name: str, kind=None, context=None, **attributes):
    """Context manager for a span; a no-op when tracing is off. Exceptions inside are recorded and re-raised."""
    if _tracer is None:
        return contextlib.nullcontext()
    try:
        return _tracer.start_as_current_span(
            name,
            context=context,
            kind=kind or _trace_api.SpanKind.INTERNAL,
            attributes={k: v for k, v in attributes.items() if v is not None},
        )
    except Exception:
        return contextlib.nullcontext()


def trace_headers() -> dict | None:
    """W3C trace context of the current span as AMQP headers (None when tracing is off)."""
    if _propagate is None:
        return None
    try:
        headers: dict = {}
        _propagate.inject(headers)
        return headers or None
    except Exception:
        return None


def record_failure(exc: Exception) -> None:
    """Mark the current span as failed (the job's error is otherwise swallowed by on_message)."""
    if _trace_api is None:
        return
    try:
        current = _trace_api.get_current_span()
        current.record_exception(exc)
        current.set_status(_trace_api.Status(_trace_api.StatusCode.ERROR, str(exc)[:200]))
    except Exception:
        pass


def get_minio():
    """Return the shared MinIO client (created lazily; configured from MINIO_* env vars)."""
    global _minio_client
    if _minio_client is None:
        from minio import Minio

        _minio_client = Minio(
            os.environ.get("MINIO_ENDPOINT", "minio:9000"),
            access_key=os.environ.get("MINIO_ACCESS_KEY", ""),
            secret_key=os.environ.get("MINIO_SECRET_KEY", ""),
            secure=os.environ.get("MINIO_USE_SSL", "false").strip().lower() in ("true", "1", "yes"),
        )
    return _minio_client


COURSE_IDENTITY_CSV = "00-meta/instructor_course_identity_template.csv"


def copy_zip_without_course_cover(src: Path, dst: Path) -> int:
    """Copy the course zip to dst with the `cover_image` column of the course identity CSV emptied.

    The pipeline uses its default cover whenever a course sets `cover_image`, so the curriculum cover
    (config/assets/cover.pdf) only applies when that value is empty. Returns how many CSV files changed.
    """
    import csv
    import io
    import zipfile

    changed = 0
    with zipfile.ZipFile(src) as zin, zipfile.ZipFile(dst, "w", zipfile.ZIP_DEFLATED) as zout:
        for info in zin.infolist():
            data = zin.read(info)
            if info.filename.replace("\\", "/").endswith(COURSE_IDENTITY_CSV):
                text = data.decode("utf-8-sig")
                reader = csv.DictReader(io.StringIO(text, newline=""))
                rows = list(reader)
                fields = reader.fieldnames or []
                if "cover_image" in fields and any((r.get("cover_image") or "").strip() for r in rows):
                    for r in rows:
                        r["cover_image"] = ""
                    out = io.StringIO(newline="")
                    writer = csv.DictWriter(out, fieldnames=fields, lineterminator="\n")
                    writer.writeheader()
                    writer.writerows(rows)
                    data = out.getvalue().encode("utf-8")
                    changed += 1
            zout.writestr(info, data)
    return changed


def send_webhook_notification(url: str, payload: dict) -> None:
    """Send an HTTP POST status callback if webhooks are requested in job payload."""
    try:
        logging.info(f"Sending webhook status to: {url}")
        resp = requests.post(url, json=payload, timeout=10)
        logging.info(f"Webhook response status: {resp.status_code}")
    except Exception as exc:
        logging.warning(f"Failed to send status webhook to {url}: {exc}")


def publish_status_event(ch: pika.channel.Channel, status_event: dict) -> None:
    """Publish a status event to RabbitMQ status queue for downstream web app listeners."""
    try:
        ch.queue_declare(queue=STATUS_QUEUE_NAME, durable=True)
        ch.basic_publish(
            exchange="",
            routing_key=STATUS_QUEUE_NAME,
            body=json.dumps(status_event).encode("utf-8"),
            properties=pika.BasicProperties(delivery_mode=2, headers=trace_headers()),  # persistent
        )
        logging.info(f"Published status event to queue '{STATUS_QUEUE_NAME}'")
    except Exception as exc:
        logging.warning(f"Failed to publish status event to queue '{STATUS_QUEUE_NAME}': {exc}")


def process_job(payload: dict) -> dict:
    """
    Execute the processing workflow in a 100% ephemeral temporary workspace.
    Exports PDFs and reports to output_dir, then completely deletes the temp workspace and input zip.
    """
    job_id = payload.get("job_id", f"job-{int(time.time())}")
    program_name = payload.get("program_name", "ephemeral-program")
    use_s3 = payload.get("storage") == "s3"
    bucket = os.environ.get("MINIO_BUCKET", "abet")
    raw_zip_path = payload.get("zip_key") if use_s3 else payload.get("zip_path")
    output_dir_path = payload.get("output_dir", str(DEFAULT_OUTPUT_DIR))
    delete_zip = payload.get("delete_zip_on_completion", DELETE_ZIP_DEFAULT)

    if not raw_zip_path:
        raise ValueError(f"Job payload missing required parameter: '{'zip_key' if use_s3 else 'zip_path'}'")

    # Create an isolated temporary directory for processing
    temp_dir = tempfile.mkdtemp(prefix=f"abet_stateless_{job_id}_")
    temp_root = Path(temp_dir).resolve()

    try:
        if use_s3:
            # Inputs come from MinIO: zip -> <temp>/input/<name>.zip; outputs go to <temp>/export, then to MinIO
            zip_path = temp_root / "input" / Path(raw_zip_path).name
            output_dir = temp_root / "export"
            try:
                zip_path.parent.mkdir(parents=True, exist_ok=True)
                with span("minio download", key=raw_zip_path):
                    get_minio().fget_object(bucket, raw_zip_path, str(zip_path))
            except Exception as exc:
                raise RuntimeError(f"Storage error: cannot download '{raw_zip_path}': {exc}") from exc
        else:
            zip_path = Path(raw_zip_path).resolve()
            output_dir = Path(output_dir_path).resolve()
            if not zip_path.exists():
                raise FileNotFoundError(f"Source zip file does not exist: {zip_path}")

        logging.info(f"[{job_id}] Starting stateless job for program '{program_name}' with zip: {zip_path}")

        # 1. Scaffold temporary workspace inside temp_root
        script = BASE_DIR / "scripts" / "create_program_workspace.py"
        cmd_init = [
            sys.executable,
            str(script),
            program_name,
            "--programs-root",
            str(temp_root),
            "--force",
        ]
        proc_init = subprocess.run(cmd_init, cwd=BASE_DIR, capture_output=True, text=True)
        if proc_init.returncode != 0:
            raise RuntimeError(f"Failed to scaffold temporary workspace: {proc_init.stderr}")

        program_dir = temp_root / program_name

        # 1b. Optional per-curriculum cover: copy into the program workspace as config/assets/cover.pdf
        cover_key = payload.get("cover_key") if use_s3 else None
        if cover_key:
            cover_dst = program_dir / "config" / "assets" / "cover.pdf"
            cover_dst.parent.mkdir(parents=True, exist_ok=True)
            try:
                with span("minio download", key=cover_key):
                    get_minio().fget_object(bucket, cover_key, str(cover_dst))
                logging.info(f"[{job_id}] Using curriculum cover page: {cover_key}")
            except Exception as exc:
                raise RuntimeError(f"Storage error: cannot download cover '{cover_key}': {exc}") from exc

        cover_pdf_path = payload.get("cover_pdf_path")
        if cover_pdf_path:
            cover_src = Path(cover_pdf_path)
            if cover_src.is_file():
                cover_dst = program_dir / "config" / "assets" / "cover.pdf"
                cover_dst.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(cover_src, cover_dst)
                logging.info(f"[{job_id}] Using curriculum cover page: {cover_src}")
            else:
                logging.warning(f"[{job_id}] cover_pdf_path not found, using default cover: {cover_src}")

        # 2. Stage zip file in temporary workspace runtime/incoming
        incoming_dir = program_dir / "runtime" / "incoming"
        incoming_dir.mkdir(parents=True, exist_ok=True)
        staged_zip = incoming_dir / zip_path.name
        if (program_dir / "config" / "assets" / "cover.pdf").is_file() and (cover_key or cover_pdf_path):
            # The curriculum cover wins over a course's own `cover_image` (D27)
            if copy_zip_without_course_cover(zip_path, staged_zip):
                logging.info(f"[{job_id}] Course cover_image ignored: the curriculum cover page is used")
        else:
            shutil.copy2(zip_path, staged_zip)

        # 3. Invoke runtime pipeline build-course command inside temp workspace
        pipeline_script = BASE_DIR / "scripts" / "runtime_pipeline.py"
        cmd = [sys.executable, str(pipeline_script), "build-course", str(staged_zip)]

        logging.info(f"[{job_id}] Compiling PDF in ephemeral workspace...")
        start_time = time.time()
        with span("portfolio build", program_name=program_name):
            proc = subprocess.run(cmd, cwd=program_dir, capture_output=True, text=True)
        elapsed = round(time.time() - start_time, 2)

        if proc.returncode != 0:
            logging.error(f"[{job_id}] Pipeline build failed with exit code {proc.returncode}")
            logging.error(f"[{job_id}] Stdout:\n{proc.stdout}")
            logging.error(f"[{job_id}] Stderr:\n{proc.stderr}")
            raise RuntimeError(f"Pipeline processing failed for job {job_id}: {proc.stderr or proc.stdout}")

        logging.info(f"[{job_id}] Pipeline completed in {elapsed}s")

        # 4. Export generated PDFs and reports to target output directory
        output_dir.mkdir(parents=True, exist_ok=True)

        source_pdfs_dir = program_dir / "runtime" / "output" / "course-pdfs"
        source_reports_dir = program_dir / "runtime" / "output" / "reports"

        generated_pdfs = list(source_pdfs_dir.glob("*.pdf")) if source_pdfs_dir.exists() else []
        generated_reports = list(source_reports_dir.glob("*.*")) if source_reports_dir.exists() else []

        if not generated_pdfs:
            logging.warning(f"[{job_id}] No PDF files found in {source_pdfs_dir}")

        exported_pdfs = []
        for pdf in generated_pdfs:
            target_pdf = output_dir / pdf.name
            shutil.copy2(pdf, target_pdf)
            exported_pdfs.append(str(target_pdf))
            logging.info(f"[{job_id}] Exported binder PDF: {target_pdf}")

        exported_reports = []
        for rpt in generated_reports:
            target_rpt = output_dir / rpt.name
            shutil.copy2(rpt, target_rpt)
            exported_reports.append(str(target_rpt))
            logging.info(f"[{job_id}] Exported report file: {target_rpt}")

        exported_pdf_keys = []
        exported_report_keys = []
        if use_s3:
            prefix = payload.get("output_prefix") or f"outputs/{job_id}/"
            if not prefix.endswith("/"):
                prefix += "/"
            try:
                for local_file, keys in [(f, exported_pdf_keys) for f in generated_pdfs] + [
                    (f, exported_report_keys) for f in generated_reports
                ]:
                    key = prefix + local_file.name
                    content_type = "application/pdf" if local_file.suffix.lower() == ".pdf" else (
                        mimetypes.guess_type(local_file.name)[0] or "application/octet-stream"
                    )
                    with span("minio upload", key=key):
                        get_minio().fput_object(bucket, key, str(local_file), content_type=content_type)
                    keys.append(key)
                    logging.info(f"[{job_id}] Uploaded output object: {key}")
            except Exception as exc:
                raise RuntimeError(f"Storage error: cannot upload outputs to '{prefix}': {exc}") from exc

        # 5. Clean up input zip file if requested
        if delete_zip and use_s3:
            try:
                get_minio().remove_object(bucket, raw_zip_path)
                logging.info(f"[{job_id}] Deleted source zip object: {raw_zip_path}")
            except Exception as exc:
                logging.warning(f"[{job_id}] Non-fatal error deleting input zip object: {exc}")
        elif delete_zip:
            try:
                if zip_path.exists():
                    os.remove(zip_path)
                    logging.info(f"[{job_id}] Deleted source zip file: {zip_path}")
            except Exception as exc:
                logging.warning(f"[{job_id}] Non-fatal error deleting input zip: {exc}")

        return {
            "status": "success",
            "job_id": job_id,
            "program_name": program_name,
            "elapsed_seconds": elapsed,
            "exported_pdfs": exported_pdfs,
            "exported_reports": exported_reports,
            "exported_pdf_keys": exported_pdf_keys,
            "exported_report_keys": exported_report_keys,
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        }

    finally:
        # GUARANTEED CLEANUP: Completely wipe the temporary processing workspace
        if os.path.exists(temp_root):
            shutil.rmtree(temp_root, ignore_errors=True)
            logging.info(f"[{job_id}] Wiped ephemeral processing directory: {temp_root}")


def on_message(ch: pika.channel.Channel, method, properties, body: bytes) -> None:
    """Callback triggered when a message arrives from RabbitMQ queue; runs the job inside a CONSUMER span."""
    ctx = job_id = program_name = None
    if _tracer is not None:
        try:
            ctx = _propagate.extract(dict(properties.headers or {}))
            job = json.loads(body.decode("utf-8"))
            job_id, program_name = job.get("job_id"), job.get("program_name")
        except Exception:
            pass
    kind = _trace_api.SpanKind.CONSUMER if _trace_api is not None else None
    with span(f"{QUEUE_NAME} process", kind=kind, context=ctx, **{"abet.job_id": job_id, "abet.program_name": program_name}):
        handle_message(ch, method, properties, body)


def handle_message(ch: pika.channel.Channel, method, properties, body: bytes) -> None:
    """Process one message (unchanged job handling)."""
    raw_body = body.decode("utf-8")
    logging.info(f"Received message payload: {raw_body}")

    callback_url = None
    job_id = "unknown"

    try:
        payload = json.loads(raw_body)
        job_id = payload.get("job_id", job_id)
        callback_url = payload.get("callback_url")

        # Execute stateless processing
        result = process_job(payload)

        # Notify downstream system via webhook if provided
        if callback_url:
            send_webhook_notification(callback_url, result)

        # Publish event to status queue
        publish_status_event(ch, result)

        # Acknowledge message to RabbitMQ
        ch.basic_ack(delivery_tag=method.delivery_tag)
        logging.info(f"[{job_id}] Job processed, exported, and temporary workspace cleaned successfully.")

    except Exception as exc:
        err_msg = str(exc)
        tb = traceback.format_exc()
        logging.error(f"[{job_id}] Error processing message: {err_msg}\n{tb}")
        record_failure(exc)

        failure_event = {
            "status": "failed",
            "job_id": job_id,
            "error": err_msg,
            "traceback": tb,
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        }

        if callback_url:
            send_webhook_notification(callback_url, failure_event)

        publish_status_event(ch, failure_event)

        # Reject message without requeueing to avoid infinite retry loop
        ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)


def handle_signal(sig, frame):
    """Handle graceful shutdown signals (SIGTERM, SIGINT)."""
    global should_stop, connection
    logging.info("Shutdown signal received. Closing worker gracefully...")
    should_stop = True
    if connection and connection.is_open:
        connection.close()
    shutdown_tracing()
    sys.exit(0)


def main() -> None:
    global connection
    setup_tracing()
    signal.signal(signal.SIGINT, handle_signal)
    signal.signal(signal.SIGTERM, handle_signal)

    credentials = pika.PlainCredentials(RABBITMQ_USER, RABBITMQ_PASS)
    parameters = pika.ConnectionParameters(
        host=RABBITMQ_HOST,
        port=RABBITMQ_PORT,
        virtual_host=RABBITMQ_VHOST,
        credentials=credentials,
        heartbeat=600,
        blocked_connection_timeout=300,
    )

    logging.info(f"Connecting to RabbitMQ at {RABBITMQ_HOST}:{RABBITMQ_PORT} (vhost={RABBITMQ_VHOST})...")

    retry_delay = 5
    while not should_stop:
        try:
            connection = pika.BlockingConnection(parameters)
            channel = connection.channel()

            # Declare queues
            channel.queue_declare(queue=QUEUE_NAME, durable=True)
            channel.queue_declare(queue=STATUS_QUEUE_NAME, durable=True)

            # Fetch 1 message at a time
            channel.basic_qos(prefetch_count=1)
            channel.basic_consume(queue=QUEUE_NAME, on_message_callback=on_message)

            logging.info(f"Stateless Worker connected! Ready and waiting for jobs on queue '{QUEUE_NAME}'...")
            channel.start_consuming()

        except pika.exceptions.AMQPConnectionError as err:
            logging.warning(f"RabbitMQ connection lost ({err}). Retrying in {retry_delay} seconds...")
            time.sleep(retry_delay)
        except Exception as err:
            logging.error(f"Unexpected connection error ({err}). Retrying in {retry_delay} seconds...")
            time.sleep(retry_delay)


if __name__ == "__main__":
    main()
