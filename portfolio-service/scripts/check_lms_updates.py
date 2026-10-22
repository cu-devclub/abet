#!/usr/bin/env python3
"""Check LMS course pages for newly modified files using a saved browser session."""

from __future__ import annotations

import argparse
import csv
import json
import os
import shutil
import sys
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Any
from urllib.parse import urlparse

from create_program_workspace import backfill_workspace

SHARED_ROOT = Path(__file__).resolve().parents[1]

COURSE_HEADERS = [
    "enabled",
    "course_code",
    "course_name",
    "check_url",
    "row_selector",
    "name_selector",
    "modified_selector",
    "last_seen_modified",
    "last_seen_item",
    "last_checked_at",
    "last_status",
    "last_note",
]

DEFAULT_SETTINGS = {
    "login_url": "",
    "browser_channel": "chrome",
    "max_report_history": 10,
    "page_ready_selector": "",
    "row_selector": "",
    "name_selector": "",
    "modified_selector": "",
    "rows_sorted_newest_first": True,
}


@dataclass
class CourseResult:
    course_code: str
    course_name: str
    check_url: str
    status: str
    changed_items: list[dict[str, str]]
    latest_item: str
    latest_modified: str
    previous_item: str
    previous_modified: str
    note: str


class LmsWatchError(RuntimeError):
    pass


def detect_workspace_root() -> Path:
    env_root = os.environ.get("COURSEDISPLAY_WORKSPACE_ROOT")
    if env_root:
        return Path(env_root).resolve()
    cwd = Path.cwd()
    if (cwd / "config" / "staff" / "course_registry.csv").is_file():
        return cwd.resolve()
    return SHARED_ROOT


WORKSPACE_ROOT = detect_workspace_root()
DEFAULT_COURSES_CSV = WORKSPACE_ROOT / "config" / "staff" / "lms_watch_courses.csv"
DEFAULT_SETTINGS_JSON = WORKSPACE_ROOT / "config" / "staff" / "lms_watch_settings.json"
DEFAULT_OUTPUT_DIR = WORKSPACE_ROOT / "runtime" / "output" / "reports" / "lms-watch"
DEFAULT_STATE_FILE = WORKSPACE_ROOT / "runtime" / "lms-watch" / "storage_state.json"
DEFAULT_PROGRAM_PROFILE_DIR = WORKSPACE_ROOT / "runtime" / "lms-watch" / "chrome-profile"
DEFAULT_PROGRESS_FILE = WORKSPACE_ROOT / "runtime" / "lms-watch" / "progress.json"
DEFAULT_LAST_SEEN_FILE = WORKSPACE_ROOT / "runtime" / "lms-watch" / "last_seen.json"


def now_iso() -> str:
    return datetime.now().isoformat(timespec="seconds")


def normalize_bool(value: str) -> bool:
    return value.strip().lower() not in {"", "0", "false", "no", "n"}


def load_settings(path: Path) -> dict[str, Any]:
    if not path.is_file():
        raise SystemExit(f"LMS settings file not found: {path}")
    data = json.loads(path.read_text(encoding="utf-8"))
    settings = DEFAULT_SETTINGS.copy()
    settings.update(data)
    required = ["login_url"]
    missing = [key for key in required if not str(settings.get(key, "")).strip()]
    if missing:
        raise SystemExit(
            "LMS settings file is missing required keys: " + ", ".join(missing) + f" ({path})"
        )
    return settings


def load_courses(path: Path) -> list[dict[str, str]]:
    if not path.is_file():
        raise SystemExit(f"LMS course CSV not found: {path}")
    with path.open(newline="", encoding="utf-8-sig") as handle:
        reader = csv.DictReader(handle)
        rows: list[dict[str, str]] = []
        for row in reader:
            normalized = {header: (row.get(header) or "").strip() for header in COURSE_HEADERS}
            if not normalized["course_code"] and not normalized["course_name"] and not normalized["check_url"]:
                continue
            rows.append(normalized)
        return rows


