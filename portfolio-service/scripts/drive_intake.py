#!/usr/bin/env python3
"""Drive-backed intake coordinator for CourseDisplay program workspaces.

This script intentionally starts with local synced folders, such as Google
Drive for Desktop. Direct Drive API support can be added later without changing
the operator workflow or program runtime pipeline.
"""

from __future__ import annotations

import argparse
import csv
import json
import os
import re
import shutil
import subprocess
import sys
import time
import zipfile
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

REPO_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_CONFIG = REPO_ROOT / "config" / "drive_intake.example.json"
STAFF_CONFIG_FILES = [
    "course_registry.csv",
    "course_notes.csv",
    "lms_watch_courses.csv",
    "lms_watch_settings.json",
]
STAFF_CONFIG_DIRS = [
    "expected_counts",
    "pdf_layout_overrides",
]

IDENTITY_FIELDS = [
    "course_name",
    "course_code",
    "course_instructors",
    "course_instructors_short",
    "program_name",
    "school_name",
    "university_name",
    "semester_label",
]


@dataclass
class DrivePaths:
    root: Path
    incoming: Path
    processing: Path
    processed: Path
    failed: Path
    reports: Path
    outputs: Path
    staff_config: Path


@dataclass
class DriveApiFolders:
    root: str
    incoming: str
    processing: str
    processed: str
    failed: str
    reports: str
    outputs: str
    staff_config: str


def api_mode(cfg: dict[str, Any]) -> bool:
    api = cfg.get("google_drive_api", {})
    return str(cfg.get("backend", "")).strip() == "google_api" or (isinstance(api, dict) and bool(api.get("enabled")))


def load_json(path: Path) -> dict[str, Any]:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as exc:
        raise SystemExit(f"Drive intake config not found: {path}") from exc
    except json.JSONDecodeError as exc:
        raise SystemExit(f"Drive intake config is not valid JSON: {path}: {exc}") from exc
    if not isinstance(data, dict):
        raise SystemExit(f"Drive intake config must be a JSON object: {path}")
    return data


def resolve_config_path(raw: str | None, program_workspace: Path | None = None) -> Path:
    if raw:
        return Path(raw).expanduser().resolve()
    if program_workspace is not None:
        candidate = program_workspace / "config" / "staff" / "drive_intake.json"
        if candidate.is_file():
            return candidate.resolve()
    return DEFAULT_CONFIG.resolve()


def program_workspace(cfg: dict[str, Any]) -> Path:
    raw = str(cfg.get("program_workspace", "")).strip()
    if not raw:
        raise SystemExit("Missing required config field: program_workspace")
    path = Path(raw).expanduser()
    if not path.is_absolute():
        path = REPO_ROOT / path
    path = path.resolve()
    if not (path / "config" / "staff").is_dir():
        raise SystemExit(f"Program workspace is missing config/staff: {path}")
    return path


def drive_paths(cfg: dict[str, Any]) -> DrivePaths:
    raw_root = str(cfg.get("drive_root", "")).strip()
    if not raw_root:
        raise SystemExit("Missing required config field: drive_root")
    root = Path(raw_root).expanduser().resolve()
    folders = cfg.get("drive_folders", {})
    if not isinstance(folders, dict):
        raise SystemExit("drive_folders must be a JSON object")

    def child(key: str, default: str) -> Path:
        return root / str(folders.get(key, default)).strip()

    return DrivePaths(
        root=root,
        incoming=child("incoming", "incoming"),
        processing=child("processing", "processing"),
        processed=child("processed", "processed"),
        failed=child("failed", "failed"),
        reports=child("reports", "reports"),
        outputs=child("outputs", "outputs"),
        staff_config=child("staff_config", "staff-config"),
    )


def api_settings(cfg: dict[str, Any]) -> dict[str, Any]:
    api = cfg.get("google_drive_api", {})
    if not isinstance(api, dict):
        raise SystemExit("google_drive_api must be a JSON object")
    folder_id = str(api.get("root_folder_id", "") or api.get("folder_id", "")).strip()
    if not folder_id:
        raise SystemExit("Missing google_drive_api.root_folder_id")
    return api


def drive_service(cfg: dict[str, Any]) -> Any:
    try:
        from googleapiclient.discovery import build
    except ImportError as exc:
        raise SystemExit(
            "Google Drive API dependencies are missing. Install requirements.txt first."
        ) from exc

    api = api_settings(cfg)
    auth_mode = str(api.get("auth_mode", "service_account")).strip() or "service_account"
    scopes = ["https://www.googleapis.com/auth/drive"]
    if auth_mode == "oauth_user":
        credentials = oauth_user_credentials(cfg, scopes)
        return build("drive", "v3", credentials=credentials, cache_discovery=False)
    if auth_mode != "service_account":
        raise SystemExit("google_drive_api.auth_mode must be 'oauth_user' or 'service_account'")

    try:
        from google.oauth2 import service_account
    except ImportError as exc:
        raise SystemExit("google-auth is missing. Install requirements.txt first.") from exc
    env_name = str(api.get("service_account_json_env", "GOOGLE_APPLICATION_CREDENTIALS")).strip()
    credentials_path = os.environ.get(env_name)
    if not credentials_path:
        local_credentials = Path(str(api.get("service_account_json_path", "")).strip()).expanduser()
        if str(local_credentials) and local_credentials.is_file():
            credentials_path = str(local_credentials)
    if not credentials_path:
        raise SystemExit(
            f"Missing Google service account credentials. Set {env_name} to the local JSON key path."
        )
    credentials = service_account.Credentials.from_service_account_file(credentials_path, scopes=scopes)
    return build("drive", "v3", credentials=credentials, cache_discovery=False)


