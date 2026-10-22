#!/usr/bin/env python3
"""Generate tracked per-course main TeX files from a shared template."""

from __future__ import annotations

import argparse
import csv
from dataclasses import dataclass
from pathlib import Path
import shutil
import os


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
TEMPLATE_PATH = SHARED_ROOT / "templates" / "course-main-template.tex"
DEFAULT_REGISTRY = WORKSPACE_ROOT / "config" / "staff" / "course_registry.csv"
DEFAULT_OUTPUT_ROOT = WORKSPACE_ROOT / "tracked" / "course-main-tex"


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


def render_main_tex(template_text: str, course_folder_name: str) -> str:
    return template_text.replace("__COURSE_FOLDER_NAME__", course_folder_name)


def write_main_tex(
    entry: RegistryEntry,
    *,
    template_text: str,
    output_root: Path,
) -> Path:
    out_dir = output_root / entry.semester_workspace
    out_dir.mkdir(parents=True, exist_ok=True)
    out_path = out_dir / f"{entry.course_folder_name}-main.tex"
    out_path.write_text(render_main_tex(template_text, entry.course_folder_name), encoding="utf-8")
    return out_path


def expected_main_paths(entries: list[RegistryEntry], output_root: Path) -> set[Path]:
    return {
        (output_root / entry.semester_workspace / f"{entry.course_folder_name}-main.tex").resolve()
        for entry in entries
        if entry.semester_workspace
    }


def prune_stale_main_tex(entries: list[RegistryEntry], output_root: Path) -> list[Path]:
    keep = expected_main_paths(entries, output_root)
    removed: list[Path] = []
    if not output_root.exists():
        return removed

    for path in sorted(output_root.rglob("*")):
        if path.name.startswith("."):
            continue
        if path.is_file() and path.suffix == ".tex" and path.resolve() not in keep:
            path.unlink()
            removed.append(path)

    for directory in sorted((p for p in output_root.rglob("*") if p.is_dir()), reverse=True):
        if any(directory.iterdir()):
            continue
        directory.rmdir()

    return removed


def sync_main_tex(
    entries: list[RegistryEntry],
    *,
    template_text: str,
    output_root: Path,
    course_code: str | None = None,
    prune: bool = True,
) -> tuple[list[Path], list[Path]]:
    removed = prune_stale_main_tex(entries, output_root) if prune else []

    written: list[Path] = []
    for entry in entries:
        if course_code and entry.course_code != course_code:
            continue
        if not entry.semester_workspace:
            continue
        written.append(write_main_tex(entry, template_text=template_text, output_root=output_root))
    return written, removed


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate tracked course main TeX files")
    parser.add_argument(
        "--registry",
        default=str(DEFAULT_REGISTRY),
        help="Path to course registry CSV",
    )
    parser.add_argument(
        "--output-root",
        default=str(DEFAULT_OUTPUT_ROOT),
        help="Root folder for tracked course main TeX files",
    )
    parser.add_argument(
        "--course-code",
        help="Generate only one course code from the registry",
    )
    parser.add_argument(
        "--no-prune",
        action="store_true",
        help="Do not remove tracked main files for courses missing from the registry",
    )
    args = parser.parse_args()

    registry = load_registry(Path(args.registry).resolve())
    template_text = TEMPLATE_PATH.read_text(encoding="utf-8")
    output_root = Path(args.output_root).resolve()

    written, removed = sync_main_tex(
        registry,
        template_text=template_text,
        output_root=output_root,
        course_code=args.course_code,
        prune=not args.no_prune,
    )

    if not written:
        raise SystemExit("No course main files were generated.")

    for path in removed:
        print(f"removed {path}")
    for path in written:
        print(path)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