def course_state_key(course: dict[str, str]) -> str:
    code = str(course.get("course_code", "")).strip()
    if code:
        return code
    return str(course.get("course_name", "")).strip()


def load_last_seen_state(path: Path) -> dict[str, Any]:
    if not path.is_file():
        return {"courses": {}}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return {"courses": {}}
    if not isinstance(data, dict):
        return {"courses": {}}
    courses = data.setdefault("courses", {})
    if not isinstance(courses, dict):
        data["courses"] = {}
    return data


def apply_last_seen_state(courses: list[dict[str, str]], state: dict[str, Any]) -> None:
    state_courses = state.get("courses", {})
    if not isinstance(state_courses, dict):
        return
    for course in courses:
        stored = state_courses.get(course_state_key(course))
        if not isinstance(stored, dict):
            continue
        for key in ["last_seen_modified", "last_seen_item", "last_checked_at", "last_status", "last_note"]:
            value = stored.get(key)
            if value is not None:
                course[key] = str(value)


def update_last_seen_state(state: dict[str, Any], course: dict[str, str]) -> None:
    courses = state.setdefault("courses", {})
    if not isinstance(courses, dict):
        courses = {}
        state["courses"] = courses
    courses[course_state_key(course)] = {
        "course_code": course.get("course_code", ""),
        "course_name": course.get("course_name", ""),
        "last_seen_modified": course.get("last_seen_modified", ""),
        "last_seen_item": course.get("last_seen_item", ""),
        "last_checked_at": course.get("last_checked_at", ""),
        "last_status": course.get("last_status", ""),
        "last_note": course.get("last_note", ""),
    }