def resolve_repo_path(raw: str) -> Path:
    path = Path(raw).expanduser()
    if not path.is_absolute():
        path = REPO_ROOT / path
    return path.resolve()


def oauth_user_credentials(cfg: dict[str, Any], scopes: list[str]) -> Any:
    try:
        from google.auth.exceptions import RefreshError
        from google.auth.transport.requests import Request
        from google.oauth2.credentials import Credentials
        from google_auth_oauthlib.flow import InstalledAppFlow
    except ImportError as exc:
        raise SystemExit(
            "OAuth user mode requires google-auth-oauthlib. Run: python -m pip install -r requirements.txt"
        ) from exc

    api = api_settings(cfg)
    token_path = resolve_repo_path(str(api.get("oauth_token_path", "config/drive_oauth_token.json")).strip())
    credentials = None
    refresh_error: RefreshError | None = None
    if token_path.is_file():
        credentials = Credentials.from_authorized_user_file(str(token_path), scopes)
    if credentials and credentials.expired and credentials.refresh_token:
        try:
            credentials.refresh(Request())
        except RefreshError as exc:
            message = str(exc)
            message_lower = message.lower()
            if "invalid_grant" not in message_lower and "expired or revoked" not in message_lower:
                raise SystemExit(f"Failed to refresh saved Google OAuth token at {token_path}: {message}") from exc
            refresh_error = exc
            credentials = None
    if not credentials or not credentials.valid:
        env_name = str(api.get("oauth_client_secret_json_env", "GOOGLE_OAUTH_CLIENT_SECRET")).strip()
        client_secret_path = os.environ.get(env_name)
        if not client_secret_path:
            configured = str(api.get("oauth_client_secret_json_path", "")).strip()
            if configured:
                client_secret_path = str(resolve_repo_path(configured))
        if not client_secret_path:
            if refresh_error is not None:
                raise SystemExit(
                    f"Saved Google OAuth token at {token_path} has expired or been revoked. "
                    f"Set {env_name} to the local OAuth client JSON path, then rerun to reauthorize."
                ) from refresh_error
            raise SystemExit(
                f"Missing OAuth client secret. Set {env_name} to the local OAuth client JSON path."
            )
        client_secret_file = Path(client_secret_path).expanduser()
        if not client_secret_file.is_file():
            raise SystemExit(
                f"OAuth client secret file not found: {client_secret_file}. "
                f"Set {env_name} to the downloaded Google OAuth desktop client JSON file."
            ) from refresh_error
        if refresh_error is not None:
            print(
                f"Saved Google OAuth token at {token_path} has expired or been revoked; starting browser reauthorization.",
                file=sys.stderr,
            )
        flow = InstalledAppFlow.from_client_secrets_file(str(client_secret_file), scopes)
        credentials = flow.run_local_server(port=0, access_type="offline", prompt="consent")
    token_path.parent.mkdir(parents=True, exist_ok=True)
    token_path.write_text(credentials.to_json(), encoding="utf-8")
    return credentials


def api_supports_all_drives(cfg: dict[str, Any]) -> bool:
    api = api_settings(cfg)
    return bool(api.get("supports_all_drives", True))


def drive_q(value: str) -> str:
    return value.replace("\\", "\\\\").replace("'", "\\'")


def api_list_children(service: Any, cfg: dict[str, Any], parent_id: str, *, name: str | None = None) -> list[dict[str, Any]]:
    query = [f"'{drive_q(parent_id)}' in parents", "trashed = false"]
    if name is not None:
        query.append(f"name = '{drive_q(name)}'")
    response = (
        service.files()
        .list(
            q=" and ".join(query),
            fields="files(id,name,mimeType,modifiedTime,size,parents)",
            supportsAllDrives=api_supports_all_drives(cfg),
            includeItemsFromAllDrives=api_supports_all_drives(cfg),
        )
        .execute()
    )
    return list(response.get("files", []))


def api_find_child(service: Any, cfg: dict[str, Any], parent_id: str, name: str) -> dict[str, Any] | None:
    matches = api_list_children(service, cfg, parent_id, name=name)
    return matches[0] if matches else None


def api_ensure_folder(service: Any, cfg: dict[str, Any], parent_id: str, name: str) -> str:
    existing = api_find_child(service, cfg, parent_id, name)
    folder_mime = "application/vnd.google-apps.folder"
    if existing:
        if existing.get("mimeType") != folder_mime:
            raise SystemExit(f"Google Drive item exists but is not a folder: {name}")
        return str(existing["id"])
    metadata = {"name": name, "mimeType": folder_mime, "parents": [parent_id]}
    created = (
        service.files()
        .create(
            body=metadata,
            fields="id",
            supportsAllDrives=api_supports_all_drives(cfg),
        )
        .execute()
    )
    return str(created["id"])


def api_folder_ids(service: Any, cfg: dict[str, Any], *, ensure: bool = True) -> DriveApiFolders:
    api = api_settings(cfg)
    root = str(api.get("root_folder_id", "") or api.get("folder_id", "")).strip()
    folders = cfg.get("drive_folders", {})
    if not isinstance(folders, dict):
        raise SystemExit("drive_folders must be a JSON object")

    def child(key: str, default: str, *, required: bool = True) -> str:
        name = str(folders.get(key, default)).strip()
        if ensure and required:
            return api_ensure_folder(service, cfg, root, name)
        existing = api_find_child(service, cfg, root, name)
        if not existing or existing.get("mimeType") != "application/vnd.google-apps.folder":
            if not required:
                return ""
            raise SystemExit(f"Google Drive folder not found under intake root: {name}")
        return str(existing["id"])

    return DriveApiFolders(
        root=root,
        incoming=child("incoming", "incoming"),
        processing=child("processing", "processing", required=False),
        processed=child("processed", "processed", required=False),
        failed=child("failed", "failed", required=False),
        reports=child("reports", "reports"),
        outputs=child("outputs", "outputs"),
        staff_config=child("staff_config", "staff-config"),
    )


