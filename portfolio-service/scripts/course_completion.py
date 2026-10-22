#!/usr/bin/env python3
"""Generate a completion summary for every course folder."""

from __future__ import annotations

import argparse
from collections import defaultdict
import csv
from dataclasses import dataclass
from datetime import date
import json
from pathlib import Path
from typing import Dict, Iterable, List, Tuple, Mapping, Set
from collections import OrderedDict

PLACEHOLDER_NAMES = {"dummy.txt"}
OVERRIDE_CATEGORIES = {
    "10-lectures",
    "20-assignments",
    "30-quizzes",
    "40-exams",
}
EXPECTED_COUNT_MULTIPLIERS = {
    "20-assignments": 5,
    "30-quizzes": 5,
}
FIXED_EXPECTED_COUNTS = {
    "90-grades": 5,
}
DEFAULT_EXPECTED_COUNTS_PATH = Path("config") / "generated" / "expected_counts.json"
DEFAULT_COURSE_NOTES_CSV = Path("config") / "staff" / "course_notes.csv"
DEFAULT_COURSE_NOTES_JSON = Path("config") / "staff" / "course_notes.json"
DEFAULT_COURSE_REGISTRY_CSV = Path("config") / "staff" / "course_registry.csv"


def is_placeholder(path: Path) -> bool:
    """Return True when the file should not be counted as a document."""
    if not path.is_file():
        return True
    name = path.name
    if name in PLACEHOLDER_NAMES:
        return True
    if name.startswith("."):
        return True
    return False


@dataclass(frozen=True)
class CourseEntry:
    semester: str
    course: str
    counts: Dict[str, int]
    files: Dict[str, List[str]]


@dataclass(frozen=True)
class CompletionResult:
    entry: CourseEntry
    completion: float
    total_actual_documents: int
    total_expected_documents: int
    details: List[Tuple[str, int, int, float]]


def load_registry_course_keys(path: Path | None) -> Set[Tuple[str, str]] | None:
    """Load valid (semester_workspace, course_folder_name) pairs from the registry."""
    if path is None or not path.is_file():
        return None

    keys: Set[Tuple[str, str]] = set()
    with path.open(newline="", encoding="utf-8-sig") as handle:
        reader = csv.DictReader(handle)
        if reader.fieldnames is None:
            return keys
        for row in reader:
            semester = (row.get("semester_workspace") or "").strip()
            course = (row.get("course_folder_name") or "").strip()
            if semester and course:
                keys.add((semester, course))
    return keys


def iter_semester_workspaces(root: Path) -> Iterable[Path]:
    """Yield real semester workspaces, independent of their naming convention."""
    if not root.is_dir():
        return
    for semester_dir in sorted(
        (p for p in root.iterdir() if p.is_dir() and not p.name.startswith(".")),
        key=lambda p: p.name,
    ):
        if (semester_dir / "course_documents").is_dir():
            yield semester_dir


def collect_course_data(
    root: Path,
    registry_path: Path | None = None,
) -> Tuple[List[CourseEntry], Dict[str, int]]:
    """Collect per-category document counts for each course."""
    course_entries: List[CourseEntry] = []
    category_max: Dict[str, int] = defaultdict(int)
    registry_keys = load_registry_course_keys(registry_path)

    for semester_dir in iter_semester_workspaces(root):
        course_docs = semester_dir / "course_documents"

        for course_dir in sorted((p for p in course_docs.iterdir() if p.is_dir()), key=lambda p: p.name):
            if registry_keys is not None and (semester_dir.name, course_dir.name) not in registry_keys:
                continue
            counts: Dict[str, int] = {}
            files: Dict[str, List[str]] = {}
            for category_dir in sorted((p for p in course_dir.iterdir() if p.is_dir()), key=lambda p: p.name):
                actual_files = sorted(
                    str(file_path.relative_to(category_dir))
                    for file_path in category_dir.rglob("*")
                    if not is_placeholder(file_path)
                )
                if category_dir.name == "01-syllabus":
                    # Treat the syllabus as complete when the ABET-specific syllabus file exists.
                    count = 1 if (category_dir / "syllabus-abet.tex").is_file() else 0
                elif category_dir.name == "02-textbook":
                    # Any non-placeholder file in the textbook folder counts as completion.
                    has_content = bool(actual_files)
                    count = 1 if has_content else 0
                else:
                    count = len(actual_files)
                counts[category_dir.name] = count
                files[category_dir.name] = actual_files
                if count > category_max[category_dir.name]:
                    category_max[category_dir.name] = count

            course_entries.append(CourseEntry(semester=semester_dir.name, course=course_dir.name, counts=counts, files=files))

    return course_entries, dict(category_max)


