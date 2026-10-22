#!/usr/bin/env python3
"""Manage staff-editable expected-count CSVs and derive JSON/TeX outputs."""

from __future__ import annotations

import argparse
import csv
import json
from dataclasses import dataclass
from pathlib import Path
import os

from build_all_pdfs import write_expected_counts


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
DEFAULT_REGISTRY = WORKSPACE_ROOT / "config" / "staff" / "course_registry.csv"
DEFAULT_COUNTS_DIR = WORKSPACE_ROOT / "config" / "staff" / "expected_counts"
DEFAULT_JSON = WORKSPACE_ROOT / "config" / "generated" / "expected_counts.json"
DEFAULT_TEX = WORKSPACE_ROOT / "config" / "generated" / "expected_counts.tex"

CATEGORY_TO_CSV = {
    "10-lectures": "10-lectures.csv",
    "20-assignments": "20-assignments.csv",
    "30-quizzes": "30-quizzes.csv",
    "40-exams": "40-exams.csv",
}

CSV_HEADERS = [
    "semester_workspace",
    "course_code",
    "course_folder_name",
    "expected_count",
]


@dataclass(frozen=True)
class RegistryEntry:
    course_code: str
    course_name: str
    semester_workspace: str
    course_folder_name: str


def load_registry(path: Path) -> list[RegistryEntry]:
    with path.open(newline="", encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        if reader.fieldnames is None:
            raise SystemExit("Registry CSV has no header row.")
        required = {"course_code", "course_name", "semester_workspace", "course_folder_name"}
        missing = required - set(reader.fieldnames)
        if missing:
            raise SystemExit(f"Registry missing required columns: {', '.join(sorted(missing))}")

        rows: list[RegistryEntry] = []
        for row in reader:
            course_code = (row.get("course_code") or "").strip()
            if not course_code:
                continue
            rows.append(
                RegistryEntry(
                    course_code=course_code,
                    course_name=(row.get("course_name") or "").strip(),
                    semester_workspace=(row.get("semester_workspace") or "").strip(),
                    course_folder_name=(row.get("course_folder_name") or "").strip(),
                )
            )
        return rows


def load_existing_csv(path: Path) -> dict[tuple[str, str], str]:
    if not path.is_file():
        return {}
    with path.open(newline="", encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        if reader.fieldnames is None:
            return {}
        out: dict[tuple[str, str], str] = {}
        for row in reader:
            key = ((row.get("semester_workspace") or "").strip(), (row.get("course_folder_name") or "").strip())
            if not all(key):
                continue
            out[key] = (row.get("expected_count") or "").strip()
        return out


def load_existing_json(path: Path) -> dict[str, dict[tuple[str, str], int]]:
    if not path.is_file():
        return {}
    data = json.loads(path.read_text(encoding="utf-8"))
    out: dict[str, dict[tuple[str, str], int]] = {}
    for category, course_map in data.items():
        if category not in CATEGORY_TO_CSV or not isinstance(course_map, dict):
            continue
        out[category] = {}
        for key, value in course_map.items():
            if "/" not in key:
                continue
            semester, course = key.split("/", 1)
            try:
                out[category][(semester, course)] = int(value)
            except (TypeError, ValueError):
                continue
    return out


def sync_category_csv(
    category: str,
    entries: list[RegistryEntry],
    *,
    counts_dir: Path,
    legacy_json_values: dict[str, dict[tuple[str, str], int]],
) -> Path:
    counts_dir.mkdir(parents=True, exist_ok=True)
    csv_path = counts_dir / CATEGORY_TO_CSV[category]
    existing_values = load_existing_csv(csv_path)
    legacy_values = legacy_json_values.get(category, {})

    with csv_path.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=CSV_HEADERS)
        writer.writeheader()
        for entry in entries:
            key = (entry.semester_workspace, entry.course_folder_name)
            expected_count = existing_values.get(key, "")
            if expected_count == "" and key in legacy_values:
                expected_count = str(legacy_values[key])
            writer.writerow(
                {
                    "semester_workspace": entry.semester_workspace,
                    "course_code": entry.course_code,
                    "course_folder_name": entry.course_folder_name,
                    "expected_count": expected_count,
                }
            )
    return csv_path


def build_json_from_csvs(counts_dir: Path) -> dict[str, dict[str, int]]:
    output: dict[str, dict[str, int]] = {}
    for category, filename in CATEGORY_TO_CSV.items():
        csv_path = counts_dir / filename
        output[category] = {}
        if not csv_path.is_file():
            continue
        with csv_path.open(newline="", encoding="utf-8-sig") as f:
            reader = csv.DictReader(f)
            for row in reader:
                semester = (row.get("semester_workspace") or "").strip()
                course_folder = (row.get("course_folder_name") or "").strip()
                raw_count = (row.get("expected_count") or "").strip()
                if not semester or not course_folder:
                    continue
                try:
                    count = int(raw_count) if raw_count else 0
                except ValueError:
                    count = 0
                output[category][f"{semester}/{course_folder}"] = count
    return output


def sync_expected_counts(
    *,
    registry_path: Path = DEFAULT_REGISTRY,
    counts_dir: Path = DEFAULT_COUNTS_DIR,
    output_json: Path = DEFAULT_JSON,
    output_tex: Path = DEFAULT_TEX,
) -> dict[str, object]:
    entries = load_registry(registry_path)
    legacy_json_values = load_existing_json(output_json)
    written_csvs = [
        sync_category_csv(
            category,
            entries,
            counts_dir=counts_dir,
            legacy_json_values=legacy_json_values,
        )
        for category in CATEGORY_TO_CSV
    ]
    data = build_json_from_csvs(counts_dir)
    output_json.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
    write_expected_counts(output_json, output_tex)
    return {
        "csv_files": written_csvs,
        "json_file": output_json,
        "tex_file": output_tex,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Sync expected-count CSVs from course registry")
    parser.add_argument("--registry", default=str(DEFAULT_REGISTRY), help="Path to course registry CSV")
    parser.add_argument("--counts-dir", default=str(DEFAULT_COUNTS_DIR), help="Directory containing per-category CSVs")
    parser.add_argument("--output-json", default=str(DEFAULT_JSON), help="Derived expected_counts.json output path")
    parser.add_argument("--output-tex", default=str(DEFAULT_TEX), help="Derived expected_counts.tex output path")
    args = parser.parse_args()

    result = sync_expected_counts(
        registry_path=Path(args.registry).resolve(),
        counts_dir=Path(args.counts_dir).resolve(),
        output_json=Path(args.output_json).resolve(),
        output_tex=Path(args.output_tex).resolve(),
    )
    for path in result["csv_files"]:
        print(path)
    print(result["json_file"])
    print(result["tex_file"])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