def api_download_file(service: Any, cfg: dict[str, Any], file_id: str, target: Path) -> None:
    target.parent.mkdir(parents=True, exist_ok=True)
    request = service.files().get_media(fileId=file_id)
    with target.open("wb") as handle:
        downloader = None
        try:
            from googleapiclient.http import MediaIoBaseDownload
            downloader = MediaIoBaseDownload(handle, request)
            done = False
            while not done:
                _status, done = downloader.next_chunk()
        finally:
            del downloader


def api_upload_file(service: Any, cfg: dict[str, Any], parent_id: str, source: Path, name: str | None = None) -> str:
    from googleapiclient.http import MediaFileUpload

    upload_name = name or source.name
    existing = api_find_child(service, cfg, parent_id, upload_name)
    media = MediaFileUpload(str(source), resumable=True)
    if existing and existing.get("mimeType") != "application/vnd.google-apps.folder":
        updated = (
            service.files()
            .update(
                fileId=existing["id"],
                media_body=media,
                fields="id",
                supportsAllDrives=api_supports_all_drives(cfg),
            )
            .execute()
        )
        return str(updated["id"])
    metadata = {"name": upload_name, "parents": [parent_id]}
    created = (
        service.files()
        .create(
            body=metadata,
            media_body=media,
            fields="id",
            supportsAllDrives=api_supports_all_drives(cfg),
        )
        .execute()
    )
    return str(created["id"])


def api_upload_text(service: Any, cfg: dict[str, Any], parent_id: str, name: str, text: str, workspace: Path) -> str:
    tmp = workspace / "runtime" / "drive-intake" / "upload" / name
    tmp.parent.mkdir(parents=True, exist_ok=True)
    tmp.write_text(text, encoding="utf-8")
    try:
        return api_upload_file(service, cfg, parent_id, tmp, name=name)
    finally:
        if tmp.exists():
            tmp.unlink()


def api_move_file(service: Any, cfg: dict[str, Any], file_id: str, destination_parent: str) -> None:
    current = (
        service.files()
        .get(
            fileId=file_id,
            fields="parents",
            supportsAllDrives=api_supports_all_drives(cfg),
        )
        .execute()
    )
    previous_parents = ",".join(current.get("parents", []))
    service.files().update(
        fileId=file_id,
        addParents=destination_parent,
        removeParents=previous_parents,
        fields="id,parents",
        supportsAllDrives=api_supports_all_drives(cfg),
    ).execute()


def ensure_drive_dirs(paths: DrivePaths) -> None:
    for path in [
        paths.incoming,
        paths.processing,
        paths.processed,
        paths.failed,
        paths.reports,
        paths.outputs,
        paths.staff_config,
    ]:
        path.mkdir(parents=True, exist_ok=True)


def copy_file_if_newer(source: Path, target: Path) -> bool:
    if not source.is_file():
        return False
    if target.exists() and target.stat().st_mtime >= source.stat().st_mtime and target.stat().st_size == source.stat().st_size:
        return False
    target.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(source, target)
    return True


def mirror_dir(source_dir: Path, target_dir: Path) -> int:
    if not source_dir.is_dir():
        return 0
    copied = 0
    for source in sorted(source_dir.rglob("*")):
        if source.is_dir() or source.name.startswith(".") or source.name.startswith("._"):
            continue
        rel = source.relative_to(source_dir)
        if copy_file_if_newer(source, target_dir / rel):
            copied += 1
    return copied


def sync_staff_config_to_drive(workspace: Path, paths: DrivePaths) -> int:
    staff = workspace / "config" / "staff"
    copied = 0
    for name in STAFF_CONFIG_FILES:
        if copy_file_if_newer(staff / name, paths.staff_config / name):
            copied += 1
    for name in STAFF_CONFIG_DIRS:
        copied += mirror_dir(staff / name, paths.staff_config / name)
    return copied


def sync_staff_config_from_drive(workspace: Path, paths: DrivePaths) -> int:
    staff = workspace / "config" / "staff"
    copied = 0
    for name in STAFF_CONFIG_FILES:
        if copy_file_if_newer(paths.staff_config / name, staff / name):
            copied += 1
    for name in STAFF_CONFIG_DIRS:
        copied += mirror_dir(paths.staff_config / name, staff / name)
    return copied


def api_staff_config_subfolders(service: Any, cfg: dict[str, Any], folders: DriveApiFolders) -> dict[str, str]:
    return {name: api_ensure_folder(service, cfg, folders.staff_config, name) for name in STAFF_CONFIG_DIRS}


def api_sync_staff_config_to_drive(service: Any, cfg: dict[str, Any], workspace: Path, folders: DriveApiFolders) -> int:
    staff = workspace / "config" / "staff"
    copied = 0
    for name in STAFF_CONFIG_FILES:
        source = staff / name
        if source.is_file():
            api_upload_file(service, cfg, folders.staff_config, source, name=name)
            copied += 1
    subfolders = api_staff_config_subfolders(service, cfg, folders)
    for dirname, folder_id in subfolders.items():
        source_dir = staff / dirname
        if not source_dir.is_dir():
            continue
        for source in sorted(source_dir.rglob("*")):
            if source.is_dir() or source.name.startswith(".") or source.name.startswith("._"):
                continue
            rel = source.relative_to(source_dir)
            parent_id = folder_id
            for part in rel.parts[:-1]:
                parent_id = api_ensure_folder(service, cfg, parent_id, part)
            api_upload_file(service, cfg, parent_id, source, name=rel.name)
            copied += 1
    return copied


