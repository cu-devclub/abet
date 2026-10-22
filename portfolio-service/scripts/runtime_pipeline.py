#!/usr/bin/env python3
"""Runtime pipeline for validating instructor submissions and building course PDFs."""

from __future__ import annotations

import argparse
import csv
import json
import logging
import os
import re
import shutil
import stat
import subprocess
import sys
import zipfile
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Iterable

SHARED_ROOT = Path(__file__).resolve().parents[1]


def detect_workspace_root() -> Path:
    env_root = os.environ.get("COURSEDISPLAY_WORKSPACE_ROOT")
    if env_root:
        return Path(env_root).resolve()
    cwd = Path.cwd()
    if (cwd / "config" / "staff" / "course_registry.csv").is_file():
        return cwd.resolve()
    if (cwd / "config" / "course_registry.csv").is_file():
        return cwd.resolve()
    return SHARED_ROOT


WORKSPACE_ROOT = detect_workspace_root()
os.environ.setdefault("COURSEDISPLAY_WORKSPACE_ROOT", str(WORKSPACE_ROOT))

from build_all_pdfs import write_expected_counts
from course_completion import build_detailed_report, build_pev_report, build_report, build_summary_report, compute_results, load_course_notes
from expected_counts_manager import sync_expected_counts
from pdf_layout_manager import ensure_course_override_csv, sync_pdf_layout_overrides
from create_program_workspace import backfill_workspace
from generate_course_grading_from_csv import build_tex as build_grading_tex
from generate_course_grading_from_csv import parse_rows as parse_grading_rows
from generate_course_meta_from_csv import build_meta_tex, load_rows as load_meta_rows
from generate_course_schedule_from_csv import build_schedule_tex, parse_rows as parse_schedule_rows
from generate_syllabus_abet_from_csv import build_tex as build_syllabus_tex
from generate_syllabus_abet_from_csv import load_rows as load_syllabus_rows
from generate_course_main_tex import render_main_tex, sync_main_tex

RUNTIME = WORKSPACE_ROOT / "runtime"
RUNTIME_INCOMING = RUNTIME / "incoming"
RUNTIME_PROCESSED = RUNTIME / "processed"
RUNTIME_PROCESSED_REPAIRED = RUNTIME / "processed-repaired"
RUNTIME_FAILED = RUNTIME / "failed"
RUNTIME_STAGING = RUNTIME / "staging"
RUNTIME_REPAIR = RUNTIME / "repair"
RUNTIME_WORKSPACES = RUNTIME / "workspaces"
RUNTIME_OUTPUT = RUNTIME / "output"
RUNTIME_REPORTS = RUNTIME_OUTPUT / "reports"
RUNTIME_PDFS = RUNTIME_OUTPUT / "course-pdfs"
RUNTIME_LOGS = RUNTIME / "logs"
ASSETS_DEFAULT = SHARED_ROOT / "assets" / "default"
PROGRAM_CONFIG_ASSETS = WORKSPACE_ROOT / "config" / "assets"
REGISTRY_PATH = WORKSPACE_ROOT / "config" / "staff" / "course_registry.csv"
EXPECTED_COUNTS_JSON = WORKSPACE_ROOT / "config" / "generated" / "expected_counts.json"
PDF_LAYOUT_JSON = WORKSPACE_ROOT / "config" / "generated" / "pdf_layout_overrides.json"
PDF_LAYOUT_TEX = WORKSPACE_ROOT / "config" / "generated" / "pdf_layout_overrides.tex"
PDF_LAYOUT_OVERRIDES_DIR = WORKSPACE_ROOT / "config" / "staff" / "pdf_layout_overrides"
COURSE_NOTES_CSV = WORKSPACE_ROOT / "config" / "staff" / "course_notes.csv"
COURSE_NOTES_JSON = WORKSPACE_ROOT / "config" / "staff" / "course_notes.json"
TRACKED_MAIN_ROOT = WORKSPACE_ROOT / "tracked" / "course-main-tex"
MAIN_TEMPLATE_PATH = SHARED_ROOT / "templates" / "course-main-template.tex"

REQUIRED_SUBDIRS = [
    "00-meta",
    "01-syllabus",
    "02-textbook",
    "03-schedule",
    "10-lectures",
    "20-assignments",
    "30-quizzes",
    "40-exams",
    "90-grades",
]
REQUIRED_FILES = [
    "00-meta/instructor_course_identity_template.csv",
    "01-syllabus/instructor_syllabus_abet_reference.csv",
    "03-schedule/instructor_course_schedule_template.csv",
    "90-grades/instructor_course_grading_template.csv",
]
REQUIRED_FILE_SUBDIRS = {relpath.split("/", 1)[0] for relpath in REQUIRED_FILES}
ARTIFACT_ONLY_SUBDIRS = [subdir for subdir in REQUIRED_SUBDIRS if subdir not in REQUIRED_FILE_SUBDIRS]

ALLOWED_ROOT_FILES = {
    "README_FIRST.txt",
    "INSTRUCTOR_DETAILED_SUBMISSION_GUIDE.md",
    "INSTRUCTOR_DETAILED_SUBMISSION_GUIDE.pdf",
}

ALLOWED_FILE_PATTERNS = {
    "00-meta": [
        re.compile(r"^course-meta\.tex$"),
        re.compile(r"^instructor_course_identity_template\.csv$"),
    ],
    "01-syllabus": [
        re.compile(r"^instructor_syllabus_abet_reference\.csv$"),
        re.compile(r"^syllabus-abet\.tex$"),
        re.compile(r"^syllabus-instructor\.pdf$"),
    ],
    "02-textbook": [
        re.compile(r"^textbook-part\d{2}\.pdf$"),
    ],
    "03-schedule": [
        re.compile(r"^course-schedule\.tex$"),
        re.compile(r"^instructor_course_schedule_template\.csv$"),
        re.compile(r"^lecture_schedule_mapping\.csv$"),
    ],
    "10-lectures": [
        re.compile(r"^week\d{2}-part\d{2}\.pdf$"),
    ],
    "20-assignments": [
        re.compile(r"^asg-all-scores\.pdf$"),
        re.compile(r"^asg\d{2}-(average|high|instructions|low|scores)\.pdf$"),
    ],
    "30-quizzes": [
        re.compile(r"^quiz-all-scores\.pdf$"),
        re.compile(r"^quiz\d{2}-(average|high|instructions|low|scores)\.pdf$"),
    ],
    "40-exams": [
        re.compile(r"^(midterm|final)-(average|high|instructions|low|scores)\.pdf$"),
    ],
    "90-grades": [
        re.compile(r"^course-grading\.tex$"),
        re.compile(r"^instructor_course_grading_template\.csv$"),
        re.compile(r"^overall-grades\.pdf$"),
    ],
}


@dataclass
class RegistryEntry:
    course_code: str
    course_name: str
    semester_workspace: str
    course_folder_name: str


@dataclass
class ValidationResult:
    course_code: str
    course_folder_name: str
    semester_workspace: str
    source_path: str
    staged_path: str
    main_tex_path: str
    generated_files: list[str]
    inspected_file_count: int
    warnings: list[str]
    errors: list[str]


@dataclass
class BatchValidationResult:
    source_path: str
    course_folder_name: str
    report_md: str
    report_json: str
    warning_count: int
    error_count: int
    status: str


@dataclass
class RepairResult:
    source_path: str
    original_source_path: str
    course_folder_name: str
    repaired_path: str
    removed_items: list[str]
    removed_backup_path: str
    remaining_warning_count: int
    remaining_error_count: int
    report_md: str
    report_json: str


COURSE_RUN_STATUS_JSON = RUNTIME_REPORTS / "course_run_status_summary.json"
COURSE_RUN_STATUS_MD = RUNTIME_REPORTS / "course_run_status_summary.md"


def ensure_runtime_dirs() -> None:
    for path in [
        RUNTIME_INCOMING,
        RUNTIME_PROCESSED,
        RUNTIME_PROCESSED_REPAIRED,
        RUNTIME_FAILED,
        RUNTIME_STAGING,
        RUNTIME_REPAIR,
        RUNTIME_WORKSPACES,
        RUNTIME_OUTPUT,
        RUNTIME_REPORTS,
        RUNTIME_PDFS,
        RUNTIME_LOGS,
    ]:
        path.mkdir(parents=True, exist_ok=True)


def is_archive_junk_name(name: str) -> bool:
    return name == ".DS_Store" or name.startswith("._") or name.lower() == "__macosx"


def make_path_writable(path: Path) -> None:
    paths = [path]
    if path.is_dir():
        paths.extend(path.rglob("*"))
    for item in paths:
        try:
            mode = item.stat().st_mode
            write_bits = stat.S_IWUSR
            if item.is_dir():
                write_bits |= stat.S_IXUSR
            item.chmod(mode | write_bits)
        except OSError:
            pass


def remove_archive_junk(path: Path) -> bool:
    if not path.exists():
        return False
    try:
        make_path_writable(path)
        if path.is_dir():
            shutil.rmtree(path)
        else:
            make_path_writable(path.parent)
            path.unlink()
        return True
    except OSError:
        return False


def strip_junk_files(*roots: Path) -> list[Path]:
    removed: list[Path] = []
    for root in roots:
        if not root.exists():
            continue
        junk_dirs = sorted(
            (path for path in root.rglob("*") if path.is_dir() and path.name.lower() == "__macosx"),
            key=lambda path: len(path.parts),
        )
        for path in junk_dirs:
            if remove_archive_junk(path):
                removed.append(path)
        for path in sorted(root.rglob("*"), reverse=True):
            if not is_archive_junk_name(path.name) or not path.exists():
                continue
            if remove_archive_junk(path):
                removed.append(path)
    return removed


def nonjunk_top_level_items(root: Path) -> list[Path]:
    return sorted(item for item in root.iterdir() if not is_archive_junk_name(item.name))


def has_required_course_layout(root: Path) -> bool:
    return all((root / subdir).is_dir() for subdir in REQUIRED_SUBDIRS)


def looks_like_course_layout(root: Path) -> bool:
    return any((root / subdir).is_dir() for subdir in REQUIRED_SUBDIRS) or any(
        (root / relpath).is_file() for relpath in REQUIRED_FILES
    )


def find_wrapped_course_layout(root: Path) -> Path:
    current = root
    for _depth in range(5):
        strip_junk_files(current)
        if has_required_course_layout(current) or looks_like_course_layout(current):
            return current
        items = nonjunk_top_level_items(current)
        dirs = [item for item in items if item.is_dir()]
        if len(items) != 1 or len(dirs) != 1:
            return root
        current = dirs[0]
    return root


