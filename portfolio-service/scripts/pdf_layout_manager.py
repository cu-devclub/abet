#!/usr/bin/env python3
"""Manage staff-editable per-course PDF layout override CSVs."""

from __future__ import annotations

import argparse
import csv
import json
import os
import shutil
import subprocess
from pathlib import Path


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
DEFAULT_OVERRIDES_DIR = WORKSPACE_ROOT / "config" / "staff" / "pdf_layout_overrides"
DEFAULT_JSON = WORKSPACE_ROOT / "config" / "generated" / "pdf_layout_overrides.json"
DEFAULT_TEX = WORKSPACE_ROOT / "config" / "generated" / "pdf_layout_overrides.tex"

CSV_HEADERS = [
    "relative_path",
    "layout",
    "pages",
    "nup",
    "landscape",
    "scale",
    "frame",
]

DEFAULT_LAYOUT_PRESETS = {
    "slides_4up": {
        "pages": "-",
        "nup": "2x2",
        "landscape": "true",
        "scale": "0.85",
        "frame": "true",
    },
    "slides_2up": {
        "pages": "-",
        "nup": "1x2",
        "landscape": "true",
        "scale": "0.9",
        "frame": "true",
    },
    "slides_2across": {
        "pages": "-",
        "nup": "2x1",
        "landscape": "true",
        "scale": "0.9",
        "frame": "true",
    },
    "fullpage_portrait": {
        "pages": "-",
        "nup": "1x1",
        "landscape": "false",
        "scale": "0.95",
        "frame": "false",
    },
    "fullpage_landscape": {
        "pages": "-",
        "nup": "1x1",
        "landscape": "true",
        "scale": "0.95",
        "frame": "false",
    },
    "scoresheet_compact": {
        "pages": "-",
        "nup": "1x1",
        "landscape": "false",
        "scale": "0.9",
        "frame": "false",
    },
}


def get_layout_presets() -> dict:
    """Dynamically load LAYOUT_PRESETS from MariaDB / settings.json via config loader."""
    try:
        import sys
        root_dir = str(Path(__file__).resolve().parents[1])
        if root_dir not in sys.path:
            sys.path.insert(0, root_dir)
        import config
        eff = getattr(config, "effective_config", {})
        custom = eff.get("LAYOUT_PRESETS")
        if custom:
            if isinstance(custom, str):
                custom = json.loads(custom)
            if isinstance(custom, dict):
                return {**DEFAULT_LAYOUT_PRESETS, **custom}
    except Exception:
        pass
    return DEFAULT_LAYOUT_PRESETS


LAYOUT_PRESETS = get_layout_presets()

OPTION_ORDER = ["pages", "nup", "landscape", "frame", "pagecommand", "scale"]

def normalize_bool(value: str) -> str:
    raw = value.strip().lower()
    if raw in {"true", "1", "yes", "y"}:
        return "true"
    if raw in {"false", "0", "no", "n"}:
        return "false"
    return ""


def pdf_metadata(pdf_path: Path) -> dict[str, float | int] | None:
    pdfinfo = shutil.which("pdfinfo")
    if pdfinfo is None:
        return None
    try:
        proc = subprocess.run(
            [pdfinfo, str(pdf_path)],
            capture_output=True,
            text=False,
            check=False,
        )
    except OSError:
        return None
    if proc.returncode != 0:
        return None
    try:
        stdout = proc.stdout.decode("utf-8")
    except UnicodeDecodeError:
        # Windows code pages can fail on PDF metadata bytes; preserve the build by
        # treating undecodable bytes as replacement characters.
        stdout = proc.stdout.decode(errors="replace")
    if not stdout:
        return None
    data: dict[str, str] = {}
    for line in stdout.splitlines():
        if ":" not in line:
            continue
        key, value = line.split(":", 1)
        data[key.strip()] = value.strip()
    try:
        pages = int(data.get("Pages", "0"))
    except ValueError:
        pages = 0
    width = 0.0
    height = 0.0
    page_size = data.get("Page size", "")
    parts = page_size.split()
    if len(parts) >= 3:
        try:
            width = float(parts[0])
            height = float(parts[2])
        except ValueError:
            width = 0.0
            height = 0.0
    return {"pages": pages, "width": width, "height": height}


