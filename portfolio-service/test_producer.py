#!/usr/bin/env python3
"""
Test Producer for ABET Processor Microservice
Use this script to push a sample job message to RabbitMQ.
"""

import json
import os
import sys
from pathlib import Path
import pika

RABBITMQ_HOST = os.getenv("RABBITMQ_HOST", "localhost")
RABBITMQ_PORT = int(os.getenv("RABBITMQ_PORT", "5672"))
RABBITMQ_USER = os.getenv("RABBITMQ_USER", "guest")
RABBITMQ_PASS = os.getenv("RABBITMQ_PASS", "guest")
QUEUE_NAME = os.getenv("RABBITMQ_QUEUE", "course_build_queue")


def publish_test_job(zip_path: str, program_name: str = "aero-ay2025", output_dir: str = None):
    credentials = pika.PlainCredentials(RABBITMQ_USER, RABBITMQ_PASS)
    connection = pika.BlockingConnection(
        pika.ConnectionParameters(host=RABBITMQ_HOST, port=RABBITMQ_PORT, credentials=credentials)
    )
    channel = connection.channel()
    channel.queue_declare(queue=QUEUE_NAME, durable=True)

    payload = {
        "job_id": f"test-job-{int(pika.compat.time.time())}",
        "program_name": program_name,
        "zip_path": zip_path,
        "delete_zip_on_completion": False,  # Keep original zip during testing
    }
    if output_dir:
        payload["output_dir"] = output_dir

    message_body = json.dumps(payload, indent=2)
    channel.basic_publish(
        exchange="",
        routing_key=QUEUE_NAME,
        body=message_body,
        properties=pika.BasicProperties(delivery_mode=2),
    )

    print(f"[+] Published test job to RabbitMQ queue '{QUEUE_NAME}':")
    print(message_body)
    connection.close()


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python test_producer.py <path_to_course_zip> [program_name] [output_dir]")
        print("Example: python test_producer.py /path/to/course.zip aero-ay2025")
        sys.exit(1)

    zip_file = sys.argv[1]
    prog = sys.argv[2] if len(sys.argv) > 2 else "aero-ay2025"
    out = sys.argv[3] if len(sys.argv) > 3 else None

    publish_test_job(zip_file, prog, out)