def api_sync_staff_config_from_drive(service: Any, cfg: dict[str, Any], workspace: Path, folders: DriveApiFolders) -> int:
    staff = workspace / "config" / "staff"
    copied = 0
    for name in STAFF_CONFIG_FILES:
        item = api_find_child(service, cfg, folders.staff_config, name)
        if item and item.get("mimeType") != "application/vnd.google-apps.folder":
            api_download_file(service, cfg, item["id"], staff / name)
            copied += 1
    for dirname in STAFF_CONFIG_DIRS:
        folder = api_find_child(service, cfg, folders.staff_config, dirname)
        if not folder or folder.get("mimeType") != "application/vnd.google-apps.folder":
            continue
        folder_id = str(folder["id"])
        target_dir = staff / dirname
        for item in api_list_children(service, cfg, folder_id):
            if item.get("mimeType") == "application/vnd.google-apps.folder":
                nested_dir = target_dir / item["name"]
                for nested in api_list_children(service, cfg, item["id"]):
                    if nested.get("mimeType") != "application/vnd.google-apps.folder":
                        api_download_file(service, cfg, nested["id"], nested_dir / nested["name"])
                        copied += 1
            else:
                api_download_file(service, cfg, item["id"], target_dir / item["name"])
                copied += 1
    return copied


def nonjunk_zips(path: Path) -> list[Path]:
    if not path.exists():
        return []
    return sorted(
        item
        for item in path.iterdir()
        if item.is_file()
        and item.suffix.lower() == ".zip"
        and not item.name.startswith(".")
        and not item.name.startswith("._")
    )


def is_stable(path: Path, stable_seconds: int) -> bool:
    if stable_seconds <= 0:
        return True
    age = time.time() - path.stat().st_mtime
    return age >= stable_seconds


def unique_path(target: Path) -> Path:
    if not target.exists():
        return target
    stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    return target.with_name(f"{target.stem}-{stamp}{target.suffix}")


def run_pipeline(workspace: Path, mode: str, source: Path) -> tuple[int, str]:
    if mode not in {"validate", "build-course"}:
        raise SystemExit("mode must be either 'validate' or 'build-course'")
    env = os.environ.copy()
    env["COURSEDISPLAY_WORKSPACE_ROOT"] = str(workspace)
    cmd = [sys.executable, str(REPO_ROOT / "scripts" / "runtime_pipeline.py"), mode, str(source)]
    proc = subprocess.run(
        cmd,
        cwd=workspace,
        env=env,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    return proc.returncode, proc.stdout


def course_code_from_stem(course_stem: str) -> str:
    match = re.match(r"^(\d+)", course_stem)
    return match.group(1) if match else ""


def load_course_registry(workspace: Path) -> dict[str, dict[str, str]]:
    registry_path = workspace / "config" / "staff" / "course_registry.csv"
    if not registry_path.is_file():
        return {}
    with registry_path.open(newline="", encoding="utf-8-sig") as f:
        return {
            (row.get("course_code") or "").strip(): {key: (value or "").strip() for key, value in row.items()}
            for row in csv.DictReader(f)
            if (row.get("course_code") or "").strip()
        }


def nonjunk_top_level_items(root: Path) -> list[Path]:
    return [
        item
        for item in root.iterdir()
        if item.name != "__MACOSX" and item.name != ".DS_Store" and not item.name.startswith("._")
    ]


def rewrite_identity_csv(course_dir: Path, registry_row: dict[str, str]) -> None:
    identity = course_dir / "00-meta" / "instructor_course_identity_template.csv"
    row: dict[str, str] = {}
    if identity.is_file():
        with identity.open(newline="", encoding="utf-8-sig") as f:
            rows = list(csv.DictReader(f))
            if rows:
                row = {key: (value or "") for key, value in rows[0].items()}

    fields = list(dict.fromkeys([*IDENTITY_FIELDS, *row.keys()]))
    row["course_code"] = registry_row.get("course_code", "")
    row["course_name"] = registry_row.get("course_name", "")
    identity.parent.mkdir(parents=True, exist_ok=True)
    with identity.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=fields)
        writer.writeheader()
        writer.writerow({field: row.get(field, "") for field in fields})


def zip_dir(source_dir: Path, zip_path: Path) -> None:
    zip_path.parent.mkdir(parents=True, exist_ok=True)
    if zip_path.exists():
        zip_path.unlink()
    with zipfile.ZipFile(zip_path, "w", compression=zipfile.ZIP_DEFLATED) as zf:
        for path in sorted(source_dir.rglob("*")):
            rel = path.relative_to(source_dir.parent)
            if any(part == "__MACOSX" for part in rel.parts):
                continue
            if path.name == ".DS_Store" or path.name.startswith("._"):
                continue
            if path.is_dir():
                continue
            zf.write(path, rel.as_posix())