def compute_completion(
    entry: CourseEntry,
    categories: Iterable[str],
    expected_by_category: Dict[str, int],
    overrides: Mapping[str, Dict[Tuple[str, str], int]],
) -> Tuple[float, int, int, List[Tuple[str, int, int, float]]]:
    """Compute the completion percentage for a course."""
    details: List[Tuple[str, int, int, float]] = []
    ratios: List[float] = []
    total_actual = 0
    total_expected = 0

    for category in categories:
        expected = expected_for(entry, category, expected_by_category, overrides)
        if expected <= 0:
            continue
        actual = entry.counts.get(category, 0)
        ratio = min(actual / expected, 1.0) if expected else 1.0
        details.append((category, actual, expected, ratio))
        ratios.append(ratio)
        total_actual += actual
        total_expected += expected

    completion = 100.0 * sum(ratios) / len(ratios) if ratios else 0.0
    return completion, total_actual, total_expected, details


def format_missing(details: Iterable[Tuple[str, int, int, float]]) -> str:
    """Render a short description of categories that are not yet complete."""
    parts = []
    for category, actual, expected, ratio in details:
        if ratio >= 1.0:
            continue
        parts.append(f"{category} ({actual}/{expected})")
    return ", ".join(parts) if parts else "—"


def expected_for(
    entry: CourseEntry,
    category: str,
    default_expected: Mapping[str, int],
    overrides: Mapping[str, Dict[Tuple[str, str], int]],
) -> int:
    """Return the expected document count for a given course/category."""
    if category in FIXED_EXPECTED_COUNTS:
        return FIXED_EXPECTED_COUNTS[category]
    course_key = (entry.semester, entry.course)
    override_map = overrides.get(category, {})
    if course_key in override_map:
        base_expected = override_map[course_key]
    else:
        base_expected = default_expected.get(category, 0)
    return base_expected * EXPECTED_COUNT_MULTIPLIERS.get(category, 1)


def load_expected_counts(path: Path) -> Dict[str, Dict[Tuple[str, str], int]]:
    """Load per-course expected document counts for override categories."""
    expected: Dict[str, Dict[Tuple[str, str], int]] = {category: {} for category in OVERRIDE_CATEGORIES}
    if not path.is_file():
        return expected

    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise SystemExit(f"Failed to parse expected counts file {path}: {exc}") from exc

    for category, course_map in data.items():
        if category not in OVERRIDE_CATEGORIES:
            continue
        target = expected.setdefault(category, {})
        if not isinstance(course_map, dict):
            continue
        for course_key, value in course_map.items():
            if not isinstance(course_key, str):
                continue
            if "/" not in course_key:
                continue
            semester, course = course_key.split("/", 1)
            try:
                count = int(value)
            except (TypeError, ValueError):
                continue
            target[(semester, course)] = count

    return expected


def load_course_notes_json(path: Path) -> Dict[Tuple[str, str], Dict[str, str]]:
    """Load optional per-course notes from JSON."""
    try:
        raw = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise SystemExit(f"Failed to parse course notes file {path}: {exc}") from exc

    notes: Dict[Tuple[str, str], Dict[str, str]] = {}
    for key, payload in raw.items():
        if not isinstance(key, str) or "/" not in key:
            continue
        semester, course = key.split("/", 1)
        if not isinstance(payload, dict):
            continue
        entry: Dict[str, str] = {}
        link = payload.get("link")
        if isinstance(link, str) and link:
            entry["link"] = link
        comment = payload.get("comment")
        if isinstance(comment, str) and comment:
            entry["comment"] = comment
        if entry:
            notes[(semester, course)] = entry
    return notes


