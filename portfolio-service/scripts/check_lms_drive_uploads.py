#!/usr/bin/env python3
"""Track LMS zip upload requests until matching files appear in Drive."""

from __future__ import annotations

import argparse
import csv
import json
import re
from dataclasses import dataclass
from datetime import datetime, time, timezone
from pathlib import Path
from typing import Any

import check_drive_incoming
import drive_intake


@dataclass
class LmsChange:
    course_code: str
    course_name: str
    latest_item: str
    latest_modified: str
    check_url: str
    report_path: Path
    report_time: datetime

    @property
    def key(self) -> str:
        return "|".join([self.course_code, self.latest_item, self.latest_modified])

    @property
    def label(self) -> str:
        return " ".join(part for part in [self.course_code, self.course_name] if part).strip()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Cross-check LMS upload requests against Drive incoming zip files.")
    parser.add_argument("--config", help="Path to drive_intake.json. Defaults to program config when --program-workspace is supplied.")
    parser.add_argument("--program-workspace", help="Program workspace, e.g. programs/ice-ay2025")
    parser.add_argument("--report-csv", help="Specific LMS staff report CSV to include.")
    parser.add_argument("--state-file", help="Pending upload state JSON.")
    parser.add_argument("--output-json", help="Latest summary JSON for notification/reporting.")
    parser.add_argument("--drive-json", help="Use a check_drive_incoming.py --json payload instead of querying Drive.")
    parser.add_argument("--max-items", type=int, default=12, help="Maximum pending item names to print.")
    parser.add_argument("--json", action="store_true", help="Print machine-readable JSON.")
    return parser.parse_args()


def utc_now() -> datetime:
    return datetime.now(timezone.utc)


def iso(dt: datetime) -> str:
    return dt.astimezone(timezone.utc).isoformat(timespec="seconds")


def parse_iso(raw: str) -> datetime | None:
    if not raw:
        return None
    try:
        return datetime.fromisoformat(raw.replace("Z", "+00:00")).astimezone(timezone.utc)
    except ValueError:
        return None


def report_time(path: Path) -> datetime:
    return datetime.fromtimestamp(path.stat().st_mtime, tz=timezone.utc)


def parse_lms_modified_date(raw: str) -> datetime | None:
    text = " ".join((raw or "").split())
    for fmt in ("%d %b %Y", "%d %B %Y"):
        try:
            value = datetime.strptime(text, fmt).date()
        except ValueError:
            continue
        return datetime.combine(value, time.min, tzinfo=timezone.utc)
    return None


def normalize_zip_name(name: str) -> str:
    stem = Path(name.strip()).stem.lower()
    return re.sub(r"[^a-z0-9]+", "", stem)


def load_state(path: Path) -> dict[str, Any]:
    if not path.is_file():
        return {"requests": {}}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return {"requests": {}}
    if not isinstance(data, dict):
        return {"requests": {}}
    requests = data.setdefault("requests", {})
    if not isinstance(requests, dict):
        data["requests"] = {}
    return data