def normalize_source_zip_for_pipeline(workspace: Path, source: Path, course_stem: str, started_at: float) -> Path:
    if not source.is_file() or source.suffix.lower() != ".zip":
        return source
    course_code = course_code_from_stem(course_stem)
    if not course_code:
        return source
    registry_row = load_course_registry(workspace).get(course_code)
    if not registry_row or not registry_row.get("course_folder_name"):
        return source

    normalized_root = workspace / "runtime" / "drive-intake" / "normalized"
    job_dir = normalized_root / f"{datetime.fromtimestamp(started_at).strftime('%Y%m%d-%H%M%S')}-{course_stem}"
    if job_dir.exists():
        shutil.rmtree(job_dir, ignore_errors=True)
    extract_dir = job_dir / "extract"
    extract_dir.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(source) as zf:
        zf.extractall(extract_dir)

    top_items = nonjunk_top_level_items(extract_dir)
    top_dirs = [item for item in top_items if item.is_dir()]
    if len(top_dirs) == 1:
        source_course_dir = top_dirs[0]
    else:
        source_course_dir = extract_dir

    course_folder_name = registry_row["course_folder_name"]
    normalized_course_dir = job_dir / course_folder_name
    shutil.copytree(
        source_course_dir,
        normalized_course_dir,
        ignore=shutil.ignore_patterns("__MACOSX", ".DS_Store", "._*"),
    )
    rewrite_identity_csv(normalized_course_dir, registry_row)
    normalized_zip = job_dir / f"{course_stem}-normalized.zip"
    zip_dir(normalized_course_dir, normalized_zip)
    return normalized_zip


def built_pdf_paths_from_output(output: str) -> list[Path]:
    paths: list[Path] = []
    for line in output.splitlines():
        match = re.search(r"Built course PDF:\s*(.+\.pdf)\s*$", line)
        if match:
            paths.append(Path(match.group(1)).expanduser())
    return paths


def recent_matching_pdfs(workspace: Path, course_stem: str, started_at: float) -> list[Path]:
    pdf_dir = workspace / "runtime" / "output" / "course-pdfs"
    if not pdf_dir.is_dir():
        return []
    course_code = course_code_from_stem(course_stem)
    matches: list[Path] = []
    for pdf in sorted(pdf_dir.glob("*.pdf")):
        if pdf.stat().st_mtime < started_at - 1:
            continue
        if pdf.stem.startswith(course_stem):
            matches.append(pdf)
        elif course_code and pdf.name.startswith(f"{course_code}-"):
            matches.append(pdf)
    return matches


def intake_output_errors(workspace: Path, mode: str, course_stem: str, started_at: float, output: str) -> list[str]:
    if mode != "build-course":
        return []
    errors: list[str] = []
    course_code = course_code_from_stem(course_stem)
    local_pdfs = recent_matching_pdfs(workspace, course_stem, started_at)
    built_pdfs = built_pdf_paths_from_output(output)

    if course_code:
        mismatched = [pdf for pdf in built_pdfs if not pdf.name.startswith(f"{course_code}-")]
        if mismatched:
            names = ", ".join(pdf.name for pdf in mismatched)
            errors.append(
                f"Source zip is named for course code {course_code}, but the pipeline built different course PDF(s): {names}."
            )

    if not local_pdfs:
        expected = f"course code {course_code}" if course_code else f"source stem {course_stem}"
        errors.append(f"No final course PDF was produced for {expected}.")
    return errors


def append_intake_errors(output: str, errors: list[str]) -> str:
    if not errors:
        return output
    lines = [
        output.rstrip(),
        "",
        "Drive intake validation failed:",
        *[f"- {error}" for error in errors],
    ]
    return "\n".join(lines).strip()


def latest_matching_reports(workspace: Path, course_stem: str, started_at: float) -> list[Path]:
    reports = workspace / "runtime" / "output" / "reports"
    if not reports.is_dir():
        return []
    candidates: list[Path] = []
    for path in reports.rglob("*"):
        if not path.is_file():
            continue
        if path.stat().st_mtime < started_at - 1:
            continue
        if course_stem in path.name or path.name.startswith("course_run_status_summary"):
            candidates.append(path)
    return sorted(candidates)


def publish_outputs(
    workspace: Path,
    paths: DrivePaths,
    course_stem: str,
    started_at: float,
    *,
    publish_reports: bool,
    publish_pdfs: bool,
) -> int:
    copied = 0
    if publish_reports:
        for report in latest_matching_reports(workspace, course_stem, started_at):
            rel = report.relative_to(workspace / "runtime" / "output" / "reports")
            if copy_file_if_newer(report, paths.reports / rel):
                copied += 1
    pdf_dir = workspace / "runtime" / "output" / "course-pdfs"
    if publish_pdfs and pdf_dir.is_dir():
        for pdf in recent_matching_pdfs(workspace, course_stem, started_at):
            if copy_file_if_newer(pdf, paths.outputs / pdf.name):
                copied += 1
    return copied


def write_run_summary(
    paths: DrivePaths,
    *,
    zip_name: str,
    status: str,
    exit_code: int,
    output: str,
    copied_outputs: int,
) -> Path:
    stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    target = paths.reports / f"drive-intake-{Path(zip_name).stem}-{stamp}.md"
    lines = [
        f"# Drive Intake Result: {zip_name}",
        "",
        f"- Generated: `{datetime.now().isoformat(timespec='seconds')}`",
        f"- Status: **{status}**",
        f"- Exit code: `{exit_code}`",
        f"- Published report/output files: {copied_outputs}",
        "",
        "## Pipeline Output",
        "",
        "```text",
        output.strip() or "(no output)",
        "```",
        "",
    ]
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text("\n".join(lines), encoding="utf-8")
    return target


