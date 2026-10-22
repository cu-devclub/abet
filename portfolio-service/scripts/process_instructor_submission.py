#!/usr/bin/env python3
"""Validate and stage one instructor submission zip for course display generation."""

from __future__ import annotations

import argparse
import csv
import json
import shutil
import sys
import tempfile
import zipfile
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path

from generate_course_grading_from_csv import build_tex as build_grading_tex
from generate_course_grading_from_csv import parse_rows as parse_grading_rows
from generate_course_meta_from_csv import build_meta_tex, load_rows as load_meta_rows, make_course_folder_name
from generate_course_schedule_from_csv import build_schedule_tex, parse_rows as parse_schedule_rows
from generate_syllabus_abet_from_csv import build_tex as build_syllabus_tex
from generate_syllabus_abet_from_csv import load_rows as load_syllabus_rows

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


@dataclass
class RegistryEntry:
    course_code: str
    course_name: str
    semester_workspace: str
    course_folder_name: str


@dataclass
class ProcessingSummary:
    course_code: str
    course_folder_name: str
    semester_workspace: str
    extracted_to: str
    generated_files: list[str]
    warnings: list[str]


def load_registry(path: Path) -> dict[str, RegistryEntry]:
    with path.open(newline="", encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        required = {"course_code", "course_name", "semester_workspace", "course_folder_name"}
        if reader.fieldnames is None:
            raise SystemExit("Registry CSV has no header row.")
        missing = required - set(reader.fieldnames)
        if missing:
            raise SystemExit(f"Registry missing required columns: {', '.join(sorted(missing))}")
        data: dict[str, RegistryEntry] = {}
        for row in reader:
            code = (row.get("course_code") or "").strip()
            if not code:
                continue
            data[code] = RegistryEntry(
                course_code=code,
                course_name=(row.get("course_name") or "").strip(),
                semester_workspace=(row.get("semester_workspace") or "").strip(),
                course_folder_name=(row.get("course_folder_name") or "").strip(),
            )
        return data


def extract_submission(zip_path: Path, staging_root: Path) -> Path:
    timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    job_dir = staging_root / f"job-{timestamp}-{zip_path.stem}"
    job_dir.mkdir(parents=True, exist_ok=False)
    with zipfile.ZipFile(zip_path) as zf:
        zf.extractall(job_dir)

    top_level_dirs = [p for p in job_dir.iterdir() if p.is_dir() and not p.name.startswith("__MACOSX")]
    if len(top_level_dirs) != 1:
        raise SystemExit(
            f"Submission zip must contain exactly one top-level folder. Found {len(top_level_dirs)} folders."
        )
    return top_level_dirs[0]


def validate_structure(course_dir: Path) -> None:
    missing_dirs = [name for name in REQUIRED_SUBDIRS if not (course_dir / name).is_dir()]
    if missing_dirs:
        raise SystemExit(f"Submission is missing required folders: {', '.join(missing_dirs)}")

    missing_files = [name for name in REQUIRED_FILES if not (course_dir / name).is_file()]
    if missing_files:
        raise SystemExit(f"Submission is missing required files: {', '.join(missing_files)}")


def generate_tex_files(course_dir: Path, registry_entry: RegistryEntry) -> list[str]:
    generated: list[str] = []

    meta_csv = course_dir / "00-meta" / "instructor_course_identity_template.csv"
    meta_rows = load_meta_rows(meta_csv)
    if len(meta_rows) != 1:
        raise SystemExit("Identity CSV must contain exactly one populated row.")
    meta_row = dict(meta_rows[0])
    meta_row["course_folder_name"] = registry_entry.course_folder_name or course_dir.name
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


def copy_to_processed(zip_path: Path, processed_root: Path) -> Path:
    processed_root.mkdir(parents=True, exist_ok=True)
    target = processed_root / zip_path.name
    shutil.copy2(zip_path, target)
    return target


def write_summary(summary: ProcessingSummary, output_root: Path) -> Path:
    output_root.mkdir(parents=True, exist_ok=True)
    summary_path = output_root / f"{summary.course_folder_name}-processing-summary.json"
    summary_path.write_text(json.dumps(summary.__dict__, indent=2), encoding="utf-8")
    return summary_path


def main() -> int:
    parser = argparse.ArgumentParser(description="Process one instructor course submission zip")
    parser.add_argument("zip_file", help="Path to instructor zip")
    parser.add_argument(
        "--registry",
        default=str(Path(__file__).resolve().parent.parent / "config" / "staff" / "course_registry.csv"),
        help="Course registry CSV",
    )
    parser.add_argument(
        "--staging-root",
        default=str(Path(__file__).resolve().parent.parent / "staging"),
        help="Staging root",
    )
    parser.add_argument(
        "--processed-root",
        default=str(Path(__file__).resolve().parent.parent / "processed"),
        help="Processed zip root",
    )
    parser.add_argument(
        "--output-root",
        default=str(Path(__file__).resolve().parent.parent / "output"),
        help="Output summary root",
    )
    args = parser.parse_args()

    zip_path = Path(args.zip_file).resolve()
    if not zip_path.is_file():
        raise SystemExit(f"Zip file not found: {zip_path}")

    registry = load_registry(Path(args.registry).resolve())
    extracted_course_dir = extract_submission(zip_path, Path(args.staging_root).resolve())
    validate_structure(extracted_course_dir)

    meta_rows = load_meta_rows(extracted_course_dir / "00-meta" / "instructor_course_identity_template.csv")
    if len(meta_rows) != 1:
        raise SystemExit("Identity CSV must contain exactly one populated row.")
    course_code = meta_rows[0]["course_code"].strip()
    if course_code not in registry:
        raise SystemExit(
            f"Course code {course_code} not found in registry. Add it to config/staff/course_registry.csv first."
        )
    entry = registry[course_code]

    warnings: list[str] = []
    if extracted_course_dir.name != entry.course_folder_name:
        warnings.append(
            f"Top-level folder name '{extracted_course_dir.name}' does not match registry folder '{entry.course_folder_name}'."
        )

    generated_files = generate_tex_files(extracted_course_dir, entry)
    processed_zip = copy_to_processed(zip_path, Path(args.processed_root).resolve())
    summary = ProcessingSummary(
        course_code=entry.course_code,
        course_folder_name=entry.course_folder_name,
        semester_workspace=entry.semester_workspace,
        extracted_to=str(extracted_course_dir),
        generated_files=generated_files,
        warnings=warnings,
    )
    summary_path = write_summary(summary, Path(args.output_root).resolve())

    print(f"Processed zip: {processed_zip}")
    print(f"Staged course folder: {extracted_course_dir}")
    print(f"Suggested semester workspace: {entry.semester_workspace}")
    print(f"Summary written to: {summary_path}")
    if warnings:
        print("Warnings:")
        for warning in warnings:
            print(f"- {warning}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