def save_state(path: Path, state: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(state, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def staff_report_paths(workspace: Path, explicit: Path | None) -> list[Path]:
    if explicit:
        return [explicit]
    report_dir = workspace / "runtime" / "output" / "reports" / "lms-watch"
    paths = [
        path
        for path in report_dir.glob("lms-watch-staff-*.csv")
        if path.name != "lms-watch-staff-latest.csv"
    ]
    latest = report_dir / "lms-watch-staff-latest.csv"
    if latest.is_file():
        paths.append(latest)
    return sorted(set(paths), key=lambda path: (path.stat().st_mtime, path.name))


def changed_rows_from_report(path: Path) -> list[LmsChange]:
    if not path.is_file():
        return []
    changes: list[LmsChange] = []
    seen_at = report_time(path)
    with path.open(newline="", encoding="utf-8-sig") as handle:
        for row in csv.DictReader(handle):
            if (row.get("status") or "").strip().lower() != "changed":
                continue
            latest_item = (row.get("latest_item") or "").strip()
            if not latest_item.lower().endswith(".zip"):
                continue
            changes.append(
                LmsChange(
                    course_code=(row.get("course_code") or "").strip(),
                    course_name=(row.get("course_name") or "").strip(),
                    latest_item=latest_item,
                    latest_modified=(row.get("latest_modified") or "").strip(),
                    check_url=(row.get("check_url") or "").strip(),
                    report_path=path,
                    report_time=seen_at,
                )
            )
    return changes


def add_lms_changes(state: dict[str, Any], changes: list[LmsChange]) -> None:
    requests = state.setdefault("requests", {})
    now = iso(utc_now())
    for change in changes:
        existing = requests.get(change.key)
        if isinstance(existing, dict) and existing.get("status") == "resolved":
            continue
        record = existing if isinstance(existing, dict) else {}
        record.update(
            {
                "course_code": change.course_code,
                "course_name": change.course_name,
                "latest_item": change.latest_item,
                "latest_modified": change.latest_modified,
                "check_url": change.check_url,
                "status": "pending",
                "last_seen_in_lms_report_at": iso(change.report_time),
                "last_seen_in_lms_report": str(change.report_path),
                "updated_at": now,
            }
        )
        record.setdefault("first_requested_at", iso(change.report_time))
        record.setdefault("created_at", now)
        record.pop("resolved_at", None)
        record.pop("matched_drive_name", None)
        record.pop("matched_drive_modified", None)
        requests[change.key] = record


def load_drive_records(args: argparse.Namespace, cfg: dict[str, Any], workspace: Path) -> tuple[list[dict[str, Any]], int]:
    if args.drive_json:
        data = json.loads(Path(args.drive_json).read_text(encoding="utf-8"))
        waiting = data.get("waiting", []) if isinstance(data, dict) else []
        recent = int(data.get("recent_count", 0)) if isinstance(data, dict) else 0
        return [item for item in waiting if isinstance(item, dict)], recent
    if drive_intake.api_mode(cfg):
        return check_drive_incoming.api_records(cfg, workspace)
    return check_drive_incoming.local_records(cfg, workspace)


def drive_record_matches(request: dict[str, Any], record: dict[str, Any]) -> bool:
    expected = normalize_zip_name(str(request.get("latest_item", "")))
    actual = normalize_zip_name(str(record.get("name", "")))
    if not expected or not actual:
        return False
    if actual != expected and not actual.startswith(expected):
        return False

    drive_modified = parse_iso(str(record.get("modified", "")))
    if drive_modified is None:
        return True

    lms_floor = parse_lms_modified_date(str(request.get("latest_modified", "")))
    if lms_floor is not None:
        return drive_modified >= lms_floor

    first_requested = parse_iso(str(request.get("first_requested_at", "")))
    if first_requested is not None:
        return drive_modified >= first_requested
    return True


def reconcile_state(state: dict[str, Any], drive_records: list[dict[str, Any]]) -> list[dict[str, Any]]:
    resolved: list[dict[str, Any]] = []
    for key, request in list(state.get("requests", {}).items()):
        if not isinstance(request, dict) or request.get("status") != "pending":
            continue
        match = next((record for record in drive_records if drive_record_matches(request, record)), None)
        if match is None:
            continue
        request["status"] = "resolved"
        request["resolved_at"] = iso(utc_now())
        request["matched_drive_name"] = match.get("name", "")
        request["matched_drive_modified"] = match.get("modified", "")
        request["updated_at"] = request["resolved_at"]
        resolved.append(request)
    return resolved


def pending_requests(state: dict[str, Any]) -> list[dict[str, Any]]:
    pending = [
        request
        for request in state.get("requests", {}).values()
        if isinstance(request, dict) and request.get("status") == "pending"
    ]
    return sorted(pending, key=lambda item: (item.get("course_code", ""), item.get("latest_item", "")))


def main() -> int:
    args = parse_args()
    program = Path(args.program_workspace).expanduser().resolve() if args.program_workspace else None
    cfg = drive_intake.load_json(drive_intake.resolve_config_path(args.config, program))
    workspace = drive_intake.program_workspace(cfg)
    state_path = Path(args.state_file).expanduser().resolve() if args.state_file else workspace / "runtime" / "lms-watch" / "pending_drive_uploads.json"
    output_path = Path(args.output_json).expanduser().resolve() if args.output_json else workspace / "runtime" / "lms-watch" / "pending_drive_uploads_latest.json"
    explicit_report = Path(args.report_csv).expanduser().resolve() if args.report_csv else None

    state = load_state(state_path)
    reports = staff_report_paths(workspace, explicit_report)
    changes: list[LmsChange] = []
    for report in reports:
        changes.extend(changed_rows_from_report(report))
    add_lms_changes(state, changes)

    drive_records, recent = load_drive_records(args, cfg, workspace)
    resolved = reconcile_state(state, drive_records)
    pending = pending_requests(state)
    save_state(state_path, state)

    payload = {
        "status": "ok",
        "workspace": workspace.name,
        "processed": False,
        "state_file": str(state_path),
        "pending_count": len(pending),
        "resolved_count": len(resolved),
        "recent_drive_count": recent,
        "drive_waiting_count": len(drive_records),
        "pending": pending,
        "resolved_this_run": resolved,
    }
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")

    if args.json:
        print(json.dumps(payload, indent=2, sort_keys=True))
        return 0

    print(f"LMS-to-Drive upload cross-check for {workspace.name}")
    print(f"Pending staff upload action items: {len(pending)}")
    if pending:
        for item in pending[: max(args.max_items, 0)]:
            label = " ".join(part for part in [item.get("course_code", ""), item.get("course_name", "")] if part).strip()
            print(f"- {label}: {item.get('latest_item', '')}")
        remaining = len(pending) - max(args.max_items, 0)
        if remaining > 0:
            print(f"- {remaining} additional pending item(s) omitted")
    else:
        print("No pending LMS upload requests are missing from Drive.")
    if resolved:
        print(f"Resolved this run: {len(resolved)}")
        for item in resolved[: max(args.max_items, 0)]:
            label = " ".join(part for part in [item.get("course_code", ""), item.get("course_name", "")] if part).strip()
            print(f"- {label}: {item.get('matched_drive_name', '')}")
    print(f"Stable Drive zip files checked: {len(drive_records)}")
    if recent:
        print(f"Recent/unstable Drive zip files ignored for now: {recent}")
    print("No Drive files were processed by this check.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
