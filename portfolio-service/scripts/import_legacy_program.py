#!/usr/bin/env python3
"""Import a legacy ABET course tree into current instructor-package zips."""

from __future__ import annotations

import argparse
import csv
import os
from dataclasses import dataclass
from pathlib import Path
import re
import shutil
import tempfile
import zipfile

from expected_counts_manager import sync_expected_counts


LEGACY_BUILD_ARTIFACT_SUFFIXES = {
    ".aux",
    ".log",
    ".out",
    ".fdb_latexmk",
    ".fls",
    ".synctex.gz",
}


@dataclass
class CourseRecord:
    course_code: str
    course_name: str
    semester_workspace: str
    course_folder_name: str
    source_dir: Path
    zip_path: Path
    warnings: list[str]


def clean_tex(text: str) -> str:
    text = text.replace(r"\#", "#").replace(r"\%", "%").replace(r"\&", "&").replace("~", " ")
    text = re.sub(r"\s+", " ", text)
    return text.strip()


def extract_newcommand(content: str, name: str) -> str:
    match = re.search(rf"\\newcommand\{{\\{re.escape(name)}\}}\{{(.*?)\}}", content, re.DOTALL)
    if not match:
        raise ValueError(f"Missing \\newcommand for {name}")
    return clean_tex(match.group(1))


def extract_syllabus_label(content: str, label: str) -> str:
    pattern = re.compile(
        rf"\\noindent\\textbf\{{{re.escape(label)}:\}}\s*(.*?)\s*(?=\\vspace\{{10pt\}}|\\FieldBlock|\\FieldEnumBlock|% --- CLO)",
        re.DOTALL,
    )
    match = pattern.search(content)
    return clean_tex(match.group(1)) if match else ""


def extract_field_block(content: str, label: str) -> str:
    match = re.search(rf"\\FieldBlock\{{{re.escape(label)}\}}\{{(.*?)\}}", content, re.DOTALL)
    return clean_tex(match.group(1)) if match else ""


def extract_field_enum_items(content: str, label: str) -> list[str]:
    marker = f"\\FieldEnumBlock{{{label}}}{{%"
    start = content.find(marker)
    if start == -1:
        return []
    start += len(marker)
    end = content.find("\n}", start)
    if end == -1:
        return []
    block = content[start:end]
    items: list[str] = []
    for line in block.splitlines():
        line = line.strip()
        if line.startswith(r"\item "):
            items.append(clean_tex(line.removeprefix(r"\item ")))
    return items


def extract_prepared_by_and_date(content: str) -> tuple[str, str]:
    match = re.search(
        r"\\noindent\s+\\textbf\{Prepared by:\}\s*(.*?)\s*\\hfill\s*\\textbf\{Date:\}\s*(.*)$",
        content,
        re.DOTALL,
    )
    if not match:
        return "", ""
    return clean_tex(match.group(1)), clean_tex(match.group(2))


def extract_ice_outcome_columns(content: str, clo_count: int) -> dict[str, str]:
    results = {f"ice_outcome_{idx}": "" for idx in range(1, 8)}
    row_pattern = re.compile(r"\\textbf\{([1-7])\}\s*&\s*(.*?)\\\\\s*\\hline", re.DOTALL)
    for outcome_number, cells_blob in row_pattern.findall(content):
        parts = [clean_tex(part) for part in cells_blob.split("&")]
        if len(parts) < 2:
            continue
        comments = parts[-1]
        clo_cells = parts[:-1][:clo_count]
        entries: list[str] = []
        for index, value in enumerate(clo_cells, start=1):
            if value and value not in {"-", "–"}:
                entries.append(f"CLO{index}={value}")
        if comments:
            entries.append(f"Comments={comments}")
        results[f"ice_outcome_{outcome_number}"] = ",".join(entries)
    return results


def parse_schedule_rows(content: str) -> list[dict[str, str]]:
    rows: list[dict[str, str]] = []
    for week, topic in re.findall(r"\\textbf\{Week\s+(\d+):\}\s*&\s*(.*?)\\\\", content):
        rows.append({"week": week, "topic": clean_tex(topic)})
    return rows


def parse_grading_rows(content: str) -> list[dict[str, str]]:
    rows: list[dict[str, str]] = []
    for component, weight in re.findall(r"\\textbf\{(.*?)\}\s*&\s*\\textcolor\{gray\}\{\(([\d.]+)\\%\)\}", content):
        rows.append({"component": clean_tex(component), "weight_percent": weight})
    return rows