def load_course_notes_csv(path: Path) -> Dict[Tuple[str, str], Dict[str, str]]:
    """Load optional per-course notes from a staff-editable CSV."""
    notes: Dict[Tuple[str, str], Dict[str, str]] = {}
    try:
        with path.open(newline="", encoding="utf-8-sig") as f:
            reader = csv.DictReader(f)
            if reader.fieldnames is None:
                return {}
            for row in reader:
                semester = (row.get("semester_workspace") or "").strip()
                course = (row.get("course_folder_name") or "").strip()
                if not semester or not course:
                    continue
                entry: Dict[str, str] = {}
                link = (row.get("link") or "").strip()
                comment = (row.get("comment") or "").strip()
                if link:
                    entry["link"] = link
                if comment:
                    entry["comment"] = comment
                if entry:
                    notes[(semester, course)] = entry
    except csv.Error as exc:
        raise SystemExit(f"Failed to parse course notes CSV {path}: {exc}") from exc
    return notes


def load_course_notes(path: Path) -> Dict[Tuple[str, str], Dict[str, str]]:
    """Load optional per-course notes from CSV or JSON."""
    if not path.is_file():
        return {}
    if path.suffix.lower() == ".csv":
        return load_course_notes_csv(path)
    return load_course_notes_json(path)


def compute_results(
    root: Path,
    expected_counts_path: Path,
    registry_path: Path | None = None,
) -> Tuple[List[CompletionResult], Dict[str, int], Dict[str, Dict[Tuple[str, str], int]]]:
    entries, category_max = collect_course_data(root, registry_path)
    overrides = load_expected_counts(expected_counts_path)
    categories = sorted(
        {
            *(cat for cat, maximum in category_max.items() if maximum > 0),
            *(cat for cat, mapping in overrides.items() if mapping),
            *FIXED_EXPECTED_COUNTS.keys(),
        }
    )
    entries.sort(key=lambda item: (item.semester, item.course))

    results: List[CompletionResult] = []
    for entry in entries:
        completion, total_actual, total_expected, details = compute_completion(entry, categories, category_max, overrides)
        results.append(
            CompletionResult(
                entry=entry,
                completion=completion,
                total_actual_documents=total_actual,
                total_expected_documents=total_expected,
                details=details,
            )
        )
    return results, category_max, overrides


def build_report(
    root: Path,
    output_path: Path,
    expected_counts_path: Path,
    registry_path: Path | None = None,
) -> List[CompletionResult]:
    """Generate the detailed markdown report and return per-course results."""
    results, _category_max, _overrides = compute_results(root, expected_counts_path, registry_path)

    lines: List[str] = []
    lines.append("# Course Completion Report")
    lines.append("")
    lines.append(f"Generated on {date.today().isoformat()}.")
    lines.append("")
    lines.append("## Methodology")
    lines.append(
        "1. Count non-placeholder documents in each category for every course folder."
    )
    lines.append(
        "2. `01-syllabus` counts as complete when `syllabus-abet.tex` exists; `02-textbook` counts as complete "
        "when any non-placeholder file is present."
    )
    lines.append(
        "3. Apply fixed expectations where defined (`90-grades` expects 5 documents)."
    )
    lines.append(
        "4. Determine default expectations for other categories from the maximum counts seen across all courses."
    )
    lines.append(
        "5. Override expectations for `10-lectures`, `20-assignments`, `30-quizzes`, and `40-exams` by "
        "editing the per-category CSV files in `config/staff/expected_counts/`; "
        "`config/generated/expected_counts.json` is derived automatically from those CSVs."
    )
    lines.append(
        "6. Compute completion as the average of per-category ratios (actual ÷ expected, capped at 100%). "
        "Categories with zero expected documents are skipped."
    )
    lines.append(
        "Assignment expectations are expanded to 5 required files per assignment: "
        "`instructions`, `scores`, `high`, `average`, and `low`."
    )
    lines.append("")
    lines.append("## Summary")
    lines.append("")
    lines.append("| Semester | Course | Documents | Completion | Missing categories |")
    lines.append("| --- | --- | ---: | ---: | --- |")

    for result in results:
        entry = result.entry
        missing = format_missing(result.details)
        lines.append(
            f"| {entry.semester} | {entry.course} | {result.total_actual_documents}/{result.total_expected_documents} | {result.completion:5.1f}% | {missing} |"
        )

    lines.append("")
    lines.append("## Course Details")
    lines.append("")

    for result in results:
        entry = result.entry
        lines.append(f"### {entry.semester} / {entry.course}")
        lines.append("")
        lines.append(f"- Total documents: {result.total_actual_documents}/{result.total_expected_documents}")
        lines.append(f"- Completion: {result.completion:0.1f}%")
        missing = format_missing(result.details)
        lines.append(f"- Missing categories: {missing}")
        lines.append("")
        lines.append("| Category | Documents | Expected | Completion |")
        lines.append("| --- | ---: | ---: | ---: |")
        for category, actual, expected, ratio in result.details:
            lines.append(f"| {category} | {actual} | {expected} | {ratio * 100:5.1f}% |")
        lines.append("")

    output_path.write_text("\n".join(lines), encoding="utf-8")
    return results


