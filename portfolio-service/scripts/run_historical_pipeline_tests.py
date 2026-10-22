#!/usr/bin/env python3
"""Run historical zip files through the runtime pipeline one at a time."""

from __future__ import annotations

import argparse
import csv
import io
import json
import os
import re
import shutil
import zipfile
from dataclasses import asdict, dataclass
from datetime import datetime
from pathlib import Path


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run dashboard-style processing tests for historical zips.")
    parser.add_argument("--program-root", type=Path, required=True, help="Program root containing runtime/ and config/")
    parser.add_argument("--historical-dir", type=Path, help="Directory containing historical zips")
    parser.add_argument(
        "--semester-workspace",
        default="ABET-Course-Display-sem1y1",
        help="Default semester workspace to use when auto-registering missing courses",
    )
    parser.add_argument(
        "--auto-register-missing",
        action="store_true",
        help="Append missing course registry entries based on zip metadata so full builds can run",
    )
    return parser.parse_args()


args = parse_args()
program_root = args.program_root.resolve()
os.environ["COURSEDISPLAY_WORKSPACE_ROOT"] = str(program_root)

import runtime_pipeline as pipeline  # noqa: E402


REMOVABLE_ERROR_PREFIXES = (
    "Unexpected file name in ",
    "Unexpected top-level file: ",
    "Unexpected top-level folder: ",
    "Unexpected file outside required folders: ",
    "Unexpected nested file path: ",
    "Unexpected nested folder: ",
)


@dataclass
class TestRunResult:
    source_zip: str
    course_code: str
    course_folder_name: str
    validation_errors: list[str]
    repair_removed: list[str]
    repair_remaining_errors: list[str]
    final_status: str
    pdf_path: str
    notes: list[str]


def registry_rows() -> list[dict[str, str]]:
    if not pipeline.REGISTRY_PATH.is_file():
        return []
    with pipeline.REGISTRY_PATH.open(newline="", encoding="utf-8-sig") as handle:
        return list(csv.DictReader(handle))


def read_zip_identity(zip_path: Path) -> dict[str, str]:
    with zipfile.ZipFile(zip_path) as zf:
        meta_name = next((name for name in zf.namelist() if name.endswith("00-meta/instructor_course_identity_template.csv")), None)
        if meta_name is None:
            return {}
        data = zf.read(meta_name).decode("utf-8-sig")
    rows = list(csv.DictReader(io.StringIO(data)))
    return rows[0] if rows else {}


def slugify_course_name(course_name: str) -> str:
    slug = re.sub(r"[^A-Za-z0-9]+", "-", course_name.strip()).strip("-")
    return slug or "course"


def ensure_registry_entry(zip_path: Path, default_semester_workspace: str) -> tuple[str, str]:
    identity = read_zip_identity(zip_path)
    course_code = (identity.get("course_code") or "").strip()
    course_name = (identity.get("course_name") or "").strip()
    if not course_code:
        raise SystemExit(f"Could not determine course_code from {zip_path}")

    registry = pipeline.load_registry()
    if course_code in registry:
        entry = registry[course_code]
        return entry.course_code, entry.course_folder_name

    rows = registry_rows()
    course_folder_name = f"{course_code}-{slugify_course_name(course_name)}"
    rows.append(
        {
            "course_code": course_code,
            "course_name": course_name,
            "semester_workspace": default_semester_workspace,
            "course_folder_name": course_folder_name,
        }
    )
    with pipeline.REGISTRY_PATH.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(
            handle,
            fieldnames=["course_code", "course_name", "semester_workspace", "course_folder_name"],
        )
        writer.writeheader()
        writer.writerows(rows)
    return course_code, course_folder_name


def reset_runtime_state() -> None:
    for path in [pipeline.RUNTIME_INCOMING, pipeline.RUNTIME_STAGING, pipeline.RUNTIME_REPAIR]:
        if not path.exists():
            continue
        for item in path.iterdir():
            if item.name.startswith("."):
                continue
            if item.is_dir():
                shutil.rmtree(item, ignore_errors=True)
            else:
                item.unlink(missing_ok=True)


def parse_removable_paths(errors: list[str]) -> list[str]:
    relpaths: list[str] = []
    for error in errors:
        if error.startswith(REMOVABLE_ERROR_PREFIXES):
            relpath = error.split(": ", 1)[1] if ": " in error else ""
            if relpath:
                relpaths.append(relpath)
    return sorted(set(relpaths))


