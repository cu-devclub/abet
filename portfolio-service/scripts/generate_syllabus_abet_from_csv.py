#!/usr/bin/env python3
"""Generate 01-syllabus/syllabus-abet.tex from the instructor CSV."""

from __future__ import annotations

import argparse
import csv
import re
from pathlib import Path

BASE_FIELDS = [
    "course_code",
    "course_name",
    "required_or_elective",
    "catalog_description",
    "prerequisites",
    "categorization_of_credits",
    "instructors",
    "class_laboratory_schedule",
    "prepared_by",
    "prepared_date",
]

LIST_FIELDS = [
    "textbooks_and_other_required_material",
    "course_learning_outcomes_expected_performance_criteria",
    "topics_covered",
]

OUTCOME_FIELDS = [f"ice_outcome_{index}" for index in range(1, 8)]

TEMPLATE_SENTINELS = {
    "your_course_code",
    "your_course_name",
    "required_elective",
    "your_catalog_description",
}

ENTRY_RE = re.compile(r"^CLO\s*(\d+)\s*=\s*([SMWsmw]?)\s*$")
COMMENT_RE = re.compile(r"^Comments\s*=\s*(.*)$", re.IGNORECASE)
LEADING_ENUM_RE = re.compile(r"^\s*\d+[\.)]\s*")


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
    out = text.strip()
    for k, v in replacements.items():
        out = out.replace(k, v)
    return out


def strip_leading_enum(text: str) -> str:
    return LEADING_ENUM_RE.sub("", text.strip())


def split_piped_items(text: str) -> list[str]:
    return [strip_leading_enum(part) for part in text.split("|") if part.strip()]