def write_single_row_csv(path: Path, fieldnames: list[str], row: dict[str, str]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerow(row)


def write_multi_row_csv(path: Path, fieldnames: list[str], rows: list[dict[str, str]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(rows)


def strip_junk_files(root: Path) -> None:
    for path in root.rglob("*"):
        if path.name == ".DS_Store":
            path.unlink()
        elif path.is_file() and path.suffix in LEGACY_BUILD_ARTIFACT_SUFFIXES:
            path.unlink()


def build_identity_csv(course_dir: Path, meta_content: str) -> None:
    row = {
        "course_name": extract_newcommand(meta_content, "CourseName"),
        "course_code": extract_newcommand(meta_content, "CourseCode"),
        "course_instructors": extract_newcommand(meta_content, "CourseInstructors"),
        "course_instructors_short": extract_newcommand(meta_content, "CourseInstructorsShort"),
        "program_name": extract_newcommand(meta_content, "ProgramName"),
        "school_name": extract_newcommand(meta_content, "SchoolName"),
        "university_name": extract_newcommand(meta_content, "UniversityName"),
        "semester_label": extract_newcommand(meta_content, "SemesterLabel"),
    }
    write_single_row_csv(
        course_dir / "00-meta" / "instructor_course_identity_template.csv",
        [
            "course_name",
            "course_code",
            "course_instructors",
            "course_instructors_short",
            "program_name",
            "school_name",
            "university_name",
            "semester_label",
        ],
        row,
    )


def build_syllabus_csv(course_dir: Path, syllabus_content: str, meta_content: str) -> None:
    outcomes = extract_field_enum_items(syllabus_content, "Course learning outcomes/expected performance criteria")
    clo_count = len(outcomes)
    ice_columns = extract_ice_outcome_columns(syllabus_content, clo_count)
    row = {
        "course_code": extract_newcommand(meta_content, "CourseCode"),
        "course_name": extract_newcommand(meta_content, "CourseName"),
        "required_or_elective": extract_syllabus_label(syllabus_content, "Required or Elective Course"),
        "catalog_description": extract_syllabus_label(syllabus_content, "Catalog Description"),
        "prerequisites": extract_syllabus_label(syllabus_content, "Prerequisites"),
        "categorization_of_credits": extract_syllabus_label(syllabus_content, "Categorization of Credits"),
        "instructors": extract_field_block(syllabus_content, "Instructors"),
        "textbooks_and_other_required_material": "\n".join(extract_field_enum_items(syllabus_content, "Textbooks and other required material")),
        "course_learning_outcomes_expected_performance_criteria": "\n".join(outcomes),
        "topics_covered": "\n".join(extract_field_enum_items(syllabus_content, "Topics covered")),
        "class_laboratory_schedule": extract_syllabus_label(syllabus_content, "Class/laboratory schedule"),
        "prepared_by": "",
        "prepared_date": "",
    }
    row.update(ice_columns)
    row["prepared_by"], row["prepared_date"] = extract_prepared_by_and_date(syllabus_content)
    write_single_row_csv(
        course_dir / "01-syllabus" / "instructor_syllabus_abet_reference.csv",
        [
            "course_code",
            "course_name",
            "required_or_elective",
            "catalog_description",
            "prerequisites",
            "categorization_of_credits",
            "instructors",
            "textbooks_and_other_required_material",
            "course_learning_outcomes_expected_performance_criteria",
            "topics_covered",
            "class_laboratory_schedule",
            "ice_outcome_1",
            "ice_outcome_2",
            "ice_outcome_3",
            "ice_outcome_4",
            "ice_outcome_5",
            "ice_outcome_6",
            "ice_outcome_7",
            "prepared_by",
            "prepared_date",
        ],
        row,
    )


def build_schedule_csv(course_dir: Path, schedule_content: str) -> None:
    write_multi_row_csv(
        course_dir / "03-schedule" / "instructor_course_schedule_template.csv",
        ["week", "topic"],
        parse_schedule_rows(schedule_content),
    )


def build_grading_csv(course_dir: Path, grading_content: str) -> None:
    write_multi_row_csv(
        course_dir / "90-grades" / "instructor_course_grading_template.csv",
        ["component", "weight_percent"],
        parse_grading_rows(grading_content),
    )


def zip_course_folder(course_dir: Path, zip_path: Path) -> None:
    zip_path.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(zip_path, "w", compression=zipfile.ZIP_DEFLATED) as zf:
        for path in sorted(course_dir.rglob("*")):
            if path.is_dir():
                continue
            arcname = Path(course_dir.name) / path.relative_to(course_dir)
            zf.write(path, arcname)


def discover_legacy_courses(source_root: Path) -> list[tuple[str, Path]]:
    courses: list[tuple[str, Path]] = []
    for semester_dir in sorted(source_root.glob("ABET-Course-Display-sem*")):
        if not semester_dir.is_dir():
            continue
        for course_dir in sorted(p for p in semester_dir.iterdir() if p.is_dir() and not p.name.startswith(".")):
            if (course_dir / "00-meta" / "course-meta.tex").is_file():
                courses.append((semester_dir.name, course_dir))
    return courses


def import_course(semester_workspace: str, source_course_dir: Path, output_zips_dir: Path) -> CourseRecord:
    warnings: list[str] = []
    with tempfile.TemporaryDirectory(prefix=f"legacy-{source_course_dir.name}-") as tmpdir:
        temp_course_dir = Path(tmpdir) / source_course_dir.name
        shutil.copytree(source_course_dir, temp_course_dir)
        strip_junk_files(temp_course_dir)

        meta_content = (temp_course_dir / "00-meta" / "course-meta.tex").read_text(encoding="utf-8")
        syllabus_content = (temp_course_dir / "01-syllabus" / "syllabus-abet.tex").read_text(encoding="utf-8")
        schedule_content = (temp_course_dir / "03-schedule" / "course-schedule.tex").read_text(encoding="utf-8")
        grading_path = temp_course_dir / "90-grades" / "course-grading.tex"
        grading_content = grading_path.read_text(encoding="utf-8") if grading_path.is_file() else ""
        if not grading_content:
            warnings.append("Missing 90-grades/course-grading.tex; grading CSV left empty.")

        build_identity_csv(temp_course_dir, meta_content)
        build_syllabus_csv(temp_course_dir, syllabus_content, meta_content)
        build_schedule_csv(temp_course_dir, schedule_content)
        build_grading_csv(temp_course_dir, grading_content)

        course_code = extract_newcommand(meta_content, "CourseCode")
        course_name = extract_newcommand(meta_content, "CourseName")
        zip_path = output_zips_dir / f"{temp_course_dir.name}.zip"
        zip_course_folder(temp_course_dir, zip_path)

    return CourseRecord(
        course_code=course_code,
        course_name=course_name,
        semester_workspace=semester_workspace,
        course_folder_name=source_course_dir.name,
        source_dir=source_course_dir,
        zip_path=zip_path,
        warnings=warnings,
    )


def write_registry(path: Path, records: list[CourseRecord]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(
            f,
            fieldnames=["course_code", "course_name", "semester_workspace", "course_folder_name"],
        )
        writer.writeheader()
        for record in records:
            writer.writerow(
                {
                    "course_code": record.course_code,
                    "course_name": record.course_name,
                    "semester_workspace": record.semester_workspace,
                    "course_folder_name": record.course_folder_name,
                }
            )


def write_report(path: Path, records: list[CourseRecord]) -> None:
    lines = [
        "# Legacy Import Report",
        "",
        f"- Imported courses: {len(records)}",
        "",
        "| Course Code | Course Folder | Semester Workspace | Zip Path | Warnings |",
        "| --- | --- | --- | --- | --- |",
    ]
    for record in records:
        warning_text = "<br>".join(record.warnings) if record.warnings else ""
        lines.append(
            f"| {record.course_code} | {record.course_folder_name} | {record.semester_workspace} | `{record.zip_path}` | {warning_text} |"
        )
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Import a legacy ABET program tree into current zips")
    parser.add_argument("source_root", help="Legacy program root containing ABET-Course-Display-sem* folders")
    parser.add_argument("workspace_root", help="Current program workspace created by create_program_workspace.py")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    source_root = Path(args.source_root).resolve()
    workspace_root = Path(args.workspace_root).resolve()
    if not source_root.is_dir():
        raise SystemExit(f"Missing source root: {source_root}")
    if not (workspace_root / "config" / "staff" / "course_registry.csv").is_file():
        raise SystemExit(f"Workspace does not look initialized: {workspace_root}")

    output_zips_dir = workspace_root / "source-zips"
    if output_zips_dir.exists():
        shutil.rmtree(output_zips_dir)
    output_zips_dir.mkdir(parents=True, exist_ok=True)

    records: list[CourseRecord] = []
    for semester_workspace, course_dir in discover_legacy_courses(source_root):
        records.append(import_course(semester_workspace, course_dir, output_zips_dir))

    write_registry(workspace_root / "config" / "staff" / "course_registry.csv", records)
    os.environ["COURSEDISPLAY_WORKSPACE_ROOT"] = str(workspace_root)
    sync_expected_counts(
        registry_path=workspace_root / "config" / "staff" / "course_registry.csv",
        counts_dir=workspace_root / "config" / "staff" / "expected_counts",
        output_json=workspace_root / "config" / "generated" / "expected_counts.json",
        output_tex=workspace_root / "config" / "generated" / "expected_counts.tex",
    )
    write_report(workspace_root / "legacy-import-report.md", records)

    print(workspace_root)
    print(output_zips_dir)
    print(workspace_root / "legacy-import-report.md")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