def run_one(zip_path: Path, default_semester_workspace: str, auto_register_missing: bool) -> TestRunResult:
    reset_runtime_state()
    incoming_zip = pipeline.RUNTIME_INCOMING / zip_path.name
    shutil.copy2(zip_path, incoming_zip)

    course_code = ""
    course_folder_name = zip_path.stem
    validation_errors: list[str] = []
    repair_removed: list[str] = []
    repair_remaining_errors: list[str] = []
    notes: list[str] = []
    pdf_path = ""

    if auto_register_missing:
        course_code, course_folder_name = ensure_registry_entry(incoming_zip, default_semester_workspace)
        notes.append("Ensured registry entry exists")

    validation = pipeline.validate_submission(incoming_zip)
    course_code = validation.course_code or course_code
    course_folder_name = validation.course_folder_name or course_folder_name
    validation_errors = list(validation.errors)

    source_for_build: Path = incoming_zip
    if validation.errors:
        notes.append(f"Initial validation failed with {len(validation.errors)} error(s)")
        repair = pipeline.repair_submission(incoming_zip)
        repair_removed = list(repair.removed_items)
        repaired_path = Path(repair.repaired_path)

        while repair.remaining_error_count:
            repair_payload = pipeline.load_json(Path(repair.report_json)) or {}
            remaining_errors = repair_payload.get("remaining_errors", []) if isinstance(repair_payload, dict) else []
            removable = parse_removable_paths(list(remaining_errors))
            if not removable:
                repair_remaining_errors = list(remaining_errors)
                break
            deleted, _backup_dir = pipeline.delete_from_repair_copy(repaired_path, course_folder_name, removable)
            repair_removed = sorted(set(repair_removed).union(deleted))
            repair = pipeline.refresh_repair_report(repaired_path, additional_removed_items=deleted)
            notes.append(f"Deleted {len(deleted)} removable repair item(s)")

        if repair.remaining_error_count:
            repair_payload = pipeline.load_json(Path(repair.report_json)) or {}
            repair_remaining_errors = repair_payload.get("remaining_errors", []) if isinstance(repair_payload, dict) else []
            return TestRunResult(
                source_zip=str(zip_path),
                course_code=course_code,
                course_folder_name=course_folder_name,
                validation_errors=validation_errors,
                repair_removed=repair_removed,
                repair_remaining_errors=repair_remaining_errors,
                final_status="FAILED_REPAIR",
                pdf_path="",
                notes=notes,
            )
        source_for_build = repaired_path
        notes.append("Repair completed cleanly")

    result, final_pdf = pipeline.build_course(source_for_build)
    pdf_path = str(final_pdf)
    notes.append("Build completed successfully")
    return TestRunResult(
        source_zip=str(zip_path),
        course_code=result.course_code or course_code,
        course_folder_name=result.course_folder_name or course_folder_name,
        validation_errors=validation_errors,
        repair_removed=repair_removed,
        repair_remaining_errors=[],
        final_status="PASS",
        pdf_path=pdf_path,
        notes=notes,
    )


def write_summary(results: list[TestRunResult], historical_dir: Path) -> Path:
    timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    output_dir = program_root / "runtime" / "output" / "reports"
    output_dir.mkdir(parents=True, exist_ok=True)
    summary_path = output_dir / f"historical-pipeline-test-{timestamp}.json"
    summary_path.write_text(json.dumps([asdict(item) for item in results], indent=2) + "\n", encoding="utf-8")
    return summary_path


def main() -> int:
    historical_dir = args.historical_dir.resolve() if args.historical_dir else (program_root / "runtime" / "historical")
    zips = sorted(historical_dir.glob("*.zip"))
    if not zips:
        raise SystemExit(f"No historical zips found in {historical_dir}")

    results: list[TestRunResult] = []
    for zip_path in zips:
        try:
            result = run_one(zip_path, args.semester_workspace, args.auto_register_missing)
        except SystemExit as exc:
            result = TestRunResult(
                source_zip=str(zip_path),
                course_code="",
                course_folder_name=zip_path.stem,
                validation_errors=[],
                repair_removed=[],
                repair_remaining_errors=[],
                final_status="PIPELINE_ERROR",
                pdf_path="",
                notes=[str(exc)],
            )
        results.append(result)
        print(f"{result.final_status}: {zip_path.name}")

    summary_path = write_summary(results, historical_dir)
    print(f"Summary written to {summary_path}")
    return 0 if all(item.final_status == "PASS" for item in results) else 1


if __name__ == "__main__":
    raise SystemExit(main())