def append_generic_slots(
    items: List[Tuple[str, bool]],
    expected_count: int,
    base_count: int,
    label_prefix: str,
) -> None:
    for index in range(base_count + 1, expected_count + 1):
        items.append((f"{label_prefix} {index:02}", False))


def expected_items_for_category(category: str, expected_count: int) -> List[Tuple[str, bool]]:
    items: List[Tuple[str, bool]] = []
    if expected_count <= 0:
        return items

    if category == "00-meta":
        base = [
            "instructor_course_identity_template.csv",
            "course-meta.tex",
        ]
        items.extend((name, True) for name in base[:expected_count])
        append_generic_slots(items, expected_count, len(base), "metadata document slot")
        return items

    if category == "01-syllabus":
        base = ["syllabus-abet.tex"]
        items.extend((name, True) for name in base[:expected_count])
        append_generic_slots(items, expected_count, len(base), "syllabus document slot")
        return items

    if category == "02-textbook":
        return [(f"textbook document slot {index:02}", False) for index in range(1, expected_count + 1)]

    if category == "03-schedule":
        base = [
            "instructor_course_schedule_template.csv",
            "course-schedule.tex",
        ]
        items.extend((name, True) for name in base[:expected_count])
        append_generic_slots(items, expected_count, len(base), "schedule document slot")
        return items

    if category == "10-lectures":
        return [(f"lecture document slot {index:02}", False) for index in range(1, expected_count + 1)]

    if category == "20-assignments":
        names = []
        for assignment in range(1, (expected_count // 5) + 1):
            for suffix in ["instructions", "scores", "high", "average", "low"]:
                names.append(f"asg{assignment:02}-{suffix}.pdf")
        return [(name, True) for name in names[:expected_count]]

    if category == "30-quizzes":
        names = []
        for quiz in range(1, (expected_count // 5) + 1):
            for suffix in ["instructions", "scores", "high", "average", "low"]:
                names.append(f"quiz{quiz:02}-{suffix}.pdf")
        return [(name, True) for name in names[:expected_count]]

    if category == "40-exams":
        names = []
        for exam_name in ["midterm", "final"]:
            for suffix in ["instructions", "scores", "high", "average", "low"]:
                names.append(f"{exam_name}-{suffix}.pdf")
        items.extend((name, True) for name in names[:expected_count])
        append_generic_slots(items, expected_count, len(names), "exam document slot")
        return items

    if category == "90-grades":
        base = [
            "instructor_course_grading_template.csv",
            "course-grading.tex",
            "overall-grades.pdf",
        ]
        items.extend((name, True) for name in base[:expected_count])
        append_generic_slots(items, expected_count, len(base), "grades document slot")
        return items

    return [(f"document slot {index:02}", False) for index in range(1, expected_count + 1)]


def detail_lines_for_result(result: CompletionResult) -> List[str]:
    entry = result.entry
    lines: List[str] = []
    lines.append(f"# Course Completion Detail: {entry.course}")
    lines.append("")
    lines.append(f"Generated on {date.today().isoformat()}.")
    lines.append("")
    lines.append(f"- Semester: {entry.semester}")
    lines.append(f"- Course: {entry.course}")
    lines.append(f"- Total documents: {result.total_actual_documents}/{result.total_expected_documents}")
    lines.append(f"- Completion: {result.completion:0.1f}%")
    lines.append("")
    lines.append(
        "Exact filenames are shown where the naming pattern is deterministic. Other categories use document slots."
    )
    lines.append("")

    for category, actual, expected, ratio in result.details:
        actual_files = list(entry.files.get(category, []))
        expected_items = expected_items_for_category(category, expected)
        actual_set = set(actual_files)
        consumed: set[str] = set()

        lines.append(f"## {category}")
        lines.append("")
        lines.append(f"- Category completion: {actual}/{expected} ({ratio * 100:0.1f}%)")
        lines.append("")
        lines.append("| Expected item | Status | Actual file |")
        lines.append("| --- | --- | --- |")

        for expected_item, exact in expected_items:
            if exact:
                present = expected_item in actual_set
                actual_name = expected_item if present else ""
                if present:
                    consumed.add(expected_item)
                lines.append(f"| `{expected_item}` | {'PRESENT' if present else 'MISSING'} | `{actual_name}` |")
                continue

            next_actual = next((name for name in actual_files if name not in consumed), None)
            if next_actual is not None:
                consumed.add(next_actual)
                lines.append(f"| `{expected_item}` | PRESENT | `{next_actual}` |")
            else:
                lines.append(f"| `{expected_item}` | MISSING | `` |")

        extra_files = [name for name in actual_files if name not in consumed]
        if extra_files:
            lines.append("")
            lines.append("Additional present files not matched to the expected list:")
            lines.append("")
            for name in extra_files:
                lines.append(f"- `{name}`")
        lines.append("")
    return lines


def build_detailed_report(
    detail_dir: Path,
    results: List[CompletionResult],
) -> None:
    if detail_dir.exists() and detail_dir.is_file():
        detail_dir.unlink()
    if detail_dir.exists():
        for path in sorted(detail_dir.glob("*")):
            if path.is_file():
                path.unlink()
    detail_dir.mkdir(parents=True, exist_ok=True)

    index_lines: List[str] = []
    index_lines.append("# Course Completion Detail Index")
    index_lines.append("")
    index_lines.append(f"Generated on {date.today().isoformat()}.")
    index_lines.append("")
    index_lines.append("| Semester | Course | Detail file |")
    index_lines.append("| --- | --- | --- |")

    for result in results:
        entry = result.entry
        course_filename = f"{entry.course}.md"
        course_path = detail_dir / course_filename
        course_path.write_text("\n".join(detail_lines_for_result(result)), encoding="utf-8")
        index_lines.append(f"| {entry.semester} | {entry.course} | `{course_filename}` |")

    (detail_dir / "index.md").write_text("\n".join(index_lines), encoding="utf-8")


def format_semester_name(raw_name: str) -> str:
    prefix = "ABET-Course-Display-"
    if raw_name.startswith(prefix):
        remainder = raw_name[len(prefix) :]
        if remainder.startswith("sem") and "y" in remainder:
            sem_year = remainder[3:]
            semester_part, _, year_part = sem_year.partition("y")
            if semester_part.isdigit() and year_part.isdigit():
                return f"Semester {int(semester_part)} Year {int(year_part)}"
        return remainder.replace("-", " ").title()
    return raw_name.replace("-", " ").title()


def build_summary_report(summary_path: Path, results: List[CompletionResult]) -> None:
    """Write a high-level summary report grouped by semester."""
    lines: List[str] = []
    lines.append("# Course Completion Summary")
    lines.append("")

    if not results:
        lines.append("No course data available.")
        summary_path.write_text("\n".join(lines), encoding="utf-8")
        return

    overall_completion = sum(result.completion for result in results) / len(results)
    lines.append(f"**Overall completion:** {overall_completion:0.1f}%")
    lines.append("")

    semester_map: Dict[str, List[CompletionResult]] = {}
    for result in results:
        semester_map.setdefault(result.entry.semester, []).append(result)

    semester_items = list(semester_map.items())
    for idx, (semester, semester_results) in enumerate(semester_items):
        display_name = format_semester_name(semester)
        semester_completion = sum(result.completion for result in semester_results) / len(semester_results)
        lines.append(f"## {display_name}")
        lines.append("")
        lines.append(f"- Semester completion: {semester_completion:0.1f}%")
        lines.append("")
        lines.append("| Course | Completion |")
        lines.append("| --- | ---: |")
        for idx, result in enumerate(semester_results):
            lines.append(f"| {result.entry.course} | {result.completion:5.1f}% |")
            if idx < len(semester_results) - 1:
                lines.append("| --- | ---: |")
        lines.append("")

    summary_path.write_text("\n".join(lines), encoding="utf-8")


def default_course_link(entry: CourseEntry) -> str:
    """Return default link to the compiled PDF for a course."""
    return f"runtime/output/course-pdfs/{entry.course}-main.pdf"


def build_pev_report(
    pev_path: Path,
    results: List[CompletionResult],
    notes: Dict[Tuple[str, str], Dict[str, str]],
) -> None:
    """Create a PEV-focused overview with links and comments."""
    lines: List[str] = []
    lines.append("# ABET PEV Course Display Overview")
    lines.append("")

    if not results:
        lines.append("No course data available.")
        pev_path.write_text("\n".join(lines), encoding="utf-8")
        return

    overall_completion = sum(result.completion for result in results) / len(results)
    lines.append(f"**Overall completion:** {overall_completion:0.1f}%")
    lines.append("")
    lines.append(
        "Each course row includes a direct link to the compiled binder. "
        "Courses below 100% completion list outstanding notes when provided."
    )
    lines.append("")

    semester_map: Dict[str, List[CompletionResult]] = {}
    for result in results:
        semester_map.setdefault(result.entry.semester, []).append(result)

    semester_items = list(semester_map.items())
    for idx, (semester, semester_results) in enumerate(semester_items):
        display_name = format_semester_name(semester)
        lines.append(f"## {display_name}")
        lines.append("")
        lines.append("| Course | Completion | Binder | Comment |")
        lines.append("| --- | ---: | --- | --- |")
        for course_idx, result in enumerate(semester_results):
            entry = result.entry
            note = notes.get((entry.semester, entry.course), {})
            link = note.get("link") or default_course_link(entry)
            comment = note.get("comment")
            if not comment and result.completion < 100.0:
                comment = "_No comment provided_"
            comment = comment or ""
            link_md = f"[Open]({link})"
            lines.append(
                f"| {entry.course} | {result.completion:5.1f}% | {link_md} | {comment} |"
            )
            if course_idx < len(semester_results) - 1:
                lines.append("| --- | ---: | --- | --- |")
        lines.append("")
        if idx < len(semester_items) - 1:
            lines.append("")
            lines.append("\\newpage")
            lines.append("")

    pev_path.write_text("\n".join(lines), encoding="utf-8")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate per-course completion percentages.")
    parser.add_argument(
        "--root",
        type=Path,
        default=Path("."),
        help="Project root that contains ABET course folders (default: current directory)",
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=Path("course_completion_report.md"),
        help="Where to write the markdown report (default: course_completion_report.md)",
    )
    parser.add_argument(
        "--summary-output",
        type=Path,
        default=Path("course_completion_summary.md"),
        help="Where to write the semester summary report (default: course_completion_summary.md)",
    )
    parser.add_argument(
        "--expected-counts",
        type=Path,
        default=DEFAULT_EXPECTED_COUNTS_PATH,
        help="JSON file that supplies per-course expected counts for lecture/assignment/quiz/exam folders",
    )
    parser.add_argument(
        "--pev-output",
        type=Path,
        default=Path("course_completion_pev.md"),
        help="Where to write the PEV overview report (default: course_completion_pev.md)",
    )
    parser.add_argument(
        "--course-notes",
        type=Path,
        default=DEFAULT_COURSE_NOTES_CSV,
        help="CSV or JSON file containing optional per-course links/comments for the PEV report",
    )
    parser.add_argument(
        "--course-registry",
        type=Path,
        default=DEFAULT_COURSE_REGISTRY_CSV,
        help="CSV file that maps valid course folders to semester workspaces",
    )
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    root = args.root.resolve()
    output_path = args.output.resolve()
    summary_output_path = args.summary_output.resolve()
    expected_counts_path = args.expected_counts if args.expected_counts.is_absolute() else (root / args.expected_counts)
    pev_output_path = args.pev_output.resolve()
    course_notes_path = args.course_notes if args.course_notes.is_absolute() else (root / args.course_notes)
    course_registry_path = args.course_registry if args.course_registry.is_absolute() else (root / args.course_registry)

    results, _category_max, _overrides = compute_results(root, expected_counts_path, course_registry_path)
    detailed_output_path = output_path.with_name("course_completion_detailed")
    build_report(root, output_path, expected_counts_path, course_registry_path)
    build_detailed_report(detailed_output_path, results)
    build_summary_report(summary_output_path, results)
    notes = load_course_notes(course_notes_path)
    build_pev_report(pev_output_path, results, notes)
    print(f"Report written to {output_path}")
    print(f"Detailed report written to {detailed_output_path}")
    print(f"Summary written to {summary_output_path}")
    print(f"PEV overview written to {pev_output_path}")


if __name__ == "__main__":
    main()