def ensure_artifact_only_subdirs(course_dir: Path) -> None:
    if not looks_like_course_layout(course_dir):
        return
    for subdir in ARTIFACT_ONLY_SUBDIRS:
        (course_dir / subdir).mkdir(parents=True, exist_ok=True)


def normalize_wrapped_course_layout(course_dir: Path) -> Path:
    """Collapse benign zip wrapper folders so required course folders are at root."""
    strip_junk_files(course_dir)
    wrapped_layout = find_wrapped_course_layout(course_dir)
    if wrapped_layout == course_dir:
        ensure_artifact_only_subdirs(course_dir)
        return course_dir

    temp_dir = course_dir.parent / f".{course_dir.name}-wrapped-layout"
    if temp_dir.exists():
        shutil.rmtree(temp_dir, ignore_errors=True)
    shutil.move(str(wrapped_layout), str(temp_dir))
    if course_dir.exists():
        shutil.rmtree(course_dir, ignore_errors=True)
    course_dir.mkdir(parents=True, exist_ok=True)
    for item in nonjunk_top_level_items(temp_dir):
        dest_item = course_dir / item.name
        if dest_item.exists():
            if dest_item.is_dir():
                shutil.rmtree(dest_item, ignore_errors=True)
            else:
                dest_item.unlink(missing_ok=True)
        shutil.move(str(item), str(dest_item))
    shutil.rmtree(temp_dir, ignore_errors=True)
    strip_junk_files(course_dir)
    ensure_artifact_only_subdirs(course_dir)
    return course_dir


def expected_registry_keys(registry: dict[str, RegistryEntry]) -> set[tuple[str, str]]:
    return {
        (entry.semester_workspace, entry.course_folder_name)
        for entry in registry.values()
    }


def prune_runtime_workspaces(registry: dict[str, RegistryEntry]) -> list[Path]:
    keep = expected_registry_keys(registry)
    removed: list[Path] = []
    if not RUNTIME_WORKSPACES.exists():
        return removed

    for semester_dir in sorted(p for p in RUNTIME_WORKSPACES.iterdir() if p.is_dir() and not p.name.startswith(".")):
        main_dir = semester_dir / "Course-main-tex"
        course_docs_dir = semester_dir / "course_documents"

        for main_path in sorted(main_dir.glob("*-main.tex")) if main_dir.exists() else []:
            course_folder_name = main_path.stem.removesuffix("-main")
            if (semester_dir.name, course_folder_name) not in keep:
                main_path.unlink()
                removed.append(main_path)
                compiled_pdf = main_dir / "Compiled pdf" / f"{main_path.stem}.pdf"
                if compiled_pdf.exists():
                    compiled_pdf.unlink()
                    removed.append(compiled_pdf)

        for course_dir in sorted(p for p in course_docs_dir.iterdir() if p.is_dir()) if course_docs_dir.exists() else []:
            if (semester_dir.name, course_dir.name) not in keep:
                shutil.rmtree(course_dir)
                removed.append(course_dir)

        if semester_dir.exists():
            # Drop empty compiled output folders once stale course files are removed.
            for subdir in sorted((p for p in semester_dir.rglob("*") if p.is_dir()), reverse=True):
                if subdir.name.startswith("."):
                    continue
                try:
                    if not any(subdir.iterdir()):
                        subdir.rmdir()
                except OSError:
                    pass

    return removed


def prune_runtime_outputs(registry: dict[str, RegistryEntry]) -> list[Path]:
    keep_pdf_names = {f"{entry.course_folder_name}-main.pdf" for entry in registry.values()}
    keep_report_prefixes = {entry.course_folder_name for entry in registry.values()}
    removed: list[Path] = []

    if RUNTIME_PDFS.exists():
        for pdf_path in sorted(RUNTIME_PDFS.glob("*-main.pdf")):
            if pdf_path.name not in keep_pdf_names:
                pdf_path.unlink()
                removed.append(pdf_path)

    if RUNTIME_REPORTS.exists():
        for report_path in sorted(RUNTIME_REPORTS.glob("*-validation.*")):
            prefix = report_path.name.split("-validation.", 1)[0]
            if prefix not in keep_report_prefixes:
                report_path.unlink()
                removed.append(report_path)

    return removed


def refresh_runtime_workspace_configs() -> list[Path]:
    updated: list[Path] = []
    if not RUNTIME_WORKSPACES.exists():
        return updated
    for semester_dir in sorted(p for p in RUNTIME_WORKSPACES.iterdir() if p.is_dir() and not p.name.startswith(".")):
        config_dir = semester_dir / "config"
        main_dir = semester_dir / "Course-main-tex"
        config_dir.mkdir(parents=True, exist_ok=True)
        main_dir.mkdir(parents=True, exist_ok=True)
        shutil.copy2(ASSETS_DEFAULT / "abet-binder.sty", semester_dir / "abet-binder.sty")
        shutil.copy2(ASSETS_DEFAULT / "abet-binder.sty", main_dir / "abet-binder.sty")
        write_expected_counts(EXPECTED_COUNTS_JSON, config_dir / "expected_counts.tex")
        shutil.copy2(EXPECTED_COUNTS_JSON, config_dir / "expected_counts.json")
        shutil.copy2(PDF_LAYOUT_JSON, config_dir / "pdf_layout_overrides.json")
        shutil.copy2(PDF_LAYOUT_TEX, config_dir / "pdf_layout_overrides.tex")
        updated.extend([
            semester_dir / "abet-binder.sty",
            main_dir / "abet-binder.sty",
            config_dir / "expected_counts.tex",
            config_dir / "expected_counts.json",
            config_dir / "pdf_layout_overrides.json",
            config_dir / "pdf_layout_overrides.tex",
        ])
    return updated


def ensure_runtime_workspace_pdf_override_sheets() -> list[Path]:
    written: list[Path] = []
    if not RUNTIME_WORKSPACES.exists():
        return written
    for semester_dir in sorted(p for p in RUNTIME_WORKSPACES.iterdir() if p.is_dir() and not p.name.startswith(".")):
        course_docs_dir = semester_dir / "course_documents"
        if not course_docs_dir.is_dir():
            continue
        for course_dir in sorted(p for p in course_docs_dir.iterdir() if p.is_dir() and not p.name.startswith(".")):
            written.append(ensure_course_override_csv(course_dir, PDF_LAYOUT_OVERRIDES_DIR))
    return written


def sync_registry_state() -> dict[str, list[Path]]:
    registry = load_registry()
    sync_expected_counts()
    written_pdf_override_sheets = ensure_runtime_workspace_pdf_override_sheets()
    sync_pdf_layout_overrides(overrides_dir=PDF_LAYOUT_OVERRIDES_DIR)
    removed_junk = strip_junk_files(WORKSPACE_ROOT / "runtime", TRACKED_MAIN_ROOT, SHARED_ROOT / "templates")
    template_text = MAIN_TEMPLATE_PATH.read_text(encoding="utf-8")
    written, removed_main = sync_main_tex(
        list(registry.values()),
        template_text=template_text,
        output_root=TRACKED_MAIN_ROOT,
        prune=True,
    )
    removed_runtime = prune_runtime_workspaces(registry)
    removed_outputs = prune_runtime_outputs(registry)
    refreshed_configs = refresh_runtime_workspace_configs()
    return {
        "written_main_tex": written,
        "removed_tracked_main_tex": removed_main,
        "removed_runtime_items": removed_runtime,
        "removed_output_items": removed_outputs,
        "removed_junk_files": removed_junk,
        "written_pdf_override_sheets": written_pdf_override_sheets,
        "refreshed_runtime_configs": refreshed_configs,
    }


