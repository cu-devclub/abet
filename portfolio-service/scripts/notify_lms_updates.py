#!/usr/bin/env python3
"""Notify staff by email when LMS Watch detects newly available zip files."""

from __future__ import annotations

import argparse
import csv
import json
import os
import smtplib
import ssl
from dataclasses import dataclass
from datetime import datetime
from email.message import EmailMessage
from pathlib import Path

SHARED_ROOT = Path(__file__).resolve().parents[1]


@dataclass
class ChangedCourse:
    course_code: str
    course_name: str
    latest_modified: str
    latest_item: str
    previous_modified: str
    previous_item: str
    check_url: str
    note: str

    @property
    def key(self) -> str:
        return "|".join([self.course_code, self.latest_item, self.latest_modified])

    @property
    def label(self) -> str:
        return f"{self.course_code} {self.course_name}".strip()


@dataclass
class LmsReportRow:
    course_code: str
    course_name: str
    status: str
    latest_modified: str
    latest_item: str
    previous_modified: str
    previous_item: str
    check_url: str
    note: str

    @property
    def label(self) -> str:
        return f"{self.course_code} {self.course_name}".strip()


@dataclass
class PendingUpload:
    course_code: str
    course_name: str
    latest_item: str
    latest_modified: str
    check_url: str
    first_requested_at: str

    @property
    def label(self) -> str:
        return f"{self.course_code} {self.course_name}".strip()


def detect_workspace_root() -> Path:
    env_root = os.environ.get("COURSEDISPLAY_WORKSPACE_ROOT")
    if env_root:
        return Path(env_root).resolve()
    cwd = Path.cwd()
    if (cwd / "config" / "staff" / "course_registry.csv").is_file():
        return cwd.resolve()
    return SHARED_ROOT


def load_report_rows(report_csv: Path) -> list[LmsReportRow]:
    if not report_csv.is_file():
        raise SystemExit(f"LMS Watch staff CSV not found: {report_csv}")
    rows: list[LmsReportRow] = []
    with report_csv.open(newline="", encoding="utf-8-sig") as handle:
        reader = csv.DictReader(handle)
        for row in reader:
            rows.append(
                LmsReportRow(
                    course_code=(row.get("course_code") or "").strip(),
                    course_name=(row.get("course_name") or "").strip(),
                    status=(row.get("status") or "").strip().lower(),
                    latest_modified=(row.get("latest_modified") or "").strip(),
                    latest_item=(row.get("latest_item") or "").strip(),
                    previous_modified=(row.get("previous_modified") or "").strip(),
                    previous_item=(row.get("previous_item") or "").strip(),
                    check_url=(row.get("check_url") or "").strip(),
                    note=(row.get("note") or "").strip(),
                )
            )
    return rows


def load_changed_courses(report_csv: Path) -> list[ChangedCourse]:
    changed: list[ChangedCourse] = []
    for row in load_report_rows(report_csv):
        if row.status != "changed":
            continue
        changed.append(
            ChangedCourse(
                course_code=row.course_code,
                course_name=row.course_name,
                latest_modified=row.latest_modified,
                latest_item=row.latest_item,
                previous_modified=row.previous_modified,
                previous_item=row.previous_item,
                check_url=row.check_url,
                note=row.note,
            )
        )
    return changed


def default_pending_uploads_path(workspace: Path) -> Path:
    return workspace / "runtime" / "lms-watch" / "pending_drive_uploads_latest.json"


def load_pending_uploads(path: Path) -> list[PendingUpload]:
    if not path.is_file():
        return []
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return []
    raw_pending = data.get("pending", []) if isinstance(data, dict) else []
    if not isinstance(raw_pending, list):
        return []
    pending: list[PendingUpload] = []
    for item in raw_pending:
        if not isinstance(item, dict):
            continue
        pending.append(
            PendingUpload(
                course_code=str(item.get("course_code", "")).strip(),
                course_name=str(item.get("course_name", "")).strip(),
                latest_item=str(item.get("latest_item", "")).strip(),
                latest_modified=str(item.get("latest_modified", "")).strip(),
                check_url=str(item.get("check_url", "")).strip(),
                first_requested_at=str(item.get("first_requested_at", "")).strip(),
            )
        )
    return pending