def save_last_seen_state(path: Path, state: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(state, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def course_label(course: dict[str, str]) -> str:
    code = str(course.get("course_code", "")).strip()
    name = str(course.get("course_name", "")).strip()
    if code and name:
        return f"{code} {name}"
    if code:
        return code
    if name:
        return name
    return "<unnamed course row>"


def find_enabled_course_url_issue_details(courses: list[dict[str, str]]) -> list[dict[str, str]]:
    issues: list[dict[str, str]] = []
    for row_number, course in enumerate(courses, start=2):
        if not normalize_bool(course.get("enabled", "") or "true"):
            continue
        url = str(course.get("check_url", "")).strip()
        if not url:
            issues.append(
                {
                    "row": str(row_number),
                    "course_code": str(course.get("course_code", "")).strip(),
                    "course_name": str(course.get("course_name", "")).strip(),
                    "issue": "missing check_url",
                    "check_url": "",
                }
            )
            continue
        parsed = urlparse(url)
        if parsed.scheme not in {"http", "https"} or not parsed.netloc:
            issues.append(
                {
                    "row": str(row_number),
                    "course_code": str(course.get("course_code", "")).strip(),
                    "course_name": str(course.get("course_name", "")).strip(),
                    "issue": "invalid check_url",
                    "check_url": url,
                }
            )
    return issues


def find_enabled_course_url_errors(courses: list[dict[str, str]]) -> list[str]:
    issues: list[str] = []
    for issue in find_enabled_course_url_issue_details(courses):
        label = course_label(issue)
        if issue["issue"] == "missing check_url":
            issues.append(f"{label} (missing check_url)")
            continue
        issues.append(f"{label} (invalid check_url: {issue['check_url']})")
    return issues


def save_courses(path: Path, rows: list[dict[str, str]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=COURSE_HEADERS)
        writer.writeheader()
        writer.writerows(rows)


def ensure_playwright() -> None:
    try:
        import playwright  # noqa: F401
    except ImportError as exc:
        raise SystemExit(
            "playwright is not installed. Install it with: python3 -m pip install -r requirements.txt "
            "and then: python3 -m playwright install chromium"
        ) from exc


def browser_launch_kwargs(settings: dict[str, Any], *, headless: bool) -> dict[str, Any]:
    channel = str(settings.get("browser_channel", "")).strip()
    kwargs: dict[str, Any] = {
        "headless": headless,
        "ignore_default_args": ["--enable-automation"],
        "args": ["--disable-blink-features=AutomationControlled"],
    }
    if channel:
        kwargs["channel"] = channel
    return kwargs


def resolve_profile_dir(settings: dict[str, Any]) -> Path:
    configured = str(settings.get("persistent_profile_dir", "")).strip()
    if configured:
        return Path(configured).expanduser().resolve()
    scope = str(settings.get("persistent_profile_scope", "staff")).strip().lower()
    if scope == "program":
        return DEFAULT_PROGRAM_PROFILE_DIR
    return Path.home() / ".coursedisplay" / "lms-watch" / "chrome-profile"


def launch_browser(playwright: Any, settings: dict[str, Any], *, headless: bool) -> Any:
    channel = str(settings.get("browser_channel", "")).strip()
    kwargs = browser_launch_kwargs(settings, headless=headless)
    try:
        return playwright.chromium.launch(**kwargs)
    except Exception as exc:
        if channel:
            raise SystemExit(
                f"Failed to launch the configured browser channel '{channel}'. "
                "Install that browser or update browser_channel in lms_watch_settings.json."
            ) from exc
        raise


def storage_state_has_cookies(state_file: Path) -> bool:
    try:
        data = json.loads(state_file.read_text(encoding="utf-8"))
    except (FileNotFoundError, json.JSONDecodeError):
        return False
    cookies = data.get("cookies")
    return isinstance(cookies, list) and bool(cookies)


def launch_check_context(
    playwright: Any, settings: dict[str, Any], state_file: Path, profile_dir: Path
) -> tuple[Any, Any | None]:
    if storage_state_has_cookies(state_file):
        browser = launch_browser(playwright, settings, headless=True)
        return browser.new_context(storage_state=str(state_file)), browser
    return (
        playwright.chromium.launch_persistent_context(
            str(profile_dir),
            **browser_launch_kwargs(settings, headless=True),
        ),
        None,
    )


def close_check_context(context: Any, browser: Any | None) -> None:
    context.close()
    if browser is not None:
        browser.close()


def launch_login(settings: dict[str, Any], state_path: Path, profile_dir: Path) -> None:
    from playwright.sync_api import sync_playwright

    state_path.parent.mkdir(parents=True, exist_ok=True)
    with sync_playwright() as playwright:
        launch_login_with_playwright(playwright, settings, state_path, profile_dir)


def launch_login_with_playwright(
    playwright: Any, settings: dict[str, Any], state_path: Path, profile_dir: Path
) -> None:
    state_path.parent.mkdir(parents=True, exist_ok=True)
    profile_dir.mkdir(parents=True, exist_ok=True)
    try:
        context = playwright.chromium.launch_persistent_context(
            str(profile_dir),
            **browser_launch_kwargs(settings, headless=False),
        )
    except Exception as exc:
        channel = str(settings.get("browser_channel", "")).strip() or "chromium"
        raise SystemExit(
            f"Failed to launch browser for login using '{channel}'. "
            "Close existing browser automation sessions and try again."
        ) from exc

    page = context.new_page()
    print()
    print("LMS login required.")
    print(f"Opening LMS login page: {settings['login_url']}")
    print("ACTION: finish the LMS login in the browser window that opens.")
    print("Do not press Enter here until the browser shows an accessible LMS course page.")
    page.goto(str(settings["login_url"]), wait_until="domcontentloaded")
    print()
    input("When the LMS page is loaded in the browser, return here and press Enter to save the session: ")
    context.storage_state(path=str(state_path))
    print(f"Saved LMS browser session to {state_path}")
    print(f"Persistent LMS browser profile path: {profile_dir}")
    print("This profile is local to this machine and is not shared through GitHub.")
    context.close()


def extract_text(row_locator: Any, selector: str) -> str:
    target = row_locator if not selector else row_locator.locator(selector).first
    text = ""
    try:
        text = target.inner_text(timeout=3000) or ""
    except Exception:
        text = ""
    if not text.strip():
        try:
            text = target.text_content(timeout=3000) or ""
        except Exception:
            return ""
    return " ".join(text.split())


def normalize_modified_text(value: str) -> str:
    text = " ".join(value.split())
    prefix = "Last Modified:"
    if text.startswith(prefix):
        return text[len(prefix) :].strip()
    return text


def scrape_single_item_page(page: Any, settings: dict[str, Any]) -> list[dict[str, str]]:
    container_selector = ".courseville-view-content-material"
    name_selector = ".courseville-view-content-material-title"
    modified_selector = ".courseville-view-content-modification-info"
    wait_selector = str(settings.get("page_ready_selector") or container_selector)

    page.wait_for_selector(wait_selector, timeout=30000)
    container = page.locator(container_selector).first
    item_name = extract_text(container, name_selector)
    item_modified = normalize_modified_text(extract_text(container, modified_selector))
    if not item_name and not item_modified:
        return []
    return [{"item": item_name, "modified": item_modified}]


def scrape_course(page: Any, course: dict[str, str], settings: dict[str, Any]) -> list[dict[str, str]]:
    check_url = (course.get("check_url") or "").strip()
    if not check_url:
        raise LmsWatchError("Missing check_url for enabled course row.")
    row_selector = course["row_selector"] or str(settings["row_selector"])
    name_selector = course["name_selector"] or str(settings["name_selector"])
    modified_selector = course["modified_selector"] or str(settings["modified_selector"])
    wait_selector = str(settings.get("page_ready_selector") or row_selector or "body")

    page.goto(check_url, wait_until="domcontentloaded", timeout=60000)
    if not row_selector:
        return scrape_single_item_page(page, settings)

    page.wait_for_selector(wait_selector, timeout=30000)

    rows = page.locator(row_selector)
    row_count = rows.count()
    items: list[dict[str, str]] = []
    for index in range(row_count):
        row = rows.nth(index)
        item_name = extract_text(row, name_selector)
        item_modified = normalize_modified_text(extract_text(row, modified_selector))
        if not item_name and not item_modified:
            continue
        items.append({"item": item_name, "modified": item_modified})
    return items


def evaluate_course(course: dict[str, str], items: list[dict[str, str]], settings: dict[str, Any]) -> CourseResult:
    previous_modified = course["last_seen_modified"]
    previous_item = course["last_seen_item"]

    if not items:
        return CourseResult(
            course_code=course["course_code"],
            course_name=course["course_name"],
            check_url=course["check_url"],
            status="error",
            changed_items=[],
            latest_item="",
            latest_modified="",
            previous_item=previous_item,
            previous_modified=previous_modified,
            note="No LMS items matched the configured selectors or supported page structure.",
        )

    latest = items[0]
    latest_item = latest["item"]
    latest_modified = latest["modified"]

    if not previous_modified and not previous_item:
        return CourseResult(
            course_code=course["course_code"],
            course_name=course["course_name"],
            check_url=course["check_url"],
            status="baseline-set",
            changed_items=[],
            latest_item=latest_item,
            latest_modified=latest_modified,
            previous_item=previous_item,
            previous_modified=previous_modified,
            note="Baseline created from the current top row.",
        )

    if latest_modified == previous_modified and latest_item == previous_item:
        return CourseResult(
            course_code=course["course_code"],
            course_name=course["course_name"],
            check_url=course["check_url"],
            status="unchanged",
            changed_items=[],
            latest_item=latest_item,
            latest_modified=latest_modified,
            previous_item=previous_item,
            previous_modified=previous_modified,
            note="No newer LMS file was detected.",
        )

    changed_items: list[dict[str, str]] = []
    if bool(settings.get("rows_sorted_newest_first", True)):
        for item in items:
            if item["modified"] == previous_modified and item["item"] == previous_item:
                break
            changed_items.append(item)
    else:
        changed_items.append(latest)

    note = f"{len(changed_items)} new or changed file(s) detected since the last check."
    return CourseResult(
        course_code=course["course_code"],
        course_name=course["course_name"],
        check_url=course["check_url"],
        status="changed",
        changed_items=changed_items,
        latest_item=latest_item,
        latest_modified=latest_modified,
        previous_item=previous_item,
        previous_modified=previous_modified,
        note=note,
    )


def action_label(result: CourseResult) -> str:
    if result.status == "changed":
        return "Download latest file"
    if result.status == "error":
        return "Check LMS link/login"
    if result.status == "baseline-set":
        return "Monitoring started"
    return ""


def prune_report_history(output_dir: Path, keep_count: int) -> None:
    if keep_count < 1:
        keep_count = 1
    prefixes = ["lms-watch-", "lms-watch-staff-"]
    suffixes = {".md", ".json", ".csv"}
    latest_names = {
        "lms-watch-latest.md",
        "lms-watch-latest.json",
        "lms-watch-staff-latest.md",
        "lms-watch-staff-latest.csv",
    }

    for prefix in prefixes:
        files = []
        for path in output_dir.iterdir():
            if not path.is_file():
                continue
            if path.name in latest_names:
                continue
            if not path.name.startswith(prefix):
                continue
            if path.suffix not in suffixes:
                continue
            files.append(path)

        groups: dict[str, list[Path]] = {}
        for path in files:
            stem = path.stem
            timestamp = stem[len(prefix) :]
            groups.setdefault(timestamp, []).append(path)

        timestamps = sorted(groups.keys(), reverse=True)
        for timestamp in timestamps[keep_count:]:
            for path in groups[timestamp]:
                path.unlink(missing_ok=True)


def write_report(output_dir: Path, results: list[CourseResult]) -> dict[str, Path]:
    output_dir.mkdir(parents=True, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    json_path = output_dir / f"lms-watch-{timestamp}.json"
    md_path = output_dir / f"lms-watch-{timestamp}.md"
    latest_json = output_dir / "lms-watch-latest.json"
    latest_md = output_dir / "lms-watch-latest.md"
    staff_md_path = output_dir / f"lms-watch-staff-{timestamp}.md"
    staff_csv_path = output_dir / f"lms-watch-staff-{timestamp}.csv"
    latest_staff_md = output_dir / "lms-watch-staff-latest.md"
    latest_staff_csv = output_dir / "lms-watch-staff-latest.csv"

    payload = {
        "generated_at": now_iso(),
        "results": [
            {
                "course_code": result.course_code,
                "course_name": result.course_name,
                "check_url": result.check_url,
                "status": result.status,
                "latest_item": result.latest_item,
                "latest_modified": result.latest_modified,
                "previous_item": result.previous_item,
                "previous_modified": result.previous_modified,
                "action": action_label(result),
                "note": result.note,
                "changed_items": result.changed_items,
            }
            for result in results
        ],
    }
    text_json = json.dumps(payload, indent=2) + "\n"
    json_path.write_text(text_json, encoding="utf-8")
    latest_json.write_text(text_json, encoding="utf-8")

    sort_order = {"changed": 0, "error": 1, "baseline-set": 2, "unchanged": 3}
    sorted_results = sorted(
        results,
        key=lambda result: (
            sort_order.get(result.status, 9),
            result.course_code.lower(),
            result.course_name.lower(),
        ),
    )

    lines = [
        "# LMS Watch Summary",
        "",
        f"Generated: {payload['generated_at']}",
        "",
        "| Course | Status | Latest Modified | Latest Item | Note |",
        "| --- | --- | --- | --- | --- |",
    ]
    for result in sorted_results:
        course_label = f"{result.course_code} {result.course_name}".strip()
        lines.append(
            f"| {course_label} | {result.status} | {result.latest_modified or '-'} | {result.latest_item or '-'} | {result.note} |"
        )
        if result.changed_items:
            lines.append("")
            lines.append(f"Changed files for `{course_label}`:")
            for item in result.changed_items:
                lines.append(f"- `{item['modified']}` - {item['item']}")
            lines.append("")
    text_md = "\n".join(lines).strip() + "\n"
    md_path.write_text(text_md, encoding="utf-8")
    latest_md.write_text(text_md, encoding="utf-8")

    changed_results = [result for result in sorted_results if result.status == "changed"]
    error_results = [result for result in sorted_results if result.status == "error"]

    staff_lines = [
        "# LMS Download Queue",
        "",
        f"Generated: {payload['generated_at']}",
        "",
        f"- Courses to download now: {len(changed_results)}",
        f"- Courses with check errors: {len(error_results)}",
        f"- Courses monitored: {len(sorted_results)}",
        "",
    ]
    if changed_results:
        staff_lines.extend(
            [
                "## Download Now",
                "",
                "| Course | Latest Modified | Latest File | Previous Modified | LMS Page |",
                "| --- | --- | --- | --- | --- |",
            ]
        )
        for result in changed_results:
            course_label = f"{result.course_code} {result.course_name}".strip()
            staff_lines.append(
                f"| {course_label} | {result.latest_modified or '-'} | {result.latest_item or '-'} | {result.previous_modified or '-'} | {result.check_url or '-'} |"
            )
        staff_lines.append("")
    else:
        staff_lines.extend(["## Download Now", "", "No courses need a new download right now.", ""])

    if error_results:
        staff_lines.extend(
            [
                "## Check Errors",
                "",
                "| Course | Issue | LMS Page |",
                "| --- | --- | --- |",
            ]
        )
        for result in error_results:
            course_label = f"{result.course_code} {result.course_name}".strip()
            staff_lines.append(f"| {course_label} | {result.note} | {result.check_url or '-'} |")
        staff_lines.append("")

    staff_lines.extend(
        [
            "## All Courses",
            "",
            "| Course | Status | Action | Latest Modified | Latest File | LMS Page |",
            "| --- | --- | --- | --- | --- | --- |",
        ]
    )
    for result in sorted_results:
        course_label = f"{result.course_code} {result.course_name}".strip()
        staff_lines.append(
            f"| {course_label} | {result.status} | {action_label(result) or '-'} | {result.latest_modified or '-'} | {result.latest_item or '-'} | {result.check_url or '-'} |"
        )
    staff_text = "\n".join(staff_lines).strip() + "\n"
    staff_md_path.write_text(staff_text, encoding="utf-8")
    latest_staff_md.write_text(staff_text, encoding="utf-8")

    staff_csv_headers = [
        "course_code",
        "course_name",
        "status",
        "action",
        "latest_modified",
        "latest_item",
        "previous_modified",
        "previous_item",
        "check_url",
        "note",
    ]
    for target in [staff_csv_path, latest_staff_csv]:
        with target.open("w", newline="", encoding="utf-8") as handle:
            writer = csv.DictWriter(handle, fieldnames=staff_csv_headers)
            writer.writeheader()
            for result in sorted_results:
                writer.writerow(
                    {
                        "course_code": result.course_code,
                        "course_name": result.course_name,
                        "status": result.status,
                        "action": action_label(result),
                        "latest_modified": result.latest_modified,
                        "latest_item": result.latest_item,
                        "previous_modified": result.previous_modified,
                        "previous_item": result.previous_item,
                        "check_url": result.check_url,
                        "note": result.note,
                    }
                )
    return {
        "json": json_path,
        "markdown": md_path,
        "latest_json": latest_json,
        "latest_markdown": latest_md,
        "staff_markdown": staff_md_path,
        "staff_csv": staff_csv_path,
        "latest_staff_markdown": latest_staff_md,
        "latest_staff_csv": latest_staff_csv,
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Check LMS course pages for newly modified files")
    parser.add_argument("--courses-csv", default=str(DEFAULT_COURSES_CSV), help="Per-program LMS watch course CSV")
    parser.add_argument("--settings", default=str(DEFAULT_SETTINGS_JSON), help="Per-program LMS watch settings JSON")
    parser.add_argument("--output-dir", default=str(DEFAULT_OUTPUT_DIR), help="Directory for LMS watch reports")
    parser.add_argument("--state-file", default=str(DEFAULT_STATE_FILE), help="Browser storage state file")
    parser.add_argument("--relogin", action="store_true", help="Force a new interactive login before checking courses")
    parser.add_argument("--login-only", action="store_true", help="Open the browser, save login state, and exit")
    parser.add_argument("--no-update", action="store_true", help="Do not write new last-seen values back to the CSV")
    parser.add_argument("--progress-file", default=str(DEFAULT_PROGRESS_FILE), help="Runtime progress status JSON path")
    parser.add_argument("--last-seen-file", default=str(DEFAULT_LAST_SEEN_FILE), help="Runtime LMS last-seen state JSON path")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    ensure_playwright()
    backfill_workspace(WORKSPACE_ROOT, refresh_launchers=True)
    courses_csv = Path(args.courses_csv).resolve()
    settings_path = Path(args.settings).resolve()
    output_dir = Path(args.output_dir).resolve()
    state_file = Path(args.state_file).resolve()
    progress_file = Path(args.progress_file).resolve()
    last_seen_file = Path(args.last_seen_file).resolve()

    def write_progress(
        *,
        state: str,
        completed: int,
        total: int,
        current_course: str = "",
        message: str = "",
    ) -> None:
        progress_file.parent.mkdir(parents=True, exist_ok=True)
        payload = {
            "updated_at": now_iso(),
            "state": state,
            "completed": completed,
            "total": total,
            "current_course": current_course,
            "message": message,
            "percent": round((completed / total) * 100, 1) if total > 0 else 0.0,
        }
        progress_file.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")

    settings = load_settings(settings_path)
    profile_dir = resolve_profile_dir(settings)
    courses = load_courses(courses_csv)
    last_seen_state = load_last_seen_state(last_seen_file)
    apply_last_seen_state(courses, last_seen_state)
    enabled_courses = [course for course in courses if normalize_bool(course["enabled"] or "true")]
    if not enabled_courses:
        raise SystemExit(f"No enabled LMS watch rows found in: {courses_csv}")
    write_progress(
        state="starting",
        completed=0,
        total=len(enabled_courses),
        message="Starting LMS Watch run.",
    )
    course_url_issues = find_enabled_course_url_errors(courses)
    if course_url_issues:
        lines = [
            "LMS watch configuration error: enabled rows must include a valid http(s) check_url.",
            "Fix or disable the following rows before running LMS Watch:",
        ]
        lines.extend(f"- {issue}" for issue in course_url_issues)
        raise SystemExit("\n".join(lines))

    interactive_login_performed = False
    if args.relogin or not state_file.is_file():
        if args.relogin:
            print("--relogin requested: starting a fresh LMS browser login before checking courses.")
        else:
            print(f"No saved LMS browser session found at {state_file}.")
            print("Starting a one-time LMS browser login before checking courses.")
        launch_login(settings, state_file, profile_dir)
        interactive_login_performed = True
        write_progress(
            state="running",
            completed=0,
            total=len(enabled_courses),
            message="Login completed. Starting course checks.",
        )
        if args.login_only:
            print(f"Saved LMS login session to {state_file}")
            write_progress(
                state="completed",
                completed=0,
                total=0,
                message="Login-only run completed.",
            )
            return 0
    elif args.login_only:
        print(f"Existing LMS login session already present at {state_file}")
        return 0

    from playwright.sync_api import TimeoutError as PlaywrightTimeoutError
    from playwright.sync_api import sync_playwright

    results: list[CourseResult] = []
    # Allow one automatic refresh-login on timeout even if this run already
    # performed an initial interactive login (e.g., first-time state capture).
    relogin_attempted = False

    with sync_playwright() as playwright:
        context, browser = launch_check_context(playwright, settings, state_file, profile_dir)
        page = context.new_page()

        for index, course in enumerate(enabled_courses, start=1):
            label = course_label(course)
            write_progress(
                state="running",
                completed=index - 1,
                total=len(enabled_courses),
                current_course=label,
                message=f"Checking course {index}/{len(enabled_courses)}.",
            )
            try:
                items = scrape_course(page, course, settings)
                result = evaluate_course(course, items, settings)
            except PlaywrightTimeoutError:
                if not relogin_attempted:
                    print()
                    print(f"Timed out checking {course_label(course)}.")
                    print("Starting one refresh login because the saved LMS session may have expired.")
                    print("After the browser opens, log in again and press Enter in this terminal when the LMS page is accessible.")
                    close_check_context(context, browser)
                    write_progress(
                        state="waiting-login",
                        completed=index - 1,
                        total=len(enabled_courses),
                        current_course=label,
                        message="Waiting for refresh login to continue.",
                    )
                    launch_login_with_playwright(playwright, settings, state_file, profile_dir)
                    relogin_attempted = True
                    context, browser = launch_check_context(playwright, settings, state_file, profile_dir)
                    page = context.new_page()
                    try:
                        items = scrape_course(page, course, settings)
                        result = evaluate_course(course, items, settings)
                    except Exception as exc:  # noqa: BLE001
                        result = CourseResult(
                            course_code=course["course_code"],
                            course_name=course["course_name"],
                            check_url=course["check_url"],
                            status="error",
                            changed_items=[],
                            latest_item="",
                            latest_modified="",
                            previous_item=course["last_seen_item"],
                            previous_modified=course["last_seen_modified"],
                            note=(
                                "Failed after the one-time refresh login. "
                                "No additional login prompt was shown in this run. "
                                f"Details: {exc}"
                            ),
                        )
                else:
                    result = CourseResult(
                        course_code=course["course_code"],
                        course_name=course["course_name"],
                        check_url=course["check_url"],
                        status="error",
                        changed_items=[],
                        latest_item="",
                        latest_modified="",
                        previous_item=course["last_seen_item"],
                        previous_modified=course["last_seen_modified"],
                        note=(
                            "Timed out waiting for the LMS page or configured selectors. "
                            "An interactive LMS login was already completed earlier in this run, "
                            "so no second refresh-login prompt was opened. Re-run with --relogin "
                            "if the saved browser session needs to be refreshed again."
                        ),
                    )
            except Exception as exc:  # noqa: BLE001
                result = CourseResult(
                    course_code=course["course_code"],
                    course_name=course["course_name"],
                    check_url=course["check_url"],
                    status="error",
                    changed_items=[],
                    latest_item="",
                    latest_modified="",
                    previous_item=course["last_seen_item"],
                    previous_modified=course["last_seen_modified"],
                    note=str(exc),
                )

            course["last_checked_at"] = now_iso()
            course["last_status"] = result.status
            course["last_note"] = result.note
            if not args.no_update and result.status in {"baseline-set", "unchanged", "changed"}:
                course["last_seen_item"] = result.latest_item
                course["last_seen_modified"] = result.latest_modified
                update_last_seen_state(last_seen_state, course)
            results.append(result)
            write_progress(
                state="running",
                completed=index,
                total=len(enabled_courses),
                current_course=label,
                message=f"Completed course {index}/{len(enabled_courses)} ({result.status}).",
            )

        close_check_context(context, browser)

    if not args.no_update:
        save_courses(courses_csv, courses)
        save_last_seen_state(last_seen_file, last_seen_state)
    report_paths = write_report(output_dir, results)
    prune_report_history(output_dir, int(settings.get("max_report_history", 10)))

    changed = [result for result in results if result.status == "changed"]
    errors = [result for result in results if result.status == "error"]

    print(f"LMS watch report: {report_paths['markdown']}")
    for result in changed:
        print(f"CHANGED: {result.course_code} {result.course_name}")
        for item in result.changed_items:
            print(f"  - {item['modified']} :: {item['item']}")
    for result in errors:
        print(f"ERROR: {result.course_code} {result.course_name} :: {result.note}")
    write_progress(
        state="failed" if errors else "completed",
        completed=len(enabled_courses),
        total=len(enabled_courses),
        message="Run completed with errors." if errors else "Run completed successfully.",
    )

    return 1 if errors else 0


if __name__ == "__main__":
    raise SystemExit(main())