def load_registry(path: Path = REGISTRY_PATH) -> dict[str, RegistryEntry]:
    with path.open(newline="", encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        required = {"course_code", "course_name", "semester_workspace", "course_folder_name"}
        if reader.fieldnames is None:
            raise SystemExit("Registry CSV has no header row.")
        missing = required - set(reader.fieldnames)
        if missing:
            raise SystemExit(f"Registry missing required columns: {', '.join(sorted(missing))}")
        out: dict[str, RegistryEntry] = {}
        for row in reader:
            code = (row.get("course_code") or "").strip()
            if not code:
                continue
            out[code] = RegistryEntry(
                course_code=code,
                course_name=(row.get("course_name") or "").strip(),
                semester_workspace=(row.get("semester_workspace") or "").strip(),
                course_folder_name=(row.get("course_folder_name") or "").strip(),
            )
        return out


def auto_register_course(
    course_code: str,
    course_name: str = "",
    semester_workspace: str = "",
    course_folder_name: str = "",
    registry_path: Path = REGISTRY_PATH,
) -> RegistryEntry:
    registry = load_registry(registry_path)
    if course_code in registry:
        return registry[course_code]

    if not course_folder_name:
        if course_code and course_name:
            normalized = "-".join(course_name.split())
            course_folder_name = f"{course_code}-{normalized}"
        else:
            course_folder_name = course_code

    if not course_name:
        course_name = course_folder_name

    if not semester_workspace:
        semester_workspace = os.getenv("DEFAULT_SEMESTER_WORKSPACE", "aero-ay2025")

    entry = RegistryEntry(
        course_code=course_code,
        course_name=course_name,
        semester_workspace=semester_workspace,
        course_folder_name=course_folder_name,
    )

    registry_path.parent.mkdir(parents=True, exist_ok=True)
    file_exists = registry_path.is_file()
    with registry_path.open("a", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        if not file_exists or registry_path.stat().st_size == 0:
            writer.writerow(["course_code", "course_name", "semester_workspace", "course_folder_name"])
        writer.writerow([entry.course_code, entry.course_name, entry.semester_workspace, entry.course_folder_name])

    logging.info(
        f"Auto-registered missing course into {registry_path.name}: "
        f"course_code='{entry.course_code}', course_name='{entry.course_name}', "
        f"semester_workspace='{entry.semester_workspace}', course_folder_name='{entry.course_folder_name}'"
    )

    try:
        from expected_counts_manager import sync_expected_counts
        sync_expected_counts()
    except Exception as exc:
        logging.warning(f"Failed to sync expected counts after auto-registration: {exc}")

    return entry


def move_file_if_exists(source: Path, target: Path, moved: list[Path], conflicts: list[str]) -> None:
    if not source.exists():
        return
    if source.resolve() == target.resolve():
        return
    target.parent.mkdir(parents=True, exist_ok=True)
    if target.exists():
        conflicts.append(f"Target already exists: {target}")
        return
    shutil.move(str(source), str(target))
    moved.append(target)


def move_dir_if_exists(source: Path, target: Path, moved: list[Path], conflicts: list[str]) -> None:
    if not source.exists():
        return
    if source.resolve() == target.resolve():
        return
    target.parent.mkdir(parents=True, exist_ok=True)
    if target.exists():
        conflicts.append(f"Target already exists: {target}")
        return
    shutil.move(str(source), str(target))
    moved.append(target)


def rewrite_course_keyed_csv(
    path: Path,
    *,
    old_entry: RegistryEntry,
    new_entry: RegistryEntry,
    columns: list[str],
) -> bool:
    if not path.is_file():
        return False
    updated_rows: list[dict[str, str]] = []
    changed = False
    with path.open(newline="", encoding="utf-8-sig") as handle:
        reader = csv.DictReader(handle)
        fieldnames = reader.fieldnames or columns
        for row in reader:
            normalized = {column: (row.get(column) or "") for column in fieldnames}
            if (
                normalized.get("course_code", "").strip() == old_entry.course_code
                and normalized.get("course_folder_name", "").strip() == old_entry.course_folder_name
            ):
                normalized["course_folder_name"] = new_entry.course_folder_name
                if "semester_workspace" in normalized:
                    normalized["semester_workspace"] = new_entry.semester_workspace
                changed = True
            updated_rows.append(normalized)
    if changed:
        with path.open("w", newline="", encoding="utf-8") as handle:
            writer = csv.DictWriter(handle, fieldnames=fieldnames)
            writer.writeheader()
            writer.writerows(updated_rows)
    return changed


def reorganize_registry_artifacts(previous_registry_path: Path) -> dict[str, list[str]]:
    previous_registry = load_registry(previous_registry_path)
    current_registry = load_registry(REGISTRY_PATH)

    moved: list[Path] = []
    conflicts: list[str] = []
    updated_csvs: list[Path] = []

    for course_code, old_entry in previous_registry.items():
        new_entry = current_registry.get(course_code)
        if new_entry is None:
            continue
        if (
            old_entry.course_folder_name == new_entry.course_folder_name
            and old_entry.semester_workspace == new_entry.semester_workspace
        ):
            continue

        old_semester = RUNTIME_WORKSPACES / old_entry.semester_workspace
        new_semester = RUNTIME_WORKSPACES / new_entry.semester_workspace

        move_dir_if_exists(
            old_semester / "course_documents" / old_entry.course_folder_name,
            new_semester / "course_documents" / new_entry.course_folder_name,
            moved,
            conflicts,
        )
        move_file_if_exists(
            old_semester / "Course-main-tex" / f"{old_entry.course_folder_name}-main.tex",
            new_semester / "Course-main-tex" / f"{new_entry.course_folder_name}-main.tex",
            moved,
            conflicts,
        )
        move_file_if_exists(
            old_semester / "Course-main-tex" / "Compiled pdf" / f"{old_entry.course_folder_name}-main.pdf",
            new_semester / "Course-main-tex" / "Compiled pdf" / f"{new_entry.course_folder_name}-main.pdf",
            moved,
            conflicts,
        )
        move_file_if_exists(
            TRACKED_MAIN_ROOT / old_entry.semester_workspace / f"{old_entry.course_folder_name}-main.tex",
            TRACKED_MAIN_ROOT / new_entry.semester_workspace / f"{new_entry.course_folder_name}-main.tex",
            moved,
            conflicts,
        )
        move_file_if_exists(
            RUNTIME_PDFS / f"{old_entry.course_folder_name}-main.pdf",
            RUNTIME_PDFS / f"{new_entry.course_folder_name}-main.pdf",
            moved,
            conflicts,
        )
        for suffix in [".md", ".json"]:
            move_file_if_exists(
                RUNTIME_REPORTS / f"{old_entry.course_folder_name}-validation{suffix}",
                RUNTIME_REPORTS / f"{new_entry.course_folder_name}-validation{suffix}",
                moved,
                conflicts,
            )
            move_file_if_exists(
                RUNTIME_REPORTS / f"{old_entry.course_folder_name}-repair{suffix}",
                RUNTIME_REPORTS / f"{new_entry.course_folder_name}-repair{suffix}",
                moved,
                conflicts,
            )
        move_file_if_exists(
            RUNTIME_PROCESSED / f"{old_entry.course_folder_name}.zip",
            RUNTIME_PROCESSED / f"{new_entry.course_folder_name}.zip",
            moved,
            conflicts,
        )
        move_file_if_exists(
            RUNTIME_PROCESSED_REPAIRED / f"{old_entry.course_folder_name}.zip",
            RUNTIME_PROCESSED_REPAIRED / f"{new_entry.course_folder_name}.zip",
            moved,
            conflicts,
        )
        move_dir_if_exists(
            removed_items_backup_dir(old_entry.course_folder_name),
            removed_items_backup_dir(new_entry.course_folder_name),
            moved,
            conflicts,
        )
        old_repair_root = RUNTIME_REPAIR / old_entry.course_folder_name
        new_repair_root = RUNTIME_REPAIR / new_entry.course_folder_name
        move_dir_if_exists(old_repair_root, new_repair_root, moved, conflicts)
        move_file_if_exists(
            PDF_LAYOUT_OVERRIDES_DIR / f"{old_entry.course_folder_name}.csv",
            PDF_LAYOUT_OVERRIDES_DIR / f"{new_entry.course_folder_name}.csv",
            moved,
            conflicts,
        )

        for counts_file in ["10-lectures.csv", "20-assignments.csv", "30-quizzes.csv", "40-exams.csv"]:
            csv_path = WORKSPACE_ROOT / "config" / "staff" / "expected_counts" / counts_file
            if rewrite_course_keyed_csv(
                csv_path,
                old_entry=old_entry,
                new_entry=new_entry,
                columns=["semester_workspace", "course_code", "course_folder_name", "expected_count"],
            ):
                updated_csvs.append(csv_path)
        notes_csv = WORKSPACE_ROOT / "config" / "staff" / "course_notes.csv"
        if rewrite_course_keyed_csv(
            notes_csv,
            old_entry=old_entry,
            new_entry=new_entry,
            columns=["semester_workspace", "course_code", "course_folder_name", "link", "comment"],
        ):
            updated_csvs.append(notes_csv)

    sync_result = sync_registry_state()
    return {
        "moved": [str(path) for path in moved],
        "updated_csvs": [str(path) for path in sorted(set(updated_csvs))],
        "conflicts": conflicts,
        "written_main_tex": [str(path) for path in sync_result["written_main_tex"]],
        "removed_runtime_items": [str(path) for path in sync_result["removed_runtime_items"]],
        "removed_output_items": [str(path) for path in sync_result["removed_output_items"]],
    }


def load_json(path: Path) -> dict | list | None:
    if not path.is_file():
        return None
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return None


def timestamped_name(prefix: str, source_stem: str) -> str:
    return f"{prefix}-{datetime.now().strftime('%Y%m%d-%H%M%S')}-{source_stem}"


def copy_source_to_job_dir(source: Path, base_dir: Path, prefix: str) -> Path:
    ensure_runtime_dirs()
    if source.is_file() and source.suffix.lower() == ".zip":
        job_dir = base_dir / timestamped_name(prefix, source.stem)
        job_dir.mkdir(parents=True, exist_ok=True)
        with zipfile.ZipFile(source) as zf:
            zf.extractall(job_dir)
        strip_junk_files(job_dir)
        top_dirs = [p for p in nonjunk_top_level_items(job_dir) if p.is_dir()]
        if len(top_dirs) != 1:
            top_level_items = nonjunk_top_level_items(job_dir)
            top_level_dir_names = {p.name for p in top_dirs}
            allowed_root_dirs = set(REQUIRED_SUBDIRS) | {"99-abet"}
            root_layout_is_submission = top_level_dir_names and top_level_dir_names.issubset(allowed_root_dirs)
            if (len(top_dirs) == 0 or root_layout_is_submission) and top_level_items:
                staged_course = job_dir / source.stem
                staged_course.mkdir(parents=True, exist_ok=True)
                for item in top_level_items:
                    dest_item = staged_course / item.name
                    if dest_item.exists() and dest_item != item:
                        if dest_item.is_dir():
                            shutil.rmtree(dest_item, ignore_errors=True)
                        else:
                            dest_item.unlink(missing_ok=True)
                    shutil.move(str(item), str(dest_item))
                return normalize_wrapped_course_layout(staged_course)
            raise SystemExit(f"Submission zip must contain exactly one top-level folder. Found {len(top_dirs)}.")
        staged_course = top_dirs[0]
        return normalize_wrapped_course_layout(staged_course)

    if source.is_dir():
        job_dir = base_dir / timestamped_name(prefix, source.name)
        staged_course = job_dir / source.name
        shutil.copytree(source, staged_course)
        return normalize_wrapped_course_layout(staged_course)

    raise SystemExit(f"Submission source not found: {source}")


def stage_from_source(source: Path) -> Path:
    return copy_source_to_job_dir(source, RUNTIME_STAGING, "job")


def prepare_repair_copy(source: Path) -> Path:
    ensure_runtime_dirs()
    temp_copy = copy_source_to_job_dir(source, RUNTIME_REPAIR, "repair-tmp")
    stable_dir = RUNTIME_REPAIR / temp_copy.name
    if stable_dir.exists():
        shutil.rmtree(stable_dir)
    shutil.move(str(temp_copy.parent), str(stable_dir))
    repaired_course = stable_dir / temp_copy.name
    if not repaired_course.is_dir():
        top_dirs = [p for p in stable_dir.iterdir() if p.is_dir() and p.name != "__MACOSX"]
        if len(top_dirs) != 1:
            raise SystemExit(f"Repair folder must contain exactly one top-level course folder. Found {len(top_dirs)}.")
        repaired_course = top_dirs[0]
    return repaired_course


def canonical_course_folder_name(course_dir: Path) -> str | None:
    meta_csv = course_dir / "00-meta" / "instructor_course_identity_template.csv"
    if not meta_csv.is_file():
        return None
    rows = load_meta_rows(meta_csv)
    if len(rows) != 1:
        return None
    meta_row = rows[0]
    course_code = (meta_row.get("course_code") or "").strip()
    if not course_code:
        return None
    registry = load_registry()
    entry = registry.get(course_code)
    if entry is None or not entry.course_folder_name:
        course_name = (meta_row.get("course_name") or "").strip()
        course_folder = (meta_row.get("course_folder_name") or "").strip() or course_dir.name
        sem_ws = (meta_row.get("semester_workspace") or "").strip()
        entry = auto_register_course(
            course_code=course_code,
            course_name=course_name,
            semester_workspace=sem_ws,
            course_folder_name=course_folder,
        )
    return entry.course_folder_name


def canonicalize_repair_copy(repaired_course: Path) -> Path:
    canonical_name = canonical_course_folder_name(repaired_course)
    if not canonical_name or canonical_name == repaired_course.name:
        return repaired_course

    job_dir = repaired_course.parent
    target_job_dir = RUNTIME_REPAIR / canonical_name
    target_course = target_job_dir / canonical_name
    if target_course.exists():
        shutil.rmtree(target_job_dir, ignore_errors=True)

    original_metadata = job_dir / "repair-metadata.json"
    temp_job_dir = RUNTIME_REPAIR / f"{canonical_name}-rename-tmp"
    if temp_job_dir.exists():
        shutil.rmtree(temp_job_dir, ignore_errors=True)
    temp_job_dir.mkdir(parents=True, exist_ok=False)
    temp_course = temp_job_dir / canonical_name
    shutil.move(str(repaired_course), str(temp_course))
    if original_metadata.is_file():
        shutil.copy2(original_metadata, temp_job_dir / "repair-metadata.json")
        original_metadata.unlink(missing_ok=True)
    if job_dir.exists():
        shutil.rmtree(job_dir, ignore_errors=True)
    shutil.move(str(temp_job_dir), str(target_job_dir))
    return target_course


def removed_items_backup_dir(course_folder_name: str) -> Path:
    return RUNTIME_PROCESSED_REPAIRED / f"{course_folder_name}-removed-items"


def rename_removed_items_backup(old_course_folder_name: str, new_course_folder_name: str) -> Path:
    old_dir = removed_items_backup_dir(old_course_folder_name)
    new_dir = removed_items_backup_dir(new_course_folder_name)
    if old_dir.resolve() == new_dir.resolve():
        return new_dir
    if new_dir.exists():
        shutil.rmtree(new_dir, ignore_errors=True)
    if old_dir.exists():
        shutil.move(str(old_dir), str(new_dir))
    return new_dir


def archive_processed_source(source: Path, course_folder_name: str) -> Path | None:
    if not source.is_file() or source.suffix.lower() != ".zip":
        return None
    target = RUNTIME_PROCESSED / f"{course_folder_name}.zip"
    target.parent.mkdir(parents=True, exist_ok=True)
    if source.resolve() == target.resolve():
        return target
    if target.exists():
        target.unlink()
    shutil.copy2(source, target)
    return target


def archive_repaired_source(source: Path, course_folder_name: str) -> Path | None:
    if not source.is_dir():
        return None
    if repair_metadata_path(source) is None:
        return None
    target = RUNTIME_PROCESSED_REPAIRED / f"{course_folder_name}.zip"
    target.parent.mkdir(parents=True, exist_ok=True)
    if target.exists():
        target.unlink()
    archive_base = target.with_suffix("")
    if archive_base.exists():
        shutil.rmtree(archive_base, ignore_errors=True)
    shutil.make_archive(str(archive_base), "zip", root_dir=source.parent, base_dir=source.name)
    created = archive_base.with_suffix(".zip")
    if created != target:
        shutil.move(str(created), str(target))
    return target


def cleanup_staging_for_course(course_folder_name: str, keep_job_dir: Path | None = None) -> None:
    if not RUNTIME_STAGING.exists():
        return
    suffix = f"-{course_folder_name}"
    for job_dir in sorted(p for p in RUNTIME_STAGING.iterdir() if p.is_dir()):
        if keep_job_dir is not None and job_dir.resolve() == keep_job_dir.resolve():
            continue
        if job_dir.name.endswith(suffix):
            shutil.rmtree(job_dir)


def cleanup_repair_for_course(course_folder_name: str) -> None:
    stable_dir = RUNTIME_REPAIR / course_folder_name
    if stable_dir.is_dir():
        shutil.rmtree(stable_dir)


def repair_metadata_path(source: Path) -> Path | None:
    if not source.is_dir():
        return None
    candidate = source.parent / "repair-metadata.json"
    return candidate if candidate.is_file() else None


def original_source_for(source: Path) -> Path:
    metadata_path = repair_metadata_path(source)
    if metadata_path is None:
        return source
    try:
        data = json.loads(metadata_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return source
    original_source = data.get("original_source_path")
    return Path(original_source).resolve() if original_source else source


def cleanup_repair_job_for_source(source: Path) -> None:
    metadata_path = repair_metadata_path(source)
    if metadata_path is None:
        return
    job_dir = metadata_path.parent
    if job_dir.is_dir():
        shutil.rmtree(job_dir)


def cleanup_successful_source(source: Path, staged_course: Path, course_folder_name: str) -> None:
    incoming_source = source.resolve().parent == RUNTIME_INCOMING.resolve()
    if incoming_source and source.exists():
        if source.is_file():
            source.unlink()
        elif source.is_dir():
            shutil.rmtree(source)

    cleanup_staging_for_course(course_folder_name)
    cleanup_repair_job_for_source(source)
    cleanup_repair_for_course(course_folder_name)


def archive_removed_repair_items(
    repaired_course: Path,
    course_folder_name: str,
    relpaths: Iterable[str],
) -> tuple[list[str], Path]:
    backup_dir = removed_items_backup_dir(course_folder_name)
    if backup_dir.exists():
        shutil.rmtree(backup_dir, ignore_errors=True)
    backup_dir.mkdir(parents=True, exist_ok=True)

    manifest_rows: list[dict[str, str]] = []
    backed_up: list[str] = []
    repaired_root = repaired_course.resolve()
    for relpath in sorted(set(relpaths), reverse=True):
        if not relpath:
            continue
        target = (repaired_course / relpath).resolve()
        try:
            target.relative_to(repaired_root)
        except ValueError:
            continue
        if not target.exists():
            continue
        destination = backup_dir / relpath
        destination.parent.mkdir(parents=True, exist_ok=True)
        if target.is_dir():
            shutil.copytree(target, destination, dirs_exist_ok=True)
        else:
            shutil.copy2(target, destination)
        manifest_rows.append(
            {
                "relative_path": relpath,
                "type": "directory" if target.is_dir() else "file",
                "backed_up_at": datetime.now().isoformat(timespec="seconds"),
            }
        )
        backed_up.append(relpath)

    manifest_path = backup_dir / "manifest.json"
    manifest_path.write_text(json.dumps(manifest_rows, indent=2) + "\n", encoding="utf-8")
    return backed_up, backup_dir


def delete_from_repair_copy(
    repaired_course: Path,
    course_folder_name: str,
    relpaths: Iterable[str],
) -> tuple[list[str], Path]:
    deleted, backup_dir = archive_removed_repair_items(repaired_course, course_folder_name, relpaths)
    repaired_root = repaired_course.resolve()
    for relpath in deleted:
        target = (repaired_course / relpath).resolve()
        try:
            target.relative_to(repaired_root)
        except ValueError:
            continue
        if not target.exists():
            continue
        if target.is_dir():
            shutil.rmtree(target, ignore_errors=True)
        else:
            target.unlink(missing_ok=True)

        parent = target.parent
        while parent != repaired_root:
            rel_parent = parent.relative_to(repaired_root)
            if len(rel_parent.parts) == 1 and rel_parent.parts[0] in REQUIRED_SUBDIRS:
                break
            try:
                next(parent.iterdir())
                break
            except StopIteration:
                parent.rmdir()
                parent = parent.parent
            except OSError:
                break

    for subdir in REQUIRED_SUBDIRS:
        (repaired_course / subdir).mkdir(parents=True, exist_ok=True)
    prune_empty_dirs(repaired_course)
    archive_repaired_source(repaired_course, course_folder_name)
    return deleted, backup_dir


def validate_structure(course_dir: Path) -> list[str]:
    errors: list[str] = []
    for subdir in REQUIRED_SUBDIRS:
        if not (course_dir / subdir).is_dir():
            errors.append(f"Missing required folder: {subdir}")
    for relpath in REQUIRED_FILES:
        if not (course_dir / relpath).is_file():
            errors.append(f"Missing required file: {relpath}")
    return errors


def is_valid_pdf(path: Path) -> bool:
    try:
        with path.open("rb") as f:
            header = f.read(1024)
    except OSError:
        return False
    return b"%PDF-" in header


def is_valid_text_file(path: Path) -> bool:
    try:
        data = path.read_bytes()
    except OSError:
        return False
    if b"\x00" in data:
        return False
    try:
        data.decode("utf-8")
    except UnicodeDecodeError:
        return False
    return True


def validate_course_files(course_dir: Path) -> tuple[int, list[str], list[str]]:
    errors: list[str] = []
    warnings: list[str] = []
    inspected_file_count = 0

    for path in sorted(course_dir.rglob("*")):
        relpath = path.relative_to(course_dir)
        if any(is_archive_junk_name(part) for part in relpath.parts):
            continue

        parts = relpath.parts
        if path.is_dir():
            if len(parts) == 1:
                if parts[0] not in REQUIRED_SUBDIRS:
                    errors.append(f"Unexpected top-level folder: {relpath}")
            else:
                errors.append(f"Unexpected nested folder: {relpath}")
            continue

        inspected_file_count += 1
        if len(parts) == 1:
            if path.name not in ALLOWED_ROOT_FILES:
                errors.append(f"Unexpected top-level file: {relpath}")
        else:
            category = parts[0]
            if category not in REQUIRED_SUBDIRS:
                errors.append(f"Unexpected file outside required folders: {relpath}")
            elif len(parts) != 2:
                errors.append(f"Unexpected nested file path: {relpath}")
            else:
                patterns = ALLOWED_FILE_PATTERNS.get(category, [])
                if not any(pattern.fullmatch(path.name) for pattern in patterns):
                    errors.append(f"Unexpected file name in {category}: {relpath}")

        suffix = path.suffix.lower()
        if suffix == ".pdf":
            if not is_valid_pdf(path):
                errors.append(f"Invalid PDF content for file with .pdf extension: {relpath}")
        elif suffix in {".csv", ".tex", ".txt", ".md"}:
            if not is_valid_text_file(path):
                errors.append(f"Invalid text content for file with {suffix} extension: {relpath}")
        else:
            warnings.append(f"Unrecognized file extension detected: {relpath}")

    return inspected_file_count, warnings, errors


def should_auto_remove_path(course_dir: Path, path: Path) -> tuple[bool, str]:
    relpath = path.relative_to(course_dir)
    relpath_lower = relpath.as_posix().lower()
    name_lower = path.name.lower()
    suffix = path.suffix.lower()

    if path.name == ".DS_Store" or path.name.startswith("._") or path.name.startswith("~$"):
        return True, "macOS / Office temporary metadata"
    if relpath.parts and relpath.parts[0] in {"99-abet", "__macosx"}:
        return True, "legacy or system artifact folder"
    if suffix in {".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp", ".heic"}:
        return True, "image artifact not part of allowed submission contract"
    if suffix in {".xlsx", ".xls", ".doc", ".docx", ".ppt", ".pptx", ".key"}:
        return True, "office artifact not part of allowed submission contract"
    if name_lower == "dummy.txt":
        return True, "dummy placeholder file"
    if "screenshot" in relpath_lower or "screen shot" in relpath_lower:
        return True, "screenshot artifact"
    if "checklist" in relpath_lower:
        return True, "checklist artifact"
    return False, ""


def prune_empty_dirs(root: Path) -> None:
    if not root.exists():
        return
    for path in sorted((p for p in root.rglob("*") if p.is_dir()), reverse=True):
        relpath = path.relative_to(root)
        if len(relpath.parts) == 1 and relpath.parts[0] in REQUIRED_SUBDIRS:
            continue
        try:
            if not any(path.iterdir()):
                path.rmdir()
        except OSError:
            pass


def locate_main_tex(course_dir: Path, registry_entry: RegistryEntry) -> Path | None:
    candidates = [
        TRACKED_MAIN_ROOT / registry_entry.semester_workspace / f"{registry_entry.course_folder_name}-main.tex",
        course_dir / f"{registry_entry.course_folder_name}-main.tex",
        course_dir / f"{course_dir.name}-main.tex",
    ]
    for candidate in candidates:
        if candidate.is_file():
            return candidate
    matches = sorted(course_dir.glob("*-main.tex"))
    return matches[0] if matches else None


def ensure_tracked_main_tex(registry_entry: RegistryEntry) -> Path:
    tracked_path = TRACKED_MAIN_ROOT / registry_entry.semester_workspace / f"{registry_entry.course_folder_name}-main.tex"
    if tracked_path.is_file():
        return tracked_path
    tracked_path.parent.mkdir(parents=True, exist_ok=True)
    template_text = MAIN_TEMPLATE_PATH.read_text(encoding="utf-8")
    tracked_path.write_text(render_main_tex(template_text, registry_entry.course_folder_name), encoding="utf-8")
    return tracked_path


def generate_tex_files(course_dir: Path, registry_entry: RegistryEntry) -> list[str]:
    generated: list[str] = []

    meta_csv = course_dir / "00-meta" / "instructor_course_identity_template.csv"
    meta_rows = load_meta_rows(meta_csv)
    if len(meta_rows) != 1:
        raise SystemExit("Identity CSV must contain exactly one populated row.")
    meta_row = dict(meta_rows[0])
    meta_row["course_folder_name"] = registry_entry.course_folder_name
    meta_out = course_dir / "00-meta" / "course-meta.tex"
    meta_out.write_text(build_meta_tex(meta_row), encoding="utf-8")
    generated.append(str(meta_out))

    syllabus_csv = course_dir / "01-syllabus" / "instructor_syllabus_abet_reference.csv"
    syllabus_out = course_dir / "01-syllabus" / "syllabus-abet.tex"
    syllabus_out.write_text(build_syllabus_tex(load_syllabus_rows(syllabus_csv)), encoding="utf-8")
    generated.append(str(syllabus_out))

    schedule_csv = course_dir / "03-schedule" / "instructor_course_schedule_template.csv"
    schedule_out = course_dir / "03-schedule" / "course-schedule.tex"
    schedule_out.write_text(build_schedule_tex(parse_schedule_rows(schedule_csv)), encoding="utf-8")
    generated.append(str(schedule_out))

    grading_csv = course_dir / "90-grades" / "instructor_course_grading_template.csv"
    grading_out = course_dir / "90-grades" / "course-grading.tex"
    grading_out.write_text(build_grading_tex(parse_grading_rows(grading_csv)), encoding="utf-8")
    generated.append(str(grading_out))
    return generated


def validate_submission(source: Path) -> ValidationResult:
    registry = load_registry()
    staged_course = stage_from_source(source)
    errors = validate_structure(staged_course)
    inspected_file_count, audit_warnings, audit_errors = validate_course_files(staged_course)
    warnings: list[str] = list(audit_warnings)
    errors.extend(audit_errors)

    course_code = ""
    course_folder_name = staged_course.name
    semester_workspace = ""
    main_tex_path = ""
    generated_files: list[str] = []

    meta_csv = staged_course / "00-meta" / "instructor_course_identity_template.csv"
    if meta_csv.is_file():
        rows = load_meta_rows(meta_csv)
        if len(rows) == 1:
            meta_row = rows[0]
            course_code = (meta_row.get("course_code") or "").strip()
            if course_code:
                if course_code not in registry:
                    course_name = (meta_row.get("course_name") or "").strip()
                    course_folder = (meta_row.get("course_folder_name") or "").strip() or staged_course.name
                    sem_ws = (meta_row.get("semester_workspace") or "").strip()
                    entry = auto_register_course(
                        course_code=course_code,
                        course_name=course_name,
                        semester_workspace=sem_ws,
                        course_folder_name=course_folder,
                    )
                    registry[course_code] = entry
                else:
                    entry = registry[course_code]

                course_folder_name = entry.course_folder_name
                semester_workspace = entry.semester_workspace
                if not entry.semester_workspace:
                    errors.append(
                        f"Course {entry.course_folder_name} is missing semester_workspace in config/staff/course_registry.csv."
                    )
                if staged_course.name != entry.course_folder_name:
                    warnings.append(
                        f"Top-level folder name '{staged_course.name}' does not match registry folder '{entry.course_folder_name}'."
                    )
                if not errors:
                    ensure_tracked_main_tex(entry)
                    main_tex = locate_main_tex(staged_course, entry)
                    if main_tex is None:
                        errors.append(f"Missing course main tex file for {entry.course_folder_name}.")
                    else:
                        main_tex_path = str(main_tex)
                if not errors:
                    generated_files = generate_tex_files(staged_course, entry)
            else:
                errors.append("Identity CSV course_code field is empty.")
        else:
            errors.append("Identity CSV must contain exactly one populated row.")

    archive_processed_source(original_source_for(source), course_folder_name)
    cleanup_staging_for_course(course_folder_name, keep_job_dir=staged_course.parent)

    return ValidationResult(
        course_code=course_code,
        course_folder_name=course_folder_name,
        semester_workspace=semester_workspace,
        source_path=str(source.resolve()),
        staged_path=str(staged_course),
        main_tex_path=main_tex_path,
        generated_files=generated_files,
        inspected_file_count=inspected_file_count,
        warnings=warnings,
        errors=errors,
    )


def render_validation_report(result: ValidationResult, report_path: Path) -> None:
    file_audit_status = "clean" if not result.errors and not result.warnings else "issues found"
    lines = [
        f"# Validation Report: {result.course_folder_name or 'unknown-course'}",
        "",
        f"- Source: `{result.source_path}`",
        f"- Staged folder: `{result.staged_path}`",
        f"- Course code: `{result.course_code or 'unknown'}`",
        f"- Semester workspace: `{result.semester_workspace or 'unknown'}`",
        f"- Main tex: `{result.main_tex_path or 'missing'}`",
        "",
        "## Status",
        "",
        f"- Errors: {len(result.errors)}",
        f"- Warnings: {len(result.warnings)}",
        f"- Files inspected: {result.inspected_file_count}",
        f"- File audit: {file_audit_status}",
        "",
    ]
    if result.errors:
        lines.extend(["## Errors", ""])
        lines.extend([f"- {item}" for item in result.errors])
        lines.append("")
    if result.warnings:
        lines.extend(["## Warnings", ""])
        lines.extend([f"- {item}" for item in result.warnings])
        lines.append("")
    if result.generated_files:
        lines.extend(["## Generated TeX Files", ""])
        lines.extend([f"- `{item}`" for item in result.generated_files])
        lines.append("")
    report_path.parent.mkdir(parents=True, exist_ok=True)
    report_path.write_text("\n".join(lines), encoding="utf-8")


def write_validation_json(result: ValidationResult, path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(result.__dict__, indent=2), encoding="utf-8")


def load_course_run_status() -> dict[str, dict]:
    if COURSE_RUN_STATUS_JSON.is_file():
        try:
            data = json.loads(COURSE_RUN_STATUS_JSON.read_text(encoding="utf-8"))
            if isinstance(data, dict):
                return data
        except json.JSONDecodeError:
            pass
    return {}


def write_course_run_status(data: dict[str, dict]) -> None:
    COURSE_RUN_STATUS_JSON.parent.mkdir(parents=True, exist_ok=True)
    COURSE_RUN_STATUS_JSON.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
    lines = [
        "# Course Run Status Summary",
        "",
        "| Course | Validation | Repair | Process | Updated | Errors | Warnings | Validation Note | Repair Note | Process Note |",
        "|---|---|---|---|---|---:|---:|---|---|---|",
    ]
    for course_folder_name in sorted(data):
        item = data[course_folder_name]
        lines.append(
            f"| {course_folder_name} | "
            f"{item.get('last_validation_status', '-')}"
            f" | {item.get('last_repair_status', '-')}"
            f" | {item.get('last_process_status', '-')}"
            f" | {item.get('updated_at', '-')}"
            f" | {item.get('last_validation_error_count', 0)}"
            f" | {item.get('last_validation_warning_count', 0)}"
            f" | {item.get('last_validation_note', '')}"
            f" | {item.get('last_repair_note', '')}"
            f" | {item.get('last_process_note', '')} |"
        )
    lines.append("")
    COURSE_RUN_STATUS_MD.write_text("\n".join(lines), encoding="utf-8")


def update_course_run_status(course_folder_name: str, **updates: object) -> None:
    data = load_course_run_status()
    current = dict(data.get(course_folder_name, {}))
    current.update({k: v for k, v in updates.items() if v is not None})
    current["updated_at"] = datetime.now().isoformat(timespec="seconds")
    data[course_folder_name] = current
    write_course_run_status(data)


def clear_stale_validation_status_for_source(result: ValidationResult) -> None:
    data = load_course_run_status()
    stale_keys = [
        course_folder_name
        for course_folder_name, item in data.items()
        if course_folder_name != result.course_folder_name
        and item.get("last_validation_source") == result.source_path
        and not item.get("last_process_status")
    ]
    if not stale_keys:
        return
    for course_folder_name in stale_keys:
        data.pop(course_folder_name, None)
    write_course_run_status(data)


def record_validation_status(result: ValidationResult, report_base: Path) -> None:
    clear_stale_validation_status_for_source(result)
    update_course_run_status(
        result.course_folder_name,
        course_code=result.course_code,
        semester_workspace=result.semester_workspace,
        last_validation_status="PASS" if not result.errors else "FAIL",
        last_validation_error_count=len(result.errors),
        last_validation_warning_count=len(result.warnings),
        last_validation_source=result.source_path,
        last_validation_report_md=str(report_base.with_suffix(".md")),
        last_validation_report_json=str(report_base.with_suffix(".json")),
        last_validation_note=result.errors[0] if result.errors else (result.warnings[0] if result.warnings else "Validation clean"),
    )


def record_process_status(
    course_folder_name: str,
    *,
    status: str,
    note: str,
    pdf_path: Path | None = None,
    source_path: str | None = None,
) -> None:
    update_course_run_status(
        course_folder_name,
        last_process_status=status,
        last_process_pdf=str(pdf_path) if pdf_path else None,
        last_process_source=source_path,
        last_process_note=note,
    )


def record_repair_status(result: RepairResult) -> None:
    update_course_run_status(
        result.course_folder_name,
        last_repair_status="PASS" if result.remaining_error_count == 0 else "ATTENTION",
        last_repair_source=result.source_path,
        last_repair_repaired_path=result.repaired_path,
        last_repair_report_md=result.report_md,
        last_repair_report_json=result.report_json,
        last_repair_note=(
            "Repair copy is clean after safe junk removal"
            if result.remaining_error_count == 0
            else "Repair copy still has unresolved validation issues"
        ),
    )


def meaningful_incoming_items() -> list[Path]:
    if not RUNTIME_INCOMING.exists():
        return []
    return sorted(
        p
        for p in RUNTIME_INCOMING.iterdir()
        if not p.name.startswith(".") and not p.name.startswith("._")
    )


def resolve_workspace_cover_pdf(course_documents: Path) -> Path:
    meta_csv = course_documents / "00-meta" / "instructor_course_identity_template.csv"
    if meta_csv.is_file():
        rows = load_meta_rows(meta_csv)
        if rows:
            cover_image = (rows[0].get("cover_image") or "").strip()
            if cover_image:
                return ASSETS_DEFAULT / "cover.pdf"

    program_cover = PROGRAM_CONFIG_ASSETS / "cover.pdf"
    if program_cover.is_file():
        return program_cover
    return ASSETS_DEFAULT / "cover.pdf"


def prepare_workspace(result: ValidationResult) -> tuple[Path, Path, Path]:
    entry = load_registry()[result.course_code]
    workspace = RUNTIME_WORKSPACES / entry.semester_workspace
    course_documents = workspace / "course_documents"
    main_dir = workspace / "Course-main-tex"
    config_dir = workspace / "config"
    workspace.mkdir(parents=True, exist_ok=True)
    course_documents.mkdir(parents=True, exist_ok=True)
    main_dir.mkdir(parents=True, exist_ok=True)
    config_dir.mkdir(parents=True, exist_ok=True)

    cover_source = resolve_workspace_cover_pdf(Path(result.staged_path))
    shutil.copy2(cover_source, workspace / "cover.pdf")
    shutil.copy2(ASSETS_DEFAULT / "abet-binder.sty", workspace / "abet-binder.sty")
    shutil.copy2(ASSETS_DEFAULT / "abet-binder.sty", main_dir / "abet-binder.sty")

    write_expected_counts(EXPECTED_COUNTS_JSON, config_dir / "expected_counts.tex")
    shutil.copy2(EXPECTED_COUNTS_JSON, config_dir / "expected_counts.json")
    shutil.copy2(PDF_LAYOUT_JSON, config_dir / "pdf_layout_overrides.json")
    shutil.copy2(PDF_LAYOUT_TEX, config_dir / "pdf_layout_overrides.tex")

    latexmkrc = main_dir / "latexmkrc"
    latexmkrc.write_text(
        "# Auto-generated by runtime pipeline to organize latexmk outputs.\n"
        "use File::Path qw(make_path);\n"
        "BEGIN {\n"
        "  make_path('build') unless -d 'build';\n"
        "  make_path('Compiled pdf') unless -d 'Compiled pdf';\n"
        "}\n"
        "$aux_dir = 'build';\n"
        "$out_dir = 'Compiled pdf';\n",
        encoding="utf-8",
    )

    target_course_dir = course_documents / entry.course_folder_name
    if target_course_dir.exists():
        shutil.rmtree(target_course_dir)
    shutil.copytree(Path(result.staged_path), target_course_dir)
    generate_tex_files(target_course_dir, entry)
    ensure_course_override_csv(target_course_dir, PDF_LAYOUT_OVERRIDES_DIR)
    sync_pdf_layout_overrides(overrides_dir=PDF_LAYOUT_OVERRIDES_DIR)
    shutil.copy2(PDF_LAYOUT_JSON, config_dir / "pdf_layout_overrides.json")
    shutil.copy2(PDF_LAYOUT_TEX, config_dir / "pdf_layout_overrides.tex")

    source_main = Path(result.main_tex_path)
    target_main = main_dir / source_main.name
    shutil.copy2(source_main, target_main)
    return workspace, target_course_dir, target_main


def extract_latex_log_errors(main_dir: Path, main_tex: Path) -> list[str]:
    candidate_logs = [
        main_dir / "build" / f"{main_tex.stem}.log",
        main_dir / f"{main_tex.stem}.log",
        main_dir / "Compiled pdf" / f"{main_tex.stem}.log",
    ]
    log_path = None
    for candidate in candidate_logs:
        if candidate.is_file():
            log_path = candidate
            break

    if log_path is None:
        return []

    errors: list[str] = []
    try:
        lines = log_path.read_text(encoding="utf-8", errors="ignore").splitlines()
        capture = False
        current_error: list[str] = []
        for line in lines:
            if line.startswith("! ") or "Fatal error" in line or "LaTeX Error" in line:
                if current_error:
                    errors.append("\n".join(current_error))
                    current_error = []
                capture = True
                current_error.append(line.strip())
            elif capture:
                if not line.strip() or line.startswith("l.") or "Emergency stop" in line:
                    current_error.append(line.strip())
                    errors.append("\n".join(current_error))
                    current_error = []
                    capture = False
                elif len(current_error) < 6:
                    current_error.append(line.strip())

        if current_error:
            errors.append("\n".join(current_error))

        if not errors:
            errors = [line.strip() for line in lines if line.startswith("! ") or "Error:" in line or "Fatal" in line]
            if not errors and len(lines) > 0:
                errors = lines[-25:]
    except Exception:
        pass
    return errors


def run_latexmk(main_tex: Path, main_dir: Path) -> Path:
    cmd = ["latexmk", "-pdf", "-g", "-interaction=nonstopmode", main_tex.name]
    proc = subprocess.run(cmd, cwd=main_dir, capture_output=True, text=True)

    log_errors = extract_latex_log_errors(main_dir, main_tex)

    if proc.returncode != 0:
        err_msg_lines = [f"latexmk failed for {main_tex} (exit code {proc.returncode})."]
        if log_errors:
            err_msg_lines.append("--- LaTeX Log Errors ---")
            err_msg_lines.extend(log_errors[:15])

        if proc.stderr:
            err_msg_lines.append("--- latexmk stderr ---")
            err_msg_lines.append(proc.stderr.strip())
        elif proc.stdout and not log_errors:
            err_msg_lines.append("--- latexmk stdout ---")
            err_msg_lines.append(proc.stdout[-1000:].strip())

        full_err_msg = "\n".join(err_msg_lines)
        logging.error(full_err_msg)
        raise SystemExit(full_err_msg)

    output_pdf = main_dir / "Compiled pdf" / f"{main_tex.stem}.pdf"
    if not output_pdf.exists():
        raise SystemExit(f"Expected compiled PDF not found: {output_pdf}")
    return output_pdf


def generate_reports() -> None:
    RUNTIME_REPORTS.mkdir(parents=True, exist_ok=True)
    report_md = RUNTIME_REPORTS / "course_completion_report.md"
    detailed_dir = RUNTIME_REPORTS / "course_completion_detailed"
    summary_md = RUNTIME_REPORTS / "course_completion_summary.md"
    pev_md = RUNTIME_REPORTS / "course_completion_pev.md"
    results, _category_max, _overrides = compute_results(RUNTIME_WORKSPACES, EXPECTED_COUNTS_JSON, REGISTRY_PATH)
    build_report(RUNTIME_WORKSPACES, report_md, EXPECTED_COUNTS_JSON, REGISTRY_PATH)
    build_detailed_report(detailed_dir, results)
    build_summary_report(summary_md, results)
    notes_path = COURSE_NOTES_CSV if COURSE_NOTES_CSV.exists() else COURSE_NOTES_JSON
    notes = load_course_notes(notes_path) if notes_path.exists() else {}
    build_pev_report(pev_md, results, notes)

    pandoc = shutil.which("pandoc")
    if pandoc:
        for md_path in [report_md, summary_md, pev_md, detailed_dir / "index.md"]:
            subprocess.run([pandoc, str(md_path), "-o", str(md_path.with_suffix(".pdf"))], check=False)


def build_course(source: Path) -> tuple[ValidationResult, Path]:
    effective_source = original_source_for(source)
    result = validate_submission(source)
    report_base = RUNTIME_REPORTS / f"{result.course_folder_name}-validation"
    render_validation_report(result, report_base.with_suffix(".md"))
    write_validation_json(result, report_base.with_suffix(".json"))
    record_validation_status(result, report_base)
    if result.errors:
        logging.info(f"Initial validation failed for {result.course_folder_name} with {len(result.errors)} error(s). Attempting auto-repair...")
        try:
            repair = repair_submission(source)
            repaired_path = Path(repair.repaired_path)

            REMOVABLE_ERROR_PREFIXES = (
                "Unexpected file name in ",
                "Unexpected top-level file: ",
                "Unexpected top-level folder: ",
                "Unexpected file outside required folders: ",
                "Unexpected nested file path: ",
                "Unexpected nested folder: ",
            )
            while repair.remaining_error_count:
                repair_payload = load_json(Path(repair.report_json)) or {}
                remaining_errors = repair_payload.get("remaining_errors", []) if isinstance(repair_payload, dict) else []
                removable = []
                for error in remaining_errors:
                    if isinstance(error, str) and error.startswith(REMOVABLE_ERROR_PREFIXES):
                        relpath = error.split(": ", 1)[1] if ": " in error else ""
                        if relpath:
                            removable.append(relpath)
                if not removable:
                    break
                deleted, _backup_dir = delete_from_repair_copy(repaired_path, result.course_folder_name, sorted(set(removable)))
                if not deleted:
                    break
                repair = refresh_repair_report(repaired_path, additional_removed_items=deleted)

            repaired_result = validate_submission(repaired_path)
            repaired_report_base = RUNTIME_REPORTS / f"{repaired_result.course_folder_name}-validation"
            render_validation_report(repaired_result, repaired_report_base.with_suffix(".md"))
            write_validation_json(repaired_result, repaired_report_base.with_suffix(".json"))
            record_validation_status(repaired_result, repaired_report_base)

            if not repaired_result.errors:
                logging.info(f"Auto-repair succeeded for {repaired_result.course_folder_name}! Proceeding to build.")
                result = repaired_result
                source = repaired_path
            else:
                logging.warning(f"Auto-repair completed for {repaired_result.course_folder_name}, but {len(repaired_result.errors)} error(s) remain.")
        except Exception as exc:
            logging.warning(f"Auto-repair attempt encountered error: {exc}")

    if result.errors:
        err_report_md = report_base.with_suffix(".md")
        report_text = ""
        if err_report_md.is_file():
            try:
                report_text = err_report_md.read_text(encoding="utf-8")
            except Exception:
                pass

        errors_list_str = "\n".join(f"  • {err}" for err in result.errors)
        logging.error(f"Validation failed for {result.course_folder_name} with {len(result.errors)} error(s):\n{errors_list_str}")
        if report_text:
            logging.error(f"--- Validation Report ({err_report_md.name}) ---\n{report_text}")

        record_process_status(
            result.course_folder_name,
            status="FAIL",
            note=f"Validation failed before build (auto-repair could not resolve all errors): {'; '.join(result.errors)}",
            source_path=str(effective_source),
        )

        fail_summary = f"Validation failed for {result.course_folder_name}.\nErrors:\n{errors_list_str}"
        if report_text:
            fail_summary += f"\n\n--- Validation Report ({err_report_md}) ---\n{report_text}"

        raise SystemExit(fail_summary)

    workspace, _course_dir, target_main = prepare_workspace(result)
    try:
        compiled_pdf = run_latexmk(target_main, workspace / "Course-main-tex")
    except SystemExit:
        record_process_status(
            result.course_folder_name,
            status="FAIL",
            note="LaTeX build failed",
            source_path=str(effective_source),
        )
        raise
    final_pdf = RUNTIME_PDFS / compiled_pdf.name
    final_pdf.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(compiled_pdf, final_pdf)
    archive_processed_source(effective_source, result.course_folder_name)
    archive_repaired_source(source, result.course_folder_name)
    cleanup_successful_source(effective_source, Path(result.staged_path), result.course_folder_name)
    generate_reports()
    record_process_status(
        result.course_folder_name,
        status="PASS",
        note="Processed and built successfully",
        pdf_path=final_pdf,
        source_path=str(effective_source),
    )
    return result, final_pdf


def build_all() -> list[Path]:
    pdfs: list[Path] = []
    registry = load_registry()
    entries_by_workspace_course = {
        (entry.semester_workspace, entry.course_folder_name): entry
        for entry in registry.values()
        if entry.semester_workspace and entry.course_folder_name
    }
    for semester_dir in sorted(
        p
        for p in RUNTIME_WORKSPACES.iterdir()
        if p.is_dir() and not p.name.startswith(".") and (p / "course_documents").is_dir()
    ):
        config_dir = semester_dir / "config"
        config_dir.mkdir(parents=True, exist_ok=True)
        write_expected_counts(EXPECTED_COUNTS_JSON, config_dir / "expected_counts.tex")
        shutil.copy2(EXPECTED_COUNTS_JSON, config_dir / "expected_counts.json")
        shutil.copy2(PDF_LAYOUT_JSON, config_dir / "pdf_layout_overrides.json")
        shutil.copy2(PDF_LAYOUT_TEX, config_dir / "pdf_layout_overrides.tex")
        main_dir = semester_dir / "Course-main-tex"
        if not main_dir.is_dir():
            continue
        course_documents_dir = semester_dir / "course_documents"
        for main_tex in sorted(main_dir.glob("*-main.tex")):
            course_folder_name = main_tex.stem.removesuffix("-main")
            course_dir = course_documents_dir / course_folder_name
            entry = entries_by_workspace_course.get((semester_dir.name, course_folder_name))
            if entry is None:
                continue
            if course_dir.is_dir():
                generate_tex_files(course_dir, entry)
            try:
                compiled_pdf = run_latexmk(main_tex, main_dir)
            except SystemExit:
                record_process_status(
                    course_folder_name,
                    status="FAIL",
                    note="LaTeX build failed during Build-All",
                )
                raise
            final_pdf = RUNTIME_PDFS / compiled_pdf.name
            shutil.copy2(compiled_pdf, final_pdf)
            pdfs.append(final_pdf)
            record_process_status(
                course_folder_name,
                status="PASS",
                note="Rebuilt successfully via Build-All",
                pdf_path=final_pdf,
            )
    generate_reports()
    return pdfs


def build_one(course_folder_name: str) -> Path:
    registry = load_registry()
    entries_by_folder = {entry.course_folder_name: entry for entry in registry.values()}
    entry = entries_by_folder.get(course_folder_name)
    if entry is None:
        raise SystemExit(
            f"Course folder '{course_folder_name}' not found in config/staff/course_registry.csv."
        )

    semester_dir = RUNTIME_WORKSPACES / entry.semester_workspace
    if not semester_dir.is_dir():
        raise SystemExit(
            f"Semester workspace not found for {course_folder_name}: {semester_dir}"
        )

    config_dir = semester_dir / "config"
    config_dir.mkdir(parents=True, exist_ok=True)
    write_expected_counts(EXPECTED_COUNTS_JSON, config_dir / "expected_counts.tex")
    shutil.copy2(EXPECTED_COUNTS_JSON, config_dir / "expected_counts.json")
    shutil.copy2(PDF_LAYOUT_JSON, config_dir / "pdf_layout_overrides.json")
    shutil.copy2(PDF_LAYOUT_TEX, config_dir / "pdf_layout_overrides.tex")

    course_dir = semester_dir / "course_documents" / course_folder_name
    if not course_dir.is_dir():
        raise SystemExit(
            f"Course workspace not found for {course_folder_name}: {course_dir}"
        )
    generate_tex_files(course_dir, entry)

    main_dir = semester_dir / "Course-main-tex"
    main_tex = main_dir / f"{course_folder_name}-main.tex"
    if not main_tex.is_file():
        raise SystemExit(f"Course main tex not found: {main_tex}")

    try:
        compiled_pdf = run_latexmk(main_tex, main_dir)
    except SystemExit:
        record_process_status(
            course_folder_name,
            status="FAIL",
            note="LaTeX build failed during Build-One-Course",
        )
        raise
    final_pdf = RUNTIME_PDFS / compiled_pdf.name
    final_pdf.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(compiled_pdf, final_pdf)
    generate_reports()
    record_process_status(
        course_folder_name,
        status="PASS",
        note="Rebuilt successfully via Build-One-Course",
        pdf_path=final_pdf,
    )
    return final_pdf


def validate_all_incoming() -> tuple[list[BatchValidationResult], Path]:
    sources = [p for p in meaningful_incoming_items() if p.is_file() and p.suffix.lower() == ".zip"]
    if not sources:
        raise SystemExit("No zip files found in runtime/incoming.")

    batch_results: list[BatchValidationResult] = []
    timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    summary_md = RUNTIME_REPORTS / f"validate-all-incoming-{timestamp}.md"
    summary_json = RUNTIME_REPORTS / f"validate-all-incoming-{timestamp}.json"

    for source in sources:
        result = validate_submission(source)
        report_base = RUNTIME_REPORTS / f"{result.course_folder_name}-validation"
        report_md = report_base.with_suffix(".md")
        report_json = report_base.with_suffix(".json")
        render_validation_report(result, report_md)
        write_validation_json(result, report_json)
        record_validation_status(result, report_base)
        batch_results.append(
            BatchValidationResult(
                source_path=str(source.resolve()),
                course_folder_name=result.course_folder_name,
                report_md=str(report_md),
                report_json=str(report_json),
                warning_count=len(result.warnings),
                error_count=len(result.errors),
                status="PASS" if not result.errors else "FAIL",
            )
        )

    lines = [
        "# Validate All Incoming Summary",
        "",
        f"- Generated: `{datetime.now().isoformat(timespec='seconds')}`",
        f"- Inbox: `{RUNTIME_INCOMING}`",
        f"- Zips scanned: {len(sources)}",
        "",
        "| Status | Course | Errors | Warnings | Source | Report |",
        "|---|---|---:|---:|---|---|",
    ]
    for item in batch_results:
        lines.append(
            f"| {item.status} | {item.course_folder_name} | {item.error_count} | {item.warning_count} | `{Path(item.source_path).name}` | `{Path(item.report_md).name}` |"
        )
    lines.append("")
    summary_md.write_text("\n".join(lines), encoding="utf-8")
    summary_json.write_text(json.dumps([item.__dict__ for item in batch_results], indent=2), encoding="utf-8")
    return batch_results, summary_md


def render_repair_report(repair: RepairResult, validation: ValidationResult, report_path: Path) -> None:
    lines = [
        f"# Repair Report: {repair.course_folder_name or 'unknown-course'}",
        "",
        f"- Original source: `{repair.original_source_path}`",
        f"- Repair source used: `{repair.source_path}`",
        f"- Repaired folder: `{repair.repaired_path}`",
        f"- Removed items: {len(repair.removed_items)}",
        f"- Removed-item backup: `{repair.removed_backup_path}`",
        f"- Remaining validation errors: {repair.remaining_error_count}",
        f"- Remaining validation warnings: {repair.remaining_warning_count}",
        "",
    ]
    if repair.removed_items:
        lines.extend(["## Auto-Removed Items", ""])
        lines.extend([f"- `{item}`" for item in repair.removed_items])
        lines.append("")
    if validation.errors:
        lines.extend(["## Remaining Errors", ""])
        lines.extend([f"- {item}" for item in validation.errors])
        lines.append("")
    if validation.warnings:
        lines.extend(["## Remaining Warnings", ""])
        lines.extend([f"- {item}" for item in validation.warnings])
        lines.append("")
    report_path.parent.mkdir(parents=True, exist_ok=True)
    report_path.write_text("\n".join(lines), encoding="utf-8")


def write_repair_json(repair: RepairResult, validation: ValidationResult, path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    payload = {
        **repair.__dict__,
        "remaining_errors": validation.errors,
        "remaining_warnings": validation.warnings,
    }
    path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")


def refresh_repair_report(repaired_path: Path, additional_removed_items: Iterable[str] = ()) -> RepairResult:
    if not repaired_path.is_dir():
        raise SystemExit(f"Repair folder not found: {repaired_path}")
    repaired_path = canonicalize_repair_copy(repaired_path)
    original_source = original_source_for(repaired_path)
    validation = validate_submission(repaired_path)
    report_base = RUNTIME_REPORTS / f"{validation.course_folder_name}-repair"
    prior_payload = load_json(report_base.with_suffix(".json")) or {}
    removed_items = prior_payload.get("removed_items", []) if isinstance(prior_payload, dict) else []
    removed_items = sorted(set(removed_items).union(str(item) for item in additional_removed_items if str(item).strip()))
    repair = RepairResult(
        source_path=str(repaired_path.resolve()),
        original_source_path=str(original_source.resolve()),
        course_folder_name=validation.course_folder_name,
        repaired_path=str(repaired_path.resolve()),
        removed_items=removed_items,
        removed_backup_path=str(removed_items_backup_dir(validation.course_folder_name).resolve()),
        remaining_warning_count=len(validation.warnings),
        remaining_error_count=len(validation.errors),
        report_md=str(report_base.with_suffix(".md")),
        report_json=str(report_base.with_suffix(".json")),
    )
    archive_processed_source(original_source, validation.course_folder_name)
    archive_repaired_source(repaired_path, validation.course_folder_name)
    render_repair_report(repair, validation, report_base.with_suffix(".md"))
    write_repair_json(repair, validation, report_base.with_suffix(".json"))
    record_repair_status(repair)
    return repair


def repair_submission(source: Path) -> RepairResult:
    metadata_path = repair_metadata_path(source)
    if metadata_path is not None:
        # An existing repair copy should be refreshed in place, not replaced.
        return refresh_repair_report(source)

    original_source = original_source_for(source)
    repaired_course = prepare_repair_copy(source)
    repaired_course = canonicalize_repair_copy(repaired_course)
    metadata_path = repaired_course.parent / "repair-metadata.json"
    metadata_path.write_text(
        json.dumps(
            {
                "original_source_path": str(original_source.resolve()),
                "repair_source_path": str(repaired_course.resolve()),
                "created_at": datetime.now().isoformat(timespec="seconds"),
            },
            indent=2,
        )
        + "\n",
        encoding="utf-8",
    )

    removable_paths: list[str] = []
    for path in sorted(repaired_course.rglob("*"), reverse=True):
        removable, _reason = should_auto_remove_path(repaired_course, path)
        if removable:
            removable_paths.append(str(path.relative_to(repaired_course)))
    removed_items, backup_dir = delete_from_repair_copy(
        repaired_course,
        repaired_course.name,
        removable_paths,
    )
    for subdir in REQUIRED_SUBDIRS:
        (repaired_course / subdir).mkdir(parents=True, exist_ok=True)
    prune_empty_dirs(repaired_course)

    validation = validate_submission(repaired_course)
    if validation.course_folder_name != repaired_course.name:
        backup_dir = rename_removed_items_backup(repaired_course.name, validation.course_folder_name)
    report_base = RUNTIME_REPORTS / f"{validation.course_folder_name}-repair"
    repair = RepairResult(
        source_path=str(source.resolve()),
        original_source_path=str(original_source.resolve()),
        course_folder_name=validation.course_folder_name,
        repaired_path=str(repaired_course.resolve()),
        removed_items=sorted(set(removed_items)),
        removed_backup_path=str(backup_dir.resolve()),
        remaining_warning_count=len(validation.warnings),
        remaining_error_count=len(validation.errors),
        report_md=str(report_base.with_suffix(".md")),
        report_json=str(report_base.with_suffix(".json")),
    )
    archive_processed_source(original_source, validation.course_folder_name)
    archive_repaired_source(repaired_course, validation.course_folder_name)
    render_repair_report(repair, validation, report_base.with_suffix(".md"))
    write_repair_json(repair, validation, report_base.with_suffix(".json"))
    record_repair_status(repair)
    return repair


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Runtime pipeline for CourseDisplay")
    subparsers = parser.add_subparsers(dest="command", required=True)

    validate_parser = subparsers.add_parser("validate", help="Validate a zip or course folder")
    validate_parser.add_argument("source", nargs="?", help="Zip or course folder path")

    build_course_parser = subparsers.add_parser("build-course", help="Validate, stage, and build one course")
    build_course_parser.add_argument("source", nargs="?", help="Zip or course folder path")

    repair_parser = subparsers.add_parser("repair-submission", help="Create a repair copy, remove safe junk, and revalidate")
    repair_parser.add_argument("source", nargs="?", help="Zip or course folder path")
    refresh_repair_parser = subparsers.add_parser("refresh-repair-report", help="Revalidate an existing repair folder and rewrite its repair report")
    refresh_repair_parser.add_argument("repaired_path", help="Path to the repaired course folder in runtime/repair")
    delete_repair_parser = subparsers.add_parser("delete-from-repair-copy", help="Back up and delete selected items from an existing repair folder")
    delete_repair_parser.add_argument("repaired_path", help="Path to the repaired course folder in runtime/repair")
    delete_repair_parser.add_argument("relpaths", nargs="+", help="One or more relative paths inside the repaired course folder")
    reorganize_parser = subparsers.add_parser("reorganize-registry-artifacts", help="Move runtime/config artifacts to match updated registry entries")
    reorganize_parser.add_argument("previous_registry_path", help="Snapshot of the registry before edits were saved")

    subparsers.add_parser("build-all", help="Rebuild all courses currently in runtime/workspaces")
    build_one_parser = subparsers.add_parser("build-one", help="Rebuild one existing course in runtime/workspaces")
    build_one_parser.add_argument("course_folder_name", help="Course folder name from config/staff/course_registry.csv")
    subparsers.add_parser("validate-all-incoming", help="Validate every zip currently in runtime/incoming")
    subparsers.add_parser("regenerate-reports", help="Regenerate completion and PEV reports from current runtime/workspaces")
    return parser.parse_args(argv)


def pick_default_source() -> Path:
    candidates: list[Path] = []
    for base in [RUNTIME_INCOMING]:
        if base.exists():
            candidates.extend(sorted(base.glob("*.zip")))
            candidates.extend(sorted(p for p in base.iterdir() if p.is_dir() and not p.name.startswith('.')))
    if not candidates:
        raise SystemExit("No source provided and no zip/folder found in runtime/incoming.")
    return candidates[0]


def main(argv: list[str] | None = None) -> int:
    backfill_workspace(WORKSPACE_ROOT, refresh_launchers=True)
    ensure_runtime_dirs()
    sync_registry_state()
    args = parse_args(argv)

    if args.command == "build-all":
        pdfs = build_all()
        print(f"Rebuilt {len(pdfs)} course PDFs into {RUNTIME_PDFS}")
        return 0

    if args.command == "repair-submission":
        source = Path(args.source).resolve() if args.source else pick_default_source()
        repair = repair_submission(source)
        print(f"Repair report: {repair.report_md}")
        print(f"Repaired submission folder: {repair.repaired_path}")
        print(
            f"Remaining issues: errors={repair.remaining_error_count}, "
            f"warnings={repair.remaining_warning_count}"
        )
        return 1 if repair.remaining_error_count else 0

    if args.command == "refresh-repair-report":
        repair = refresh_repair_report(Path(args.repaired_path).resolve())
        print(f"Repair report: {repair.report_md}")
        print(f"Repaired submission folder: {repair.repaired_path}")
        print(
            f"Remaining issues: errors={repair.remaining_error_count}, "
            f"warnings={repair.remaining_warning_count}"
        )
        return 1 if repair.remaining_error_count else 0

    if args.command == "delete-from-repair-copy":
        repaired_path = Path(args.repaired_path).resolve()
        course_folder_name = repaired_path.name
        deleted, backup_dir = delete_from_repair_copy(repaired_path, course_folder_name, args.relpaths)
        repair = refresh_repair_report(repaired_path, additional_removed_items=deleted)
        print(f"Deleted {len(deleted)} item(s) from {repaired_path}")
        print(f"Removed-item backup: {backup_dir}")
        print(f"Repair report: {repair.report_md}")
        return 0

    if args.command == "build-one":
        pdf_path = build_one(args.course_folder_name)
        print(f"Rebuilt course PDF: {pdf_path}")
        return 0

    if args.command == "validate-all-incoming":
        results, summary_path = validate_all_incoming()
        print(f"Batch validation summary: {summary_path}")
        failed = [item for item in results if item.error_count]
        for item in results:
            print(
                f"{item.status}: {item.course_folder_name} "
                f"(errors={item.error_count}, warnings={item.warning_count})"
            )
        return 1 if failed else 0

    if args.command == "regenerate-reports":
        generate_reports()
        print(f"Regenerated reports in {RUNTIME_REPORTS}")
        return 0

    if args.command == "reorganize-registry-artifacts":
        summary = reorganize_registry_artifacts(Path(args.previous_registry_path).resolve())
        print(json.dumps(summary, indent=2))
        return 1 if summary["conflicts"] else 0

    source = Path(args.source).resolve() if args.source else pick_default_source()
    if args.command == "validate":
        result = validate_submission(source)
        report_base = RUNTIME_REPORTS / f"{result.course_folder_name}-validation"
        render_validation_report(result, report_base.with_suffix(".md"))
        write_validation_json(result, report_base.with_suffix(".json"))
        record_validation_status(result, report_base)
        print(f"Validation report: {report_base.with_suffix('.md')}")
        if result.errors:
            for err in result.errors:
                print(f"ERROR: {err}")
            return 1
        for warning in result.warnings:
            print(f"WARNING: {warning}")
        return 0

    result, pdf_path = build_course(source)
    print(f"Built course PDF: {pdf_path}")
    print(f"Validation report: {RUNTIME_REPORTS / (result.course_folder_name + '-validation.md')}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