def load_rows(csv_path: Path) -> list[dict[str, str]]:
    with csv_path.open(newline="", encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        if reader.fieldnames is None:
            raise SystemExit("CSV has no header row.")
        required = set(BASE_FIELDS + LIST_FIELDS + OUTCOME_FIELDS)
        missing = sorted(required - set(reader.fieldnames))
        if missing:
            raise SystemExit(f"Missing required columns: {', '.join(missing)}")

        rows: list[dict[str, str]] = []
        for row in reader:
            if not any((value or "").strip() for value in row.values()):
                continue
            code = (row.get("course_code") or "").strip()
            name = (row.get("course_name") or "").strip()
            if code in TEMPLATE_SENTINELS or name in TEMPLATE_SENTINELS:
                continue
            rows.append(row)
        if not rows:
            raise SystemExit("No non-template syllabus rows found in CSV.")
        return rows


def pick_first(rows: list[dict[str, str]], field: str, default: str = "") -> str:
    for row in rows:
        value = (row.get(field) or "").strip()
        if value:
            return value
    return default


def gather_list(rows: list[dict[str, str]], field: str, split_pipes: bool = False) -> list[str]:
    items: list[str] = []
    seen: set[str] = set()
    for row in rows:
        raw = (row.get(field) or "").strip()
        if not raw:
            continue
        candidates = split_piped_items(raw) if split_pipes else [strip_leading_enum(raw)]
        for item in candidates:
            if item in seen:
                continue
            seen.add(item)
            items.append(item)
    return items


def parse_outcome_mapping(rows: list[dict[str, str]]) -> tuple[int, dict[int, dict[int, str]], dict[int, str]]:
    clo_count = 0
    mappings: dict[int, dict[int, str]] = {index: {} for index in range(1, 8)}
    comments: dict[int, str] = {index: "" for index in range(1, 8)}

    for outcome_index, field in enumerate(OUTCOME_FIELDS, start=1):
        for row in rows:
            raw = (row.get(field) or "").strip()
            if not raw:
                continue
            comment_match = COMMENT_RE.match(raw)
            if comment_match:
                comments[outcome_index] = comment_match.group(1).strip()
                continue
            entry_match = ENTRY_RE.match(raw)
            if not entry_match:
                continue
            clo_index = int(entry_match.group(1))
            strength = entry_match.group(2).upper() or "-"
            mappings[outcome_index][clo_index] = strength
            clo_count = max(clo_count, clo_index)

    if clo_count == 0:
        outcome_items = gather_list(rows, "course_learning_outcomes_expected_performance_criteria", split_pipes=True)
        clo_count = len(outcome_items)
    return max(clo_count, 1), mappings, comments


def field_block(label: str, body: str) -> str:
    return f"\\FieldBlock{{{label}}}{{{tex_escape(body)}}}"

def field_block_tex(label: str, body_tex: str) -> str:
    return f"\\FieldBlock{{{label}}}{{{body_tex}}}"

def render_instructors(raw: str) -> str:
    text = raw.strip()
    if not text:
        return ""
    parts = [part.strip() for part in re.split(r"[;\n]+", text) if part.strip()]
    if len(parts) <= 1:
        return tex_escape(text)
    return r"\par ".join(tex_escape(part) for part in parts)


def enum_block(label: str, items: list[str]) -> str:
    rendered_items = "\n".join(f"  \\item {tex_escape(item)}" for item in items)
    return f"\\FieldEnumBlock{{{label}}}{{%\n{rendered_items}\n}}"


def plain_block(label: str, body: str) -> str:
    return "\n".join(
        [
            r"\vspace{10pt}",
            rf"\noindent\textbf{{{label}:}}",
            "",
            "",
            tex_escape(body),
            "",
            "",
            "",
        ]
    )


def build_mapping_table(clo_count: int, mappings: dict[int, dict[int, str]], comments: dict[int, str]) -> str:
    header_cells = " & ".join(rf"\textbf{{{index}}}" for index in range(1, clo_count + 1))
    lines = [
        "% --- CLO -> Program SO mapping table ---",
        r"\par\medskip",
        r"\noindent\textbf{Relationship of course learning outcomes to program student outcomes:}",
        r"\par\smallskip",
        "",
        r"\begingroup",
        r"  \leftskip=0pt",
        r"  \centering",
        r"  \small",
        r"  \setlength{\tabcolsep}{6pt}",
        r"  \renewcommand{\arraystretch}{1.2}",
        "",
        rf"  \begin{{tabularx}}{{\textwidth}}{{|>{{\centering\arraybackslash}}p{{2cm}}|*{{{clo_count}}}{{>{{\centering\arraybackslash}}p{{0.6cm}}|}}X|}}",
        r"    \hline",
        rf"    \textbf{{Program SO}} & \multicolumn{{{clo_count}}}{{c|}}{{\textbf{{Course Outcomes}}}} & \textbf{{Comments}} \\",
        rf"    \cline{{2-{clo_count + 1}}}",
        rf"     & {header_cells} & \\ \hline",
    ]

    for outcome_index in range(1, 8):
        cells = []
        for clo_index in range(1, clo_count + 1):
            cells.append(tex_escape(mappings[outcome_index].get(clo_index, "-" if clo_index else "-")))
        comment = tex_escape(comments.get(outcome_index, ""))
        lines.append(
            rf"    \textbf{{{outcome_index}}} & " + " & ".join(cells) + rf" & {comment} \\ \hline"
        )

    lines.extend(
        [
            r"  \end{tabularx}",
            r"\par\endgroup",
            "",
            r"\smallskip",
            r"{\small S -- Strong connection; M -- Medium connection; W -- Weak connection.}",
        ]
    )
    return "\n".join(lines)


def build_tex(rows: list[dict[str, str]]) -> str:
    course_code = pick_first(rows, "course_code")
    course_name = pick_first(rows, "course_name")
    textbooks = gather_list(rows, "textbooks_and_other_required_material")
    outcomes = gather_list(rows, "course_learning_outcomes_expected_performance_criteria", split_pipes=True)
    topics = gather_list(rows, "topics_covered")
    clo_count, mappings, comments = parse_outcome_mapping(rows)

    parts = [
        "% --- auto-generated from CSV; do not edit by hand ---",
        r"\thispagestyle{fancy}",
        "",
        r"\begin{center}",
        r"  {\Large\bfseries Course Syllabus for ABET\par}",
        r"  \vspace{0.5em}",
        rf"  {{\large \textbf{{{tex_escape(course_code)}}} -- \textbf{{{tex_escape(course_name)}}}\par}}",
        r"\end{center}",
        "",
        plain_block("Required or Elective Course", pick_first(rows, "required_or_elective")),
        plain_block("Catalog Description", pick_first(rows, "catalog_description")),
        plain_block("Prerequisites", pick_first(rows, "prerequisites", "None")),
        plain_block("Categorization of Credits", pick_first(rows, "categorization_of_credits")),
        field_block_tex("Instructors", render_instructors(pick_first(rows, "instructors"))),
        enum_block("Textbooks and other required material", textbooks),
        enum_block("Course learning outcomes/expected performance criteria", outcomes),
        enum_block("Topics covered", topics),
        plain_block("Class/laboratory schedule", pick_first(rows, "class_laboratory_schedule")),
        build_mapping_table(clo_count, mappings, comments),
        "",
        r"\vspace{0.6em}",
        rf"\noindent \textbf{{Prepared by:}} {tex_escape(pick_first(rows, 'prepared_by'))} \hfill \textbf{{Date:}} {tex_escape(pick_first(rows, 'prepared_date'))}",
        "",
    ]
    return "\n".join(parts)


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate syllabus-abet.tex from instructor CSV")
    parser.add_argument("csv_file", help="Path to instructor syllabus CSV")
    parser.add_argument("--output", required=True, help="Output syllabus-abet.tex path")
    args = parser.parse_args()

    csv_path = Path(args.csv_file).resolve()
    out_path = Path(args.output).resolve()
    out_path.parent.mkdir(parents=True, exist_ok=True)

    rows = load_rows(csv_path)
    out_path.write_text(build_tex(rows), encoding="utf-8")
    print(f"Wrote syllabus tex: {out_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
