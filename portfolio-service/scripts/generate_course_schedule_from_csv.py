#!/usr/bin/env python3
"""Generate course-schedule.tex from a simple week/topic CSV."""

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


def parse_rows(csv_path: Path) -> list[tuple[int, str]]:
    rows: list[tuple[int, str]] = []
    with csv_path.open(newline="", encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        if reader.fieldnames is None:
            raise SystemExit("CSV has no header row.")
        required = {"week", "topic"}
        missing = required - set(reader.fieldnames)
        if missing:
            raise SystemExit(f"Missing required columns: {', '.join(sorted(missing))}")

        for line_num, row in enumerate(reader, start=2):
            week_raw = (row.get("week") or "").strip()
            topic = (row.get("topic") or "").strip()
            if not week_raw and not topic:
                continue
            if not week_raw:
                raise SystemExit(f"Row {line_num}: week is required when topic is provided.")
            try:
                week = int(week_raw)
            except ValueError as exc:
                raise SystemExit(f"Row {line_num}: invalid week '{week_raw}'. Use integers.") from exc
            if week <= 0:
                raise SystemExit(f"Row {line_num}: week must be > 0.")
            if not topic:
                continue
            rows.append((week, topic))

    rows.sort(key=lambda x: x[0])
    return rows


def build_schedule_tex(rows: list[tuple[int, str]]) -> str:
    lines = [
        "% --- auto-generated; do not edit by hand ---",
        r"\newpage",
        r"\renewcommand{\arraystretch}{1.3}",
        r"\setlength{\tabcolsep}{0pt}",
        "",
        r"\begin{center}",
        r"\begin{tabularx}{\textwidth}{@{} L{2.6cm} V @{}}",
    ]
    for week, topic in rows:
        lines.append(rf"\textbf{{Week {week}:}}  & {tex_escape(topic)} \\")
    lines.extend([r"\end{tabularx}", r"\end{center}", ""])
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate course-schedule.tex from CSV")
    parser.add_argument("csv_file", help="Path to instructor schedule CSV")
    parser.add_argument("--output", required=True, help="Output course-schedule.tex path")
    args = parser.parse_args()

    csv_path = Path(args.csv_file).resolve()
    out_path = Path(args.output).resolve()
    out_path.parent.mkdir(parents=True, exist_ok=True)

    rows = parse_rows(csv_path)
    if not rows:
        raise SystemExit("No non-empty week/topic rows found in CSV.")

    out_path.write_text(build_schedule_tex(rows), encoding="utf-8")
    print(f"Wrote schedule tex: {out_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