def process_one(cfg: dict[str, Any]) -> int:
    workspace = program_workspace(cfg)
    paths = drive_paths(cfg)
    ensure_drive_dirs(paths)
    if cfg.get("sync_staff_config_from_drive", True):
        sync_staff_config_from_drive(workspace, paths)

    stable_seconds = int(cfg.get("stable_seconds", 30))
    ready = [path for path in nonjunk_zips(paths.incoming) if is_stable(path, stable_seconds)]
    if not ready:
        print(f"No stable zip files found in {paths.incoming}")
        return 0

    mode = str(cfg.get("mode", "build-course")).strip() or "build-course"
    source = ready[0]
    started_at = time.time()
    processing_source = unique_path(paths.processing / source.name)
    shutil.move(str(source), str(processing_source))
    local_source = workspace / "runtime" / "incoming" / processing_source.name
    local_source.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(processing_source, local_source)
    course_stem = processing_source.stem
    pipeline_source = normalize_source_zip_for_pipeline(workspace, local_source, course_stem, started_at)

    print(f"Processing {processing_source.name} for workspace {workspace.name}")
    exit_code, output = run_pipeline(workspace, mode, pipeline_source)
    intake_errors = intake_output_errors(workspace, mode, course_stem, started_at, output) if exit_code == 0 else []
    if intake_errors:
        output = append_intake_errors(output, intake_errors)
        exit_code = 1
    status = "PASS" if exit_code == 0 else "FAIL"
    copied_outputs = 0
    if exit_code == 0:
        copied_outputs = publish_outputs(
            workspace,
            paths,
            course_stem,
            started_at,
            publish_reports=bool(cfg.get("publish_reports_to_drive", True)),
            publish_pdfs=bool(cfg.get("publish_outputs_to_drive", True)),
        )
    summary = write_run_summary(
        paths,
        zip_name=processing_source.name,
        status=status,
        exit_code=exit_code,
        output=output,
        copied_outputs=copied_outputs,
    )
    destination_root = paths.processed if exit_code == 0 else paths.failed
    destination = unique_path(destination_root / processing_source.name)
    shutil.move(str(processing_source), str(destination))
    if local_source.exists():
        local_source.unlink()
    print(f"{status}: {destination.name}")
    print(f"Summary: {summary}")
    return exit_code


def process_all(cfg: dict[str, Any]) -> int:
    worst_exit = 0
    processed = 0
    while True:
        workspace = program_workspace(cfg)
        paths = drive_paths(cfg)
        stable_seconds = int(cfg.get("stable_seconds", 30))
        ready = [path for path in nonjunk_zips(paths.incoming) if is_stable(path, stable_seconds)]
        if not ready:
            break
        exit_code = process_one(cfg)
        processed += 1
        worst_exit = max(worst_exit, exit_code)
    print(f"Processed {processed} stable Drive intake zip file(s).")
    return worst_exit


def api_ready_zip_files(service: Any, cfg: dict[str, Any], folders: DriveApiFolders) -> list[dict[str, Any]]:
    files = [
        item
        for item in api_list_children(service, cfg, folders.incoming)
        if item.get("mimeType") != "application/vnd.google-apps.folder"
        and str(item.get("name", "")).lower().endswith(".zip")
        and not str(item.get("name", "")).startswith(".")
        and not str(item.get("name", "")).startswith("._")
    ]
    return sorted(files, key=lambda item: item.get("name", ""))


def api_processed_state_path(workspace: Path) -> Path:
    return workspace / "runtime" / "drive-intake" / "processed-drive-files.json"


def load_api_processed_state(workspace: Path) -> dict[str, Any]:
    path = api_processed_state_path(workspace)
    if not path.is_file():
        return {"processed": {}}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return {"processed": {}}
    if not isinstance(data, dict):
        return {"processed": {}}
    processed = data.setdefault("processed", {})
    if not isinstance(processed, dict):
        data["processed"] = {}
    return data


