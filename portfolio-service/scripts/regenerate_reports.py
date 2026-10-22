#!/usr/bin/env python3
"""Convenience script to regenerate the markdown and PDF reports in one go."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def run(command: list[str], cwd: Path) -> None:
    """Run a command and raise on failure."""
    print(f"> {' '.join(command)}")
    subprocess.run(command, check=True, cwd=str(cwd))


def main() -> None:
    report = ROOT / "course_completion_report.md"
    summary = ROOT / "course_completion_summary.md"
    pev = ROOT / "course_completion_pev.md"

    pdf_report = report.with_suffix(".pdf")
    pdf_summary = summary.with_suffix(".pdf")
    pdf_pev = pev.with_suffix(".pdf")

    run([sys.executable, "scripts/course_completion.py"], ROOT)
    run(["pandoc", str(report), "-o", str(pdf_report)], ROOT)
    run(["pandoc", str(summary), "-o", str(pdf_summary)], ROOT)
    run(["pandoc", str(pev), "-o", str(pdf_pev)], ROOT)


if __name__ == "__main__":
    main()