def pending_uploads_from_changed_rows(rows: list[LmsReportRow]) -> list[PendingUpload]:
    pending: list[PendingUpload] = []
    for row in rows:
        if row.status != "changed":
            continue
        pending.append(
            PendingUpload(
                course_code=row.course_code,
                course_name=row.course_name,
                latest_item=row.latest_item,
                latest_modified=row.latest_modified,
                check_url=row.check_url,
                first_requested_at="",
            )
        )
    return pending


def load_history(path: Path) -> dict[str, object]:
    if not path.is_file():
        return {"sent": {}}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return {"sent": {}}
    return data if isinstance(data, dict) else {"sent": {}}


def save_history(path: Path, history: dict[str, object]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(history, indent=2) + "\n", encoding="utf-8")


def google_drive_incoming_label(workspace: Path) -> str:
    cfg = workspace / "config" / "staff" / "drive_intake.json"
    if not cfg.is_file():
        return "Google Drive CourseDisplay Intake > incoming/"
    try:
        data = json.loads(cfg.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return "Google Drive CourseDisplay Intake > incoming/"
    folders = data.get("drive_folders", {})
    incoming = folders.get("incoming", "incoming") if isinstance(folders, dict) else "incoming"
    api = data.get("google_drive_api", {})
    folder_id = api.get("root_folder_id", "") if isinstance(api, dict) else ""
    if folder_id:
        return f"Google Drive CourseDisplay Intake folder ({folder_id}) > {incoming}/"
    root = data.get("drive_root", "")
    return f"{root}/{incoming}/" if root else f"Google Drive CourseDisplay Intake > {incoming}/"


def drive_followup_text() -> str:
    return (
        os.environ.get("COURSEDISPLAY_NOTIFY_DRIVE_FOLLOWUP_TEXT", "").strip()
        or "After the zip file is uploaded to Google Drive, the course coordinator will be notified for local processing."
    )


def build_email(workspace: Path, changed: list[ChangedCourse], recipient_text: str) -> tuple[str, str]:
    program_name = workspace.name
    subject = f"CourseDisplay LMS update detected: {program_name}"
    incoming = google_drive_incoming_label(workspace)
    lines = [
        "Dear staff,",
        "",
        f"CourseDisplay detected updated LMS file(s) for {program_name}.",
        "",
        "Please download the updated zip file(s) from MyCourseVille and upload them to:",
        "",
        f"{incoming}",
        "",
        "Courses requiring download:",
        "",
    ]
    for course in changed:
        lines.extend(
            [
                f"- {course.label}",
                f"  Latest LMS file: {course.latest_item or '-'}",
                f"  Latest modified: {course.latest_modified or '-'}",
                f"  Previous file: {course.previous_item or '-'}",
                f"  LMS page: {course.check_url or '-'}",
                "",
            ]
        )
    lines.extend(
        [
            drive_followup_text(),
            "",
            "Best,",
            "CourseDisplay",
            "",
            f"Notification recipients: {recipient_text}",
        ]
    )
    return subject, "\n".join(lines)


def build_daily_report_email(
    workspace: Path,
    rows: list[LmsReportRow],
    pending_uploads: list[PendingUpload],
    recipient_text: str,
    report_csv: Path,
) -> tuple[str, str]:
    program_name = workspace.name
    generated = datetime.now().isoformat(timespec="seconds")
    incoming = google_drive_incoming_label(workspace)
    counts: dict[str, int] = {}
    for row in rows:
        counts[row.status or "unknown"] = counts.get(row.status or "unknown", 0) + 1
    changed = [row for row in rows if row.status == "changed"]
    errors = [row for row in rows if row.status == "error"]
    unchanged = [row for row in rows if row.status == "unchanged"]
    baseline = [row for row in rows if row.status == "baseline-set"]
    subject_status = "uploads pending" if pending_uploads else ("updates detected" if changed else "no LMS zip changes")
    subject = f"CourseDisplay LMS daily report: {program_name} - {subject_status}"

    lines = [
        "Dear staff,",
        "",
        f"CourseDisplay completed the daily LMS Watch check for {program_name}.",
        f"Generated: {generated}",
        "",
        "Summary:",
        f"- Updated LMS zip files detected: {len(changed)}",
        f"- Unchanged LMS zip files: {len(unchanged)}",
        f"- New baselines recorded: {len(baseline)}",
        f"- Check errors: {len(errors)}",
        f"- Pending LMS zip uploads missing from Drive: {len(pending_uploads)}",
        f"- Total monitored rows in report: {len(rows)}",
        "",
    ]
    if pending_uploads:
        lines.extend(
            [
                "Action required:",
                "",
                "Please download the following LMS zip file(s) from MyCourseVille and upload them to:",
                "",
                f"{incoming}",
                "",
                "Items stay on this list until a matching zip file appears in Google Drive incoming.",
                "",
                "Pending uploads:",
                "",
            ]
        )
        for item in pending_uploads:
            lines.extend(
                [
                    f"- {item.label}",
                    f"  Required zip file: {item.latest_item or '-'}",
                    f"  LMS modified: {item.latest_modified or '-'}",
                    f"  LMS page: {item.check_url or '-'}",
                    "",
                ]
            )
    else:
        lines.extend(
            [
                "No action is required today. No newer LMS zip files were detected.",
                "",
                "This daily email confirms that the LMS Watch process is running.",
                "",
            ]
        )

    if errors:
        lines.extend(["Rows needing coordinator review:", ""])
        for row in errors[:12]:
            lines.extend(
                [
                    f"- {row.label}",
                    f"  Note: {row.note or '-'}",
                    f"  LMS page: {row.check_url or '-'}",
                    "",
                ]
            )
        if len(errors) > 12:
            lines.append(f"- {len(errors) - 12} additional error row(s) omitted from this email.")
            lines.append("")

    if pending_uploads:
        lines.extend(
            [
                drive_followup_text(),
                "",
            ]
        )
    lines.extend(
        [
            f"Local report CSV: {report_csv}",
            "",
            "Best,",
            "CourseDisplay",
            "",
            f"Notification recipients: {recipient_text}",
        ]
    )
    return subject, "\n".join(lines)


def split_recipients(raw: str) -> list[str]:
    return [item.strip() for item in raw.replace(";", ",").split(",") if item.strip()]


def send_email(subject: str, body: str, recipients: list[str]) -> None:
    sender = os.environ.get("COURSEDISPLAY_NOTIFY_EMAIL_FROM", "").strip()
    password = os.environ.get("COURSEDISPLAY_GMAIL_APP_PASSWORD", "").strip()
    if not sender:
        raise SystemExit("Missing COURSEDISPLAY_NOTIFY_EMAIL_FROM")
    if not password:
        raise SystemExit("Missing COURSEDISPLAY_GMAIL_APP_PASSWORD")
    message = EmailMessage()
    message["From"] = sender
    message["To"] = ", ".join(recipients)
    message["Subject"] = subject
    message.set_content(body)
    context = ssl.create_default_context()
    with smtplib.SMTP_SSL("smtp.gmail.com", 465, context=context) as server:
        server.login(sender, password)
        server.send_message(message)


def parse_args() -> argparse.Namespace:
    workspace = detect_workspace_root()
    parser = argparse.ArgumentParser(description="Email staff when LMS Watch detects changed files.")
    parser.add_argument("--workspace", default=str(workspace), help="Program workspace root.")
    parser.add_argument("--report-csv", help="Path to lms-watch-staff-latest.csv.")
    parser.add_argument("--history", help="Notification history JSON path.")
    parser.add_argument("--to", help="Comma-separated recipients. Defaults to COURSEDISPLAY_NOTIFY_EMAIL_TO.")
    parser.add_argument("--dry-run", action="store_true", help="Print the email instead of sending it.")
    parser.add_argument("--force", action="store_true", help="Notify even if the same course/file was already sent.")
    parser.add_argument("--daily-report", action="store_true", help="Send a daily status report even when no LMS files changed.")
    parser.add_argument("--pending-uploads", help="Path to pending_drive_uploads_latest.json for repeated Drive upload action items.")
    parser.add_argument("--test-email", action="store_true", help="Send a harmless test email without reading LMS changes.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    workspace = Path(args.workspace).expanduser().resolve()
    report_csv = (
        Path(args.report_csv).expanduser().resolve()
        if args.report_csv
        else workspace / "runtime" / "output" / "reports" / "lms-watch" / "lms-watch-staff-latest.csv"
    )
    history_path = (
        Path(args.history).expanduser().resolve()
        if args.history
        else workspace / "runtime" / "lms-watch" / "notification_history.json"
    )
    recipients = split_recipients(args.to or os.environ.get("COURSEDISPLAY_NOTIFY_EMAIL_TO", ""))
    if not recipients and not args.dry_run:
        raise SystemExit("Missing recipients. Set COURSEDISPLAY_NOTIFY_EMAIL_TO or pass --to.")

    if args.test_email:
        subject = f"CourseDisplay email test: {workspace.name}"
        body = "\n".join(
            [
                "This is a CourseDisplay notification test.",
                "",
                f"Program workspace: {workspace}",
                f"Generated: {datetime.now().isoformat(timespec='seconds')}",
                "",
                "If you received this email, Gmail notification delivery is working.",
            ]
        )
        if args.dry_run:
            print(f"Subject: {subject}")
            print()
            print(body)
        else:
            send_email(subject, body, recipients)
            print(f"Sent test email to {', '.join(recipients)}.")
        return 0

    if args.daily_report:
        rows = load_report_rows(report_csv)
        pending_path = (
            Path(args.pending_uploads).expanduser().resolve()
            if args.pending_uploads
            else default_pending_uploads_path(workspace)
        )
        pending_uploads = load_pending_uploads(pending_path) if pending_path.is_file() else pending_uploads_from_changed_rows(rows)
        subject, body = build_daily_report_email(
            workspace,
            rows,
            pending_uploads,
            ", ".join(recipients) if recipients else "(dry run)",
            report_csv,
        )
        if args.dry_run:
            print(f"Subject: {subject}")
            print()
            print(body)
        else:
            send_email(subject, body, recipients)
            print(f"Sent LMS daily report for {len(rows)} monitored row(s).")
        return 0

    changed = load_changed_courses(report_csv)
    history = load_history(history_path)
    sent = history.setdefault("sent", {})
    if not isinstance(sent, dict):
        sent = {}
        history["sent"] = sent
    to_notify = [course for course in changed if args.force or course.key not in sent]
    if not to_notify:
        print("No new LMS changes to notify.")
        return 0

    subject, body = build_email(workspace, to_notify, ", ".join(recipients) if recipients else "(dry run)")
    if args.dry_run:
        print(f"Subject: {subject}")
        print()
        print(body)
    else:
        send_email(subject, body, recipients)
        now = datetime.now().isoformat(timespec="seconds")
        for course in to_notify:
            sent[course.key] = {
                "sent_at": now,
                "course_code": course.course_code,
                "course_name": course.course_name,
                "latest_item": course.latest_item,
                "latest_modified": course.latest_modified,
                "recipients": recipients,
            }
        save_history(history_path, history)
        print(f"Sent LMS update notification for {len(to_notify)} course(s).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
