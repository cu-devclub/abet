#!/usr/bin/env python3
"""Generate 90-grades/course-grading.tex from CSV."""

from __future__ import annotations

import argparse
import csv
from pathlib import Path


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


def parse_rows(csv_path: Path) -> list[tuple[str, str]]:
    rows: list[tuple[str, str]] = []
    with csv_path.open(newline="", encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        if reader.fieldnames is None:
            raise SystemExit("CSV has no header row.")
        required = {"component", "weight_percent"}
        missing = required - set(reader.fieldnames)
        if missing:
            raise SystemExit(f"Missing required columns: {', '.join(sorted(missing))}")

        for line_num, row in enumerate(reader, start=2):
            comp = (row.get("component") or "").strip()
            pct = (row.get("weight_percent") or "").strip()
            if not comp and not pct:
                continue
            if not comp or not pct:
                raise SystemExit(
                    f"Row {line_num}: both component and weight_percent are required."
                )
            try:
                pct_value = float(pct)
            except ValueError as exc:
                raise SystemExit(f"Row {line_num}: weight_percent must be numeric.") from exc
            if pct_value < 0:
                raise SystemExit(f"Row {line_num}: weight_percent must be >= 0.")
            pct_text = str(int(pct_value)) if pct_value.is_integer() else str(pct_value)
            rows.append((comp, pct_text))

    if not rows:
        raise SystemExit("No grading rows found in CSV.")
    return rows


def build_tex(rows: list[tuple[str, str]]) -> str:
    lines = [
        "% --- auto-generated; do not edit by hand ---",
        r"\newpage",
        r"\renewcommand{\arraystretch}{1.3}",
        r"\setlength{\tabcolsep}{0pt}",
        "",
        r"\begin{center}",
        r"\vspace*{0.2\textheight} % push it down",
        r"\begin{tabularx}{0.6\textwidth}{@{} L{0.6\textwidth} r @{}}",
    ]

    for comp, pct in rows:
        lines.append(rf"\textbf{{{tex_escape(comp)}}} & \textcolor{{gray}}{{({tex_escape(pct)}\%)}} \\")

    lines.extend(
        [
            r"\end{tabularx}",
            r"\vfill % push bottom to balance",
            r"\end{center}",
            "",
        ]
    )
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate course-grading.tex from CSV")
    parser.add_argument("csv_file", help="Path to instructor grading CSV")
    parser.add_argument("--output", required=True, help="Output course-grading.tex path")
    args = parser.parse_args()

    csv_path = Path(args.csv_file).resolve()
    out_path = Path(args.output).resolve()
    out_path.parent.mkdir(parents=True, exist_ok=True)

    rows = parse_rows(csv_path)
    out_path.write_text(build_tex(rows), encoding="utf-8")
    print(f"Wrote grading tex: {out_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