def suggest_layout(course_dir: Path, relpath: str) -> str:
    name = Path(relpath).name.lower()
    rel = relpath.lower()
    metadata = pdf_metadata(course_dir / relpath) or {"pages": 0, "width": 0.0, "height": 0.0}
    pages = int(metadata.get("pages", 0))
    width = float(metadata.get("width", 0.0))
    height = float(metadata.get("height", 0.0))
    aspect = (width / height) if width and height else 1.0

    if rel.startswith("10-lectures/"):
        return "slides_4up" if pages >= 4 else "slides_2up"

    if name in {"overall-grades.pdf", "asg-all-scores.pdf", "quiz-all-scores.pdf"} or name.endswith("-scores.pdf"):
        return "scoresheet_compact"

    if "instructions" in name and pages <= 2 and aspect > 1.15:
        return "fullpage_landscape"

    if aspect > 1.2:
        return "fullpage_landscape"
    if aspect < 0.9:
        return "fullpage_portrait"

    if rel.startswith("02-textbook/"):
        return "fullpage_portrait"
    if rel.startswith("20-assignments/") or rel.startswith("30-quizzes/") or rel.startswith("40-exams/"):
        return "fullpage_portrait"
    return "fullpage_portrait"


def default_option_map_for_relpath(relpath: str) -> dict[str, str]:
    rel = relpath.lower()
    if rel.startswith("02-textbook/"):
        return {
            "pages": "-",
            "pagecommand": r"{\thispagestyle{fancy}}",
            "scale": "0.85",
        }
    if rel.startswith("10-lectures/"):
        return {
            "pages": "-",
            "nup": "2x2",
            "landscape": "true",
            "frame": "true",
            "pagecommand": r"{\thispagestyle{fancy}}",
            "scale": "0.85",
        }
    return {
        "pages": "-",
        "pagecommand": r"{\thispagestyle{fancy}}",
    }


def render_option_map(option_map: dict[str, str]) -> str:
    parts: list[str] = []
    for key in OPTION_ORDER:
        value = option_map.get(key, "").strip()
        if value:
            parts.append(f"{key}={value}")
    return ",".join(parts)


def iter_includable_pdfs(course_dir: Path) -> list[str]:
    included: list[str] = []

    def add_if_exists(relpath: str) -> None:
        if (course_dir / relpath).is_file():
            included.append(relpath)

    for path in sorted((course_dir / "02-textbook").glob("textbook-part*.pdf")) if (course_dir / "02-textbook").is_dir() else []:
        included.append(str(path.relative_to(course_dir)))
    add_if_exists("90-grades/overall-grades.pdf")
    add_if_exists("20-assignments/asg-all-scores.pdf")
    add_if_exists("30-quizzes/quiz-all-scores.pdf")

    for week in range(1, 21):
        for part in range(1, 10):
            add_if_exists(f"10-lectures/week{week:02d}-part{part:02d}.pdf")

    for assignment in range(1, 31):
        for suffix in ["instructions", "high", "average", "low", "scores"]:
            add_if_exists(f"20-assignments/asg{assignment:02d}-{suffix}.pdf")

    for quiz in range(1, 31):
        for suffix in ["instructions", "high", "average", "low", "scores"]:
            add_if_exists(f"30-quizzes/quiz{quiz:02d}-{suffix}.pdf")

    for exam in ["midterm", "final"]:
        for suffix in ["scores", "instructions", "high", "average", "low"]:
            add_if_exists(f"40-exams/{exam}-{suffix}.pdf")

    return included


def ensure_course_override_csv(course_dir: Path, overrides_dir: Path) -> Path:
    overrides_dir.mkdir(parents=True, exist_ok=True)
    csv_path = overrides_dir / f"{course_dir.name}.csv"
    existing_rows: dict[str, dict[str, str]] = {}
    if csv_path.is_file():
        with csv_path.open(newline="", encoding="utf-8-sig") as f:
            reader = csv.DictReader(f)
            if reader.fieldnames:
                for row in reader:
                    relpath = (row.get("relative_path") or "").strip()
                    if relpath:
                        existing_rows[relpath] = {header: (row.get(header) or "").strip() for header in CSV_HEADERS}

    rows: list[dict[str, str]] = []
    for relpath in iter_includable_pdfs(course_dir):
        row = existing_rows.get(relpath, {})
        rows.append({header: row.get(header, "") for header in CSV_HEADERS})
        rows[-1]["relative_path"] = relpath
        if not any(rows[-1].get(header, "").strip() for header in CSV_HEADERS if header != "relative_path"):
            rows[-1]["layout"] = suggest_layout(course_dir, relpath)

    with csv_path.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=CSV_HEADERS)
        writer.writeheader()
        writer.writerows(rows)

    return csv_path


