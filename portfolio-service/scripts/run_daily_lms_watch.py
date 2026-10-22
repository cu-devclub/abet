#!/usr/bin/env python3
"""Run LMS Watch, notify staff, and process any new Drive intake files."""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]


def run(cmd: list[str], *, cwd: Path, env: dict[str, str]) -> int:
    print(f"Running: {' '.join(cmd)}")
    proc = subprocess.run(cmd, cwd=cwd, env=env)
    print(f"Exit code: {proc.returncode}")
    return proc.returncode


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run the daily CourseDisplay LMS automation.")
    parser.add_argument("--program-workspace", required=True, help="Program workspace, e.g. programs/ice-ay2025")
    parser.add_argument("--notify-dry-run", action="store_true", help="Print email notification instead of sending.")
    parser.add_argument("--notify-force", action="store_true", help="Send notification even if already sent.")
    parser.add_argument("--notify-changes-only", action="store_true", help="Only email newly detected changed files, not the daily status report.")
    parser.add_argument("--skip-lms", action="store_true", help="Skip LMS check and only notify from latest report.")
    parser.add_argument("--skip-notify", action="store_true", help="Run LMS check without email notification.")
    parser.add_argument("--process-drive", action="store_true", help="Deprecated: Drive processing now runs by default.")
    parser.add_argument("--skip-drive", action="store_true", help="Skip processing unprocessed Google Drive incoming zips.")
    parser.add_argument("--test-email", action="store_true", help="Send a harmless test email instead of LMS-change notification.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    workspace = Path(args.program_workspace).expanduser()
    if not workspace.is_absolute():
        workspace = REPO_ROOT / workspace
    workspace = workspace.resolve()
    env = os.environ.copy()
    env["COURSEDISPLAY_WORKSPACE_ROOT"] = str(workspace)

    worst = 0
    if not args.skip_lms:
        lms_script = REPO_ROOT / "scripts" / "check_lms_updates.py"
        worst = max(worst, run([sys.executable, str(lms_script)], cwd=workspace, env=env))

    if not args.skip_notify:
        notify_cmd = [
            sys.executable,
            str(REPO_ROOT / "scripts" / "notify_lms_updates.py"),
            "--workspace",
            str(workspace),
        ]
        if args.notify_dry_run:
            notify_cmd.append("--dry-run")
        if args.notify_force:
            notify_cmd.append("--force")
        if not args.notify_changes_only and not args.test_email:
            notify_cmd.append("--daily-report")
        if args.test_email:
            notify_cmd.append("--test-email")
        worst = max(worst, run(notify_cmd, cwd=workspace, env=env))

    if args.process_drive:
        print("--process-drive is no longer required; Drive processing runs by default.")

    if not args.skip_drive:
        drive_cmd = [
            sys.executable,
            str(REPO_ROOT / "scripts" / "drive_intake.py"),
            "--program-workspace",
            str(workspace),
            "process-all",
        ]
        worst = max(worst, run(drive_cmd, cwd=REPO_ROOT, env=env))
    return worst


if __name__ == "__main__":
    raise SystemExit(main())
