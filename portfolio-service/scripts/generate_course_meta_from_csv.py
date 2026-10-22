#!/usr/bin/env python3
"""Generate course-meta.tex files from a CSV template."""

from __future__ import annotations

import argparse
import csv
from pathlib import Path

REQUIRED_COLUMNS = [
    "course_name",
    "course_code",
    "course_instructors",
    "course_instructors_short",
    "program_name",
    "school_name",
    "university_name",
    "semester_label",
]

OPTIONAL_COLUMNS = ["course_folder_name", "cover_image"]


def tex_escape(text: str) -> str:
    replacements = {
        "\\": r"\textbackslash{}",
        "&": r"\&",
        "%": r"\%",
        "$": r"\$",
        "#": r"\#",
        "_": r"\_",
        "{": r"\{",
        "}": r"\}",
    }
    out = text
    for k, v in replacements.items():
        out = out.replace(k, v)
    return out


def make_course_folder_name(row: dict[str, str]) -> str:
    explicit = (row.get("course_folder_name") or "").strip()
    if explicit:
        return explicit
    code = (row.get("course_code") or "").strip()
    name = (row.get("course_name") or "").strip()
    if not code or not name:
        raise SystemExit("course_code and course_name are required to derive course_folder_name")
    normalized_name = "-".join(name.split())
    return f"{code}-{normalized_name}"


def build_meta_tex(row: dict[str, str]) -> str:
    folder = make_course_folder_name(row)
    return "\n".join(
        [
            "% ===== Per-course metadata & build flags =====",
            "",
            "% FAST COMPILE: set true while editing; false for final",
            r"\newif\iffastcompile",
            r"\fastcompilefalse           % final",
            r"%\fastcompiletrue          % dev fast mode",
            "",
            "% Course identity",
            rf"\newcommand{{\CourseName}}{{{tex_escape(row['course_name'].strip())}}}",
            rf"\newcommand{{\CourseCode}}{{{tex_escape(row['course_code'].strip())}}}",
            rf"\newcommand{{\CourseInstructors}}{{{tex_escape(row['course_instructors'].strip())}}}",
            rf"\newcommand{{\CourseInstructorsShort}}{{{tex_escape(row['course_instructors_short'].strip())}}}",
            rf"\newcommand{{\ProgramName}}{{{tex_escape(row['program_name'].strip())}}}",
            rf"\newcommand{{\SchoolName}}{{{tex_escape(row['school_name'].strip())}}}",
            rf"\newcommand{{\UniversityName}}{{{tex_escape(row['university_name'].strip())}}}",
            rf"\newcommand{{\SemesterLabel}}{{{tex_escape(row['semester_label'].strip())}}}",
            "",
            "% Root path for ALL course files",
            rf"\newcommand{{\CourseRoot}}{{course_documents/{tex_escape(folder)}}}",
            "",
            "% Cover image path relative to project root",
            rf"\newcommand{{\CoverImage}}{{{tex_escape(row.get('cover_image', 'cover.pdf').strip() or 'cover.pdf')}}}",
            "",
        ]
    )


def load_rows(csv_path: Path) -> list[dict[str, str]]:
    with csv_path.open(newline="", encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        if reader.fieldnames is None:
            raise SystemExit("CSV has no header row.")

        missing_columns = [c for c in REQUIRED_COLUMNS if c not in reader.fieldnames]
        if missing_columns:
            raise SystemExit(f"Missing required columns: {', '.join(missing_columns)}")

        rows: list[dict[str, str]] = []
        for line_num, row in enumerate(reader, start=2):
            if not any((v or "").strip() for v in row.values()):
                continue

            missing_values = [c for c in REQUIRED_COLUMNS if not (row.get(c) or "").strip()]
            if missing_values:
                raise SystemExit(
                    f"Row {line_num} missing required values: {', '.join(missing_values)}"
                )
            rows.append(row)
        return rows


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Generate course-meta.tex files from instructor identity CSV."
    )
    parser.add_argument("csv_file", help="Path to identity CSV file")
    parser.add_argument(
        "--output-root",
        default="generated_course_meta",
        help="Folder where generated course-meta files are written",
    )
    args = parser.parse_args()

    csv_path = Path(args.csv_file).resolve()
    out_root = Path(args.output_root).resolve()
    out_root.mkdir(parents=True, exist_ok=True)

    rows = load_rows(csv_path)
    count = 0
    for row in rows:
        folder = make_course_folder_name(row)
        out_dir = out_root / folder / "00-meta"
        out_dir.mkdir(parents=True, exist_ok=True)
        out_file = out_dir / "course-meta.tex"
        out_file.write_text(build_meta_tex(row), encoding="utf-8")
        count += 1

    print(f"Generated {count} course-meta files in: {out_root}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
