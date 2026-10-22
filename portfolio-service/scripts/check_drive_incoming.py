#!/usr/bin/env python3
"""Report Drive intake zip files waiting in incoming/ without processing them."""

from __future__ import annotations

import argparse
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import drive_intake


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Check Drive intake incoming/ for waiting zip files.")
    parser.add_argument("--config", help="Path to drive_intake.json. Defaults to program config when --program-workspace is supplied.")
    parser.add_argument("--program-workspace", help="Program workspace, e.g. programs/ice-ay2025")
    parser.add_argument("--max-items", type=int, default=12, help="Maximum waiting zip names to print.")
    parser.add_argument("--json", action="store_true", help="Print machine-readable JSON.")
    return parser.parse_args()


def human_size(raw: object) -> str:
    try:
        size = int(raw)
    except (TypeError, ValueError):
        return ""
    units = ["B", "KB", "MB", "GB"]
    value = float(size)
    for unit in units:
        if value < 1024 or unit == units[-1]:
            return f"{value:.1f} {unit}" if unit != "B" else f"{size} B"
        value /= 1024
    return f"{size} B"


def api_records(cfg: dict[str, Any], workspace: Path) -> tuple[list[dict[str, Any]], int]:
    service = drive_intake.drive_service(cfg)
    folders = drive_intake.api_folder_ids(service, cfg, ensure=False)
    stable_seconds = int(cfg.get("stable_seconds", 30))
    records: list[dict[str, Any]] = []
    recent = 0
    for item in drive_intake.api_ready_zip_files(service, cfg, folders):
        stable = drive_intake.api_item_is_stable(item, stable_seconds)
        if not stable:
            recent += 1
            continue
        records.append(
            {
                "name": item.get("name", ""),
                "size": item.get("size", ""),
                "size_label": human_size(item.get("size", "")),
                "modified": item.get("modifiedTime", ""),
                "id": item.get("id", ""),
                "source": "google_api",
            }
        )
    return records, recent


def local_records(cfg: dict[str, Any], workspace: Path) -> tuple[list[dict[str, Any]], int]:
    paths = drive_intake.drive_paths(cfg)
    stable_seconds = int(cfg.get("stable_seconds", 30))
    records: list[dict[str, Any]] = []
    recent = 0
    for path in drive_intake.nonjunk_zips(paths.incoming):
        if not drive_intake.is_stable(path, stable_seconds):
            recent += 1
            continue
        stat = path.stat()
        records.append(
            {
                "name": path.name,
                "size": stat.st_size,
                "size_label": human_size(stat.st_size),
                "modified": datetime.fromtimestamp(stat.st_mtime, tz=timezone.utc).isoformat(),
                "path": str(path),
                "source": "local_drive",
            }
        )
    return records, recent


def main() -> int:
    args = parse_args()
    program = Path(args.program_workspace).expanduser().resolve() if args.program_workspace else None
    cfg = drive_intake.load_json(drive_intake.resolve_config_path(args.config, program))
    workspace = drive_intake.program_workspace(cfg)
    if drive_intake.api_mode(cfg):
        records, recent = api_records(cfg, workspace)
    else:
        records, recent = local_records(cfg, workspace)

    payload = {
        "status": "ok",
        "workspace": workspace.name,
        "waiting_count": len(records),
        "recent_count": recent,
        "waiting": records,
        "processed": False,
    }
    if args.json:
        print(json.dumps(payload, indent=2, sort_keys=True))
        return 0

    print(f"Drive incoming check for {workspace.name}")
    print(f"Stable zip files waiting: {len(records)}")
    if recent:
        print(f"Recent/unstable zip files ignored for now: {recent}")
    if records:
        for item in records[: max(args.max_items, 0)]:
            detail = item["size_label"] or "size unknown"
            modified = item.get("modified") or ""
            suffix = f", modified {modified}" if modified else ""
            print(f"- {item['name']} ({detail}{suffix})")
        remaining = len(records) - max(args.max_items, 0)
        if remaining > 0:
            print(f"- {remaining} additional zip file(s) omitted")
    else:
        print("No zip files are waiting in Drive incoming.")
    print("No Drive files were processed by this check.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
