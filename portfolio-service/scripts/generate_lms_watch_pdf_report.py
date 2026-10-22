#!/usr/bin/env python3
"""Generate a formatted PDF report from lms-watch-staff CSV output."""

from __future__ import annotations

import argparse
import csv
import shutil
import subprocess
import sys
import textwrap
from datetime import datetime
from pathlib import Path


REQUIRED_COLUMNS = [
    "course_code",
    "course_name",
    "status",
    "action",
    "latest_modified",
    "latest_item",
    "check_url",
    "note",
]

STATUS_ORDER = {
    "changed": 0,
    "error": 1,
    "baseline-set": 2,
    "unchanged": 3,
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate LMS watch staff PDF report from CSV")
    parser.add_argument("csv_file", help="Path to lms-watch-staff CSV file")
    parser.add_argument(
        "--output-prefix",
        help="Output path prefix (without extension). Defaults to CSV filename stem.",
    )
    parser.add_argument(
        "--skip-pdf",
        action="store_true",
        help="Generate only .tex and skip PDF compilation.",
    )
    return parser.parse_args()


def latex_escape(text: str) -> str:
    value = text or ""
    replacements = {
        "\\": r"\textbackslash{}",
        "&": r"\&",
        "%": r"\%",
        "$": r"\$",
        "#": r"\#",
        "_": r"\_",
        "{": r"\{",
        "}": r"\}",
        "~": r"\textasciitilde{}",
        "^": r"\textasciicircum{}",
    }
    for old, new in replacements.items():
        value = value.replace(old, new)
    return value


def load_rows(csv_path: Path) -> list[dict[str, str]]:
    with csv_path.open(newline="", encoding="utf-8-sig") as handle:
        reader = csv.DictReader(handle)
        if reader.fieldnames is None:
            raise SystemExit(f"CSV has no header row: {csv_path}")
        missing = [name for name in REQUIRED_COLUMNS if name not in reader.fieldnames]
        if missing:
            raise SystemExit(f"CSV missing required columns: {', '.join(missing)}")
        rows = []
        for row in reader:
            rows.append({name: (row.get(name) or "").strip() for name in REQUIRED_COLUMNS})
    return rows


def sorted_rows(rows: list[dict[str, str]]) -> list[dict[str, str]]:
    return sorted(
        rows,
        key=lambda row: (
            STATUS_ORDER.get((row.get("status") or "").lower(), 9),
            (row.get("course_code") or "").lower(),
            (row.get("course_name") or "").lower(),
        ),
    )


def build_tex(rows: list[dict[str, str]], source_csv: Path) -> str:
    total = len(rows)
    changed = sum(1 for row in rows if row["status"].lower() == "changed")
    errors = sum(1 for row in rows if row["status"].lower() == "error")
    unchanged = sum(1 for row in rows if row["status"].lower() == "unchanged")
    baseline = sum(1 for row in rows if row["status"].lower() == "baseline-set")

    lines = [
        r"\documentclass[11pt]{article}",
        r"\usepackage[a4paper,margin=0.7in]{geometry}",
        r"\usepackage[T1]{fontenc}",
        r"\usepackage[utf8]{inputenc}",
        r"\usepackage{lmodern}",
        r"\usepackage{booktabs}",
        r"\usepackage{longtable}",
        r"\usepackage{array}",
        r"\usepackage{xcolor}",
        r"\usepackage[hidelinks]{hyperref}",
        r"\setlength{\parindent}{0pt}",
        r"\setlength{\parskip}{4pt}",
        r"\begin{document}",
        r"{\LARGE \textbf{LMS Watch Staff Report}}\\",
        f"Generated: {latex_escape(datetime.now().isoformat(timespec='seconds'))}\\\\",
        f"Source CSV: {latex_escape(str(source_csv))}\\\\",
        r"\vspace{6pt}",
        r"\begin{tabular}{@{}ll@{}}",
        f"Total Courses: & {total}\\\\",
        f"Changed: & {changed}\\\\",
        f"Errors: & {errors}\\\\",
        f"Baseline Set: & {baseline}\\\\",
        f"Unchanged: & {unchanged}\\\\",
        r"\end{tabular}",
        r"\vspace{10pt}",
        r"\renewcommand{\arraystretch}{1.15}",
        r"\small",
        r"\begin{longtable}{>{\raggedright\arraybackslash}p{0.10\textwidth}"
        r">{\raggedright\arraybackslash}p{0.19\textwidth}"
        r">{\raggedright\arraybackslash}p{0.09\textwidth}"
        r">{\raggedright\arraybackslash}p{0.13\textwidth}"
        r">{\raggedright\arraybackslash}p{0.10\textwidth}"
        r">{\raggedright\arraybackslash}p{0.16\textwidth}"
        r">{\raggedright\arraybackslash}p{0.16\textwidth}}",
        r"\toprule",
        r"Code & Course & Status & Action & Last Modified & Latest Item & Note\\",
        r"\midrule",
        r"\endfirsthead",
        r"\toprule",
        r"Code & Course & Status & Action & Last Modified & Latest Item & Note\\",
        r"\midrule",
        r"\endhead",
        r"\bottomrule",
        r"\endfoot",
    ]

    for row in rows:
        line = " & ".join(
            [
                latex_escape(row["course_code"]),
                latex_escape(row["course_name"]),
                latex_escape(row["status"]),
                latex_escape(row["action"]),
                latex_escape(row["latest_modified"]),
                latex_escape(row["latest_item"]),
                latex_escape(row["note"]),
            ]
        )
        lines.append(f"{line}\\\\")

    lines.extend([r"\end{longtable}", r"\end{document}"])
    return "\n".join(lines) + "\n"


def run_compile(tex_path: Path) -> None:
    workdir = tex_path.parent
    latexmk = shutil.which("latexmk")
    pdflatex = shutil.which("pdflatex")

    if latexmk:
        cmd = [latexmk, "-pdf", "-interaction=nonstopmode", tex_path.name]
        subprocess.run(cmd, cwd=workdir, check=True)
        return

    if pdflatex:
        cmd = [pdflatex, "-interaction=nonstopmode", tex_path.name]
        subprocess.run(cmd, cwd=workdir, check=True)
        subprocess.run(cmd, cwd=workdir, check=True)
        return

    raise RuntimeError("No LaTeX compiler found")


def build_text_report(rows: list[dict[str, str]], source_csv: Path) -> str:
    total = len(rows)
    changed = sum(1 for row in rows if row["status"].lower() == "changed")
    errors = sum(1 for row in rows if row["status"].lower() == "error")
    unchanged = sum(1 for row in rows if row["status"].lower() == "unchanged")
    baseline = sum(1 for row in rows if row["status"].lower() == "baseline-set")
    generated_at = datetime.now().isoformat(timespec="seconds")

    lines = [
        "LMS WATCH STAFF REPORT",
        "=" * 80,
        f"Generated: {generated_at}",
        f"Source CSV: {source_csv}",
        "",
        f"Total Courses: {total}",
        f"Changed: {changed}",
        f"Errors: {errors}",
        f"Baseline Set: {baseline}",
        f"Unchanged: {unchanged}",
        "",
        "-" * 80,
    ]

    for row in rows:
        lines.append(f"{row['course_code']} - {row['course_name']}")
        lines.append(
            f"  Status: {row['status'] or '-'} | Action: {row['action'] or '-'} | Last Modified: {row['latest_modified'] or '-'}"
        )
        lines.append(f"  Latest Item: {row['latest_item'] or '-'}")
        if row["check_url"]:
            lines.append("  LMS URL:")
            lines.extend([f"    {chunk}" for chunk in textwrap.wrap(row["check_url"], width=96)])
        note = row["note"] or "-"
        lines.append("  Note:")
        wrapped_note = textwrap.wrap(note, width=96) or ["-"]
        lines.extend([f"    {chunk}" for chunk in wrapped_note])
        lines.append("-" * 80)

    return "\n".join(lines) + "\n"


def compile_with_cupsfilter(text_path: Path, pdf_path: Path) -> None:
    cupsfilter = shutil.which("cupsfilter")
    if not cupsfilter:
        raise SystemExit(
            "No PDF compiler found. Install latexmk/pdflatex, or ensure cupsfilter is available."
        )
    with pdf_path.open("wb") as out_handle:
        subprocess.run(
            [cupsfilter, "-m", "application/pdf", str(text_path)],
            check=True,
            stdout=out_handle,
            stderr=subprocess.PIPE,
        )


def main() -> int:
    args = parse_args()
    csv_path = Path(args.csv_file).resolve()
    if not csv_path.is_file():
        raise SystemExit(f"CSV file not found: {csv_path}")

    if args.output_prefix:
        out_prefix = Path(args.output_prefix).resolve()
    else:
        out_prefix = csv_path.with_suffix("")

    rows = sorted_rows(load_rows(csv_path))
    tex_path = out_prefix.with_suffix(".tex")
    pdf_path = out_prefix.with_suffix(".pdf")
    txt_path = out_prefix.with_suffix(".txt")
    tex_path.parent.mkdir(parents=True, exist_ok=True)
    tex_path.write_text(build_tex(rows, csv_path), encoding="utf-8")

    if args.skip_pdf:
        print(f"TeX report: {tex_path}")
        return 0

    method = "latex"
    try:
        run_compile(tex_path)
    except RuntimeError:
        txt_path.write_text(build_text_report(rows, csv_path), encoding="utf-8")
        compile_with_cupsfilter(txt_path, pdf_path)
        method = "cupsfilter-text"

    if not pdf_path.is_file():
        raise SystemExit(f"Expected PDF not found after compile: {pdf_path}")

    print(f"TeX report: {tex_path}")
    if txt_path.is_file():
        print(f"Text report: {txt_path}")
    print(f"PDF report: {pdf_path}")
    print(f"PDF method: {method}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