def save_api_processed_state(workspace: Path, state: dict[str, Any]) -> None:
    path = api_processed_state_path(workspace)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(state, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def api_file_run_key(item: dict[str, Any]) -> str:
    return "|".join(
        [
            str(item.get("id", "")),
            str(item.get("modifiedTime", "")),
            str(item.get("size", "")),
        ]
    )


def api_unprocessed_zip_files(workspace: Path, files: list[dict[str, Any]]) -> list[dict[str, Any]]:
    processed = load_api_processed_state(workspace).get("processed", {})
    return [item for item in files if api_file_run_key(item) not in processed]


def mark_api_file_processed(
    workspace: Path,
    item: dict[str, Any],
    *,
    status: str,
    exit_code: int,
    local_source: Path,
    summary: Path,
) -> None:
    state = load_api_processed_state(workspace)
    processed = state.setdefault("processed", {})
    processed[api_file_run_key(item)] = {
        "id": item.get("id", ""),
        "name": item.get("name", ""),
        "modifiedTime": item.get("modifiedTime", ""),
        "size": item.get("size", ""),
        "processed_at": datetime.now().isoformat(timespec="seconds"),
        "status": status,
        "exit_code": exit_code,
        "local_source": str(local_source),
        "summary": str(summary),
    }
    save_api_processed_state(workspace, state)


def api_item_is_stable(item: dict[str, Any], stable_seconds: int) -> bool:
    if stable_seconds <= 0:
        return True
    raw = str(item.get("modifiedTime", "")).strip()
    if not raw:
        return True
    try:
        modified = datetime.fromisoformat(raw.replace("Z", "+00:00"))
    except ValueError:
        return True
    return (datetime.now(timezone.utc) - modified).total_seconds() >= stable_seconds


def api_publish_outputs(
    service: Any,
    cfg: dict[str, Any],
    workspace: Path,
    folders: DriveApiFolders,
    course_stem: str,
    started_at: float,
) -> int:
    copied = 0
    if cfg.get("publish_reports_to_drive", True) and folders.reports:
        for report in latest_matching_reports(workspace, course_stem, started_at):
            api_upload_file(service, cfg, folders.reports, report, name=report.name)
            copied += 1
    pdf_dir = workspace / "runtime" / "output" / "course-pdfs"
    if cfg.get("publish_outputs_to_drive", True) and folders.outputs and pdf_dir.is_dir():
        for pdf in recent_matching_pdfs(workspace, course_stem, started_at):
            api_upload_file(service, cfg, folders.outputs, pdf, name=pdf.name)
            copied += 1
    return copied


def api_process_one(cfg: dict[str, Any]) -> int:
    workspace = program_workspace(cfg)
    service = drive_service(cfg)
    folders = api_folder_ids(service, cfg, ensure=False)
    if cfg.get("sync_staff_config_from_drive", True):
        api_sync_staff_config_from_drive(service, cfg, workspace, folders)

    stable_seconds = int(cfg.get("stable_seconds", 30))
    ready = api_unprocessed_zip_files(
        workspace,
        [item for item in api_ready_zip_files(service, cfg, folders) if api_item_is_stable(item, stable_seconds)],
    )
    if not ready:
        print("No unprocessed zip files found in Google Drive incoming folder.")
        return 0

    mode = str(cfg.get("mode", "build-course")).strip() or "build-course"
    item = ready[0]
    started_at = time.time()
    local_source = workspace / "runtime" / "drive-intake" / "incoming" / item["name"]
    api_download_file(service, cfg, item["id"], local_source)
    course_stem = Path(item["name"]).stem
    pipeline_source = normalize_source_zip_for_pipeline(workspace, local_source, course_stem, started_at)

    print(f"Processing {item['name']} downloaded from Google Drive for workspace {workspace.name}")
    exit_code, output = run_pipeline(workspace, mode, pipeline_source)
    intake_errors = intake_output_errors(workspace, mode, course_stem, started_at, output) if exit_code == 0 else []
    if intake_errors:
        output = append_intake_errors(output, intake_errors)
        exit_code = 1
    status = "PASS" if exit_code == 0 else "FAIL"
    copied_outputs = 0
    if exit_code == 0:
        copied_outputs = api_publish_outputs(service, cfg, workspace, folders, course_stem, started_at)
    local_reports = latest_matching_reports(workspace, course_stem, started_at)
    pdf_dir = workspace / "runtime" / "output" / "course-pdfs"
    local_pdfs = recent_matching_pdfs(workspace, course_stem, started_at) if pdf_dir.is_dir() else []
    summary_text = "\n".join(
        [
            f"# Drive Intake Result: {item['name']}",
            "",
            f"- Generated: `{datetime.now().isoformat(timespec='seconds')}`",
            f"- Status: **{status}**",
            f"- Exit code: `{exit_code}`",
            "- Drive source zip handling: **download-only**",
            f"- Downloaded source: `{local_source}`",
            f"- Published final report/PDF files to Drive: {copied_outputs}",
            f"- Local report files: {len(local_reports)}",
            f"- Local PDF files: {len(local_pdfs)}",
            "",
            "## Pipeline Output",
            "",
            "```text",
            output.strip() or "(no output)",
            "```",
            "",
        ]
    )
    stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    summary = workspace / "runtime" / "output" / "reports" / "drive-intake" / f"drive-intake-{course_stem}-{stamp}.md"
    summary.parent.mkdir(parents=True, exist_ok=True)
    summary.write_text(summary_text, encoding="utf-8")
    if exit_code == 0 and cfg.get("publish_reports_to_drive", True) and folders.reports:
        api_upload_file(service, cfg, folders.reports, summary, name=summary.name)
        copied_outputs += 1
    mark_api_file_processed(
        workspace,
        item,
        status=status,
        exit_code=exit_code,
        local_source=local_source,
        summary=summary,
    )
    print(f"{status}: {item['name']}")
    print(f"Summary: {summary}")
    if exit_code == 0:
        print(f"Published {copied_outputs} final report/PDF file(s) to Google Drive.")
    print("Google Drive source zips were download-only; no zip files were uploaded or moved in Drive.")
    return exit_code


def api_process_all(cfg: dict[str, Any]) -> int:
    worst_exit = 0
    processed = 0
    while True:
        workspace = program_workspace(cfg)
        service = drive_service(cfg)
        folders = api_folder_ids(service, cfg, ensure=False)
        stable_seconds = int(cfg.get("stable_seconds", 30))
        ready = api_unprocessed_zip_files(
            workspace,
            [item for item in api_ready_zip_files(service, cfg, folders) if api_item_is_stable(item, stable_seconds)],
        )
        if not ready:
            break
        exit_code = api_process_one(cfg)
        processed += 1
        worst_exit = max(worst_exit, exit_code)
    print(f"Processed {processed} Google Drive intake zip file(s).")
    return worst_exit


def init_drive(cfg: dict[str, Any]) -> int:
    if api_mode(cfg):
        workspace = program_workspace(cfg)
        service = drive_service(cfg)
        folders = api_folder_ids(service, cfg)
        copied = api_sync_staff_config_to_drive(service, cfg, workspace, folders)
        api_upload_text(
            service,
            cfg,
            folders.root,
            "README_CourseDisplay_Intake.txt",
            "CourseDisplay Drive Intake\n\n"
            "Staff update CSV/JSON files under staff-config/.\n"
            "Staff place new instructor course zip files under incoming/.\n"
            "CourseDisplay downloads source zips locally and uploads final reports/PDFs back to Drive.\n"
            "Do not place credentials, passwords, browser profiles, or LMS session files here.\n",
            workspace,
        )
        print("Initialized Google Drive API intake folders.")
        print(f"Published {copied} staff config file(s).")
        return 0

    workspace = program_workspace(cfg)
    paths = drive_paths(cfg)
    ensure_drive_dirs(paths)
    copied = sync_staff_config_to_drive(workspace, paths)
    readme = paths.root / "README_CourseDisplay_Intake.txt"
    if not readme.exists():
        readme.write_text(
            "CourseDisplay Drive Intake\n\n"
            "Staff update CSV/JSON files under staff-config/.\n"
            "Staff place new instructor course zip files under incoming/.\n"
            "CourseDisplay processes source zips locally and writes final reports/PDFs back to Drive.\n"
            "Do not place credentials, passwords, browser profiles, or LMS session files here.\n",
            encoding="utf-8",
        )
    print(f"Initialized Drive intake folders at {paths.root}")
    print(f"Published {copied} staff config file(s).")
    return 0


def write_program_config(args: argparse.Namespace) -> int:
    workspace = Path(args.program_workspace).expanduser()
    if not workspace.is_absolute():
        workspace = REPO_ROOT / workspace
    workspace = workspace.resolve()
    target = workspace / "config" / "staff" / "drive_intake.json"
    cfg = load_json(DEFAULT_CONFIG)
    cfg["enabled"] = True
    try:
        cfg["program_workspace"] = str(workspace.relative_to(REPO_ROOT))
    except ValueError:
        cfg["program_workspace"] = str(workspace)
    if args.google_folder_id:
        cfg["backend"] = "google_api"
        cfg["drive_root"] = ""
        cfg["google_drive_api"]["enabled"] = True
        cfg["google_drive_api"]["auth_mode"] = args.auth_mode
        cfg["google_drive_api"]["root_folder_id"] = args.google_folder_id
    else:
        cfg["backend"] = "local_sync"
        cfg["drive_root"] = str(Path(args.drive_root).expanduser())
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(json.dumps(cfg, indent=2) + "\n", encoding="utf-8")
    print(f"Wrote {target}")
    return 0


def watch(cfg: dict[str, Any]) -> int:
    interval = int(cfg.get("poll_seconds", 60))
    print(f"Watching Drive intake every {interval}s. Press Ctrl+C to stop.")
    while True:
        if api_mode(cfg):
            api_process_one(cfg)
        else:
            process_one(cfg)
        time.sleep(interval)


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Coordinate Google Drive backed CourseDisplay intake.")
    parser.add_argument("--config", help="Path to drive_intake.json. Defaults to program config when --program-workspace is supplied.")
    parser.add_argument("--program-workspace", help="Program workspace, e.g. programs/ice-ay2025")
    subparsers = parser.add_subparsers(dest="command", required=True)

    setup = subparsers.add_parser("write-config", help="Write config/staff/drive_intake.json for a program.")
    setup.add_argument("--program-workspace", required=True, help="Program workspace, e.g. programs/ice-ay2025")
    setup.add_argument("--drive-root", help="Local synced Google Drive intake folder.")
    setup.add_argument("--google-folder-id", help="Google Drive API root folder ID for cloud-only intake.")
    setup.add_argument(
        "--auth-mode",
        choices=["oauth_user", "service_account"],
        default="oauth_user",
        help="Google API auth mode. Use oauth_user for normal My Drive folders; service_account for Shared Drives.",
    )

    subparsers.add_parser("init-drive", help="Create Drive folders and publish current staff config.")
    subparsers.add_parser("sync-config-to-drive", help="Publish config/staff files to Drive staff-config.")
    subparsers.add_parser("sync-config-from-drive", help="Pull Drive staff-config files into program config/staff.")
    subparsers.add_parser("process-once", help="Process one stable zip from Drive incoming.")
    subparsers.add_parser("process-all", help="Process every stable zip from Drive incoming, one at a time.")
    subparsers.add_parser("watch", help="Continuously poll Drive incoming.")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    if args.command == "write-config":
        if not args.drive_root and not args.google_folder_id:
            raise SystemExit("write-config requires either --drive-root or --google-folder-id")
        return write_program_config(args)

    program = Path(args.program_workspace).expanduser().resolve() if args.program_workspace else None
    cfg = load_json(resolve_config_path(args.config, program))
    workspace = program_workspace(cfg)

    if api_mode(cfg):
        service = drive_service(cfg)
        if args.command == "init-drive":
            return init_drive(cfg)
        if args.command == "sync-config-to-drive":
            folders = api_folder_ids(service, cfg)
            copied = api_sync_staff_config_to_drive(service, cfg, workspace, folders)
            print(f"Published {copied} staff config file(s) to Google Drive staff-config.")
            return 0
        if args.command == "sync-config-from-drive":
            folders = api_folder_ids(service, cfg, ensure=False)
            copied = api_sync_staff_config_from_drive(service, cfg, workspace, folders)
            print(f"Pulled {copied} staff config file(s) from Google Drive staff-config.")
            return 0
        if args.command == "process-once":
            return api_process_one(cfg)
        if args.command == "process-all":
            return api_process_all(cfg)
        if args.command == "watch":
            return watch(cfg)

    paths = drive_paths(cfg)
    ensure_drive_dirs(paths)

    if args.command == "init-drive":
        return init_drive(cfg)
    if args.command == "sync-config-to-drive":
        copied = sync_staff_config_to_drive(workspace, paths)
        print(f"Published {copied} staff config file(s) to {paths.staff_config}")
        return 0
    if args.command == "sync-config-from-drive":
        copied = sync_staff_config_from_drive(workspace, paths)
        print(f"Pulled {copied} staff config file(s) from {paths.staff_config}")
        return 0
    if args.command == "process-once":
        return process_one(cfg)
    if args.command == "process-all":
        return process_all(cfg)
    if args.command == "watch":
        return watch(cfg)
    raise SystemExit(f"Unsupported command: {args.command}")


if __name__ == "__main__":
    raise SystemExit(main())