def load_overrides_from_dir(overrides_dir: Path) -> dict[str, str]:
    overrides: dict[str, str] = {}
    if not overrides_dir.is_dir():
        return overrides

    for csv_path in sorted(overrides_dir.glob("*.csv")):
        course_folder_name = csv_path.stem
        with csv_path.open(newline="", encoding="utf-8-sig") as f:
            reader = csv.DictReader(f)
            if reader.fieldnames is None:
                continue
            for row in reader:
                relpath = (row.get("relative_path") or "").strip().lstrip("/")
                if not relpath:
                    continue
                option_map = default_option_map_for_relpath(relpath)
                presets = get_layout_presets()
                preset_name = (row.get("layout") or row.get("preset") or "").strip()
                preset = presets.get(preset_name, {})
                pages = (row.get("pages") or "").strip() or preset.get("pages", "")
                nup = (row.get("nup") or "").strip() or preset.get("nup", "")
                landscape = normalize_bool((row.get("landscape") or "").strip()) or preset.get("landscape", "")
                scale = (row.get("scale") or "").strip() or preset.get("scale", "")
                frame = normalize_bool((row.get("frame") or "").strip()) or preset.get("frame", "")
                if pages:
                    option_map["pages"] = pages
                if nup:
                    option_map["nup"] = nup
                if landscape:
                    option_map["landscape"] = landscape
                if scale:
                    option_map["scale"] = scale
                if frame:
                    option_map["frame"] = frame
                resolved = render_option_map(option_map)
                if not resolved:
                    continue
                canonical_key = f"course_documents/{course_folder_name}/{relpath}"
                runtime_key = f"../course_documents/{course_folder_name}/{relpath}"
                overrides[canonical_key] = resolved
                overrides[runtime_key] = resolved
    return overrides


def write_tex(overrides: dict[str, str], tex_path: Path) -> None:
    lines = [
        "% Auto-generated from pdf_layout_overrides/*.csv; do not edit manually.",
        "% Generated by scripts/pdf_layout_manager.py.",
    ]
    for path, options in sorted(overrides.items()):
        lines.append(f"\\AddPdfLayoutOverride{{{path}}}{{{options}}}")
    tex_path.parent.mkdir(parents=True, exist_ok=True)
    tex_path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def sync_pdf_layout_overrides(
    *,
    overrides_dir: Path = DEFAULT_OVERRIDES_DIR,
    output_json: Path = DEFAULT_JSON,
    output_tex: Path = DEFAULT_TEX,
) -> dict[str, Path]:
    overrides = load_overrides_from_dir(overrides_dir)
    output_json.parent.mkdir(parents=True, exist_ok=True)
    output_json.write_text(json.dumps(overrides, indent=2) + "\n", encoding="utf-8")
    write_tex(overrides, output_tex)
    return {
        "overrides_dir": overrides_dir,
        "json_file": output_json,
        "tex_file": output_tex,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Sync per-course PDF layout override CSVs")
    parser.add_argument("--overrides-dir", default=str(DEFAULT_OVERRIDES_DIR), help="Directory containing per-course override CSVs")
    parser.add_argument("--output-json", default=str(DEFAULT_JSON), help="Derived JSON output path")
    parser.add_argument("--output-tex", default=str(DEFAULT_TEX), help="Derived TeX output path")
    args = parser.parse_args()

    result = sync_pdf_layout_overrides(
        overrides_dir=Path(args.overrides_dir).resolve(),
        output_json=Path(args.output_json).resolve(),
        output_tex=Path(args.output_tex).resolve(),
    )
    print(result["overrides_dir"])
    print(result["json_file"])
    print(result["tex_file"])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
