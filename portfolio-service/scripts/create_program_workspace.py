#!/usr/bin/env python3
"""Scaffold a new program workspace under programs/."""

from __future__ import annotations

import argparse
import csv
import json
import os
from pathlib import Path
from typing import Any

from expected_counts_manager import sync_expected_counts
from pdf_layout_manager import sync_pdf_layout_overrides


SHARED_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_PROGRAMS_ROOT = SHARED_ROOT / "programs"

RUNTIME_DIRS = [
    "incoming",
    "processed",
    "failed",
    "staging",
    "repair",
    "workspaces",
    "output",
    "output/course-pdfs",
    "output/reports",
    "logs",
]

TRACKED_DIRS = [
    "course-main-tex",
]

EXPECTED_COUNT_FILES = [
    "10-lectures.csv",
    "20-assignments.csv",
    "30-quizzes.csv",
    "40-exams.csv",
]

CONFIG_README = """# Config Layout

## Staff Editable

Use only files under `config/staff/` for manual edits:

- `course_registry.csv`
- `course_notes.csv`
- `lms_watch_courses.csv`
- `lms_watch_settings.json`
- `pdf_layout_overrides/<course>.csv` (auto-generated after the course is processed once)
- `expected_counts/*.csv`

In each `pdf_layout_overrides/<course>.csv`, prefer the `layout` column with presets like:

- `slides_4up`
- `slides_2up`
- `slides_2across`
- `fullpage_portrait`
- `fullpage_landscape`
- `scoresheet_compact`

The raw columns (`pages`, `nup`, `landscape`, `scale`, `frame`) are optional overrides for edge cases.

## Generated

Files under `config/generated/` are derived automatically by the pipeline.
Do not edit these files manually.
"""

COURSE_NOTES_HEADERS = [
    "semester_workspace",
    "course_code",
    "course_folder_name",
    "link",
    "comment",
]

LMS_WATCH_HEADERS = [
    "enabled",
    "course_code",
    "course_name",
    "check_url",
    "row_selector",
    "name_selector",
    "modified_selector",
    "last_seen_modified",
    "last_seen_item",
    "last_checked_at",
    "last_status",
    "last_note",
]

LMS_WATCH_SETTINGS = """{
  "login_url": "https://www.mycourseville.com",
  "browser_channel": "chrome",
  "persistent_profile_scope": "staff",
  "persistent_profile_dir": "",
  "max_report_history": 10,
  "page_ready_selector": "",
  "row_selector": "",
  "name_selector": "",
  "modified_selector": "",
  "rows_sorted_newest_first": true
}
"""

REGISTRY_HEADERS = [
    "course_code",
    "course_name",
    "semester_workspace",
    "course_folder_name",
]

COMMAND_TEMPLATES = {
    "Process-Instructor-Zip.command": """#!/bin/zsh
set -euo pipefail
WORKSPACE_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$WORKSPACE_DIR"
mkdir -p "runtime/logs"
timestamp=$(date +"%Y%m%d-%H%M%S")
log_path="runtime/logs/process-instructor-zip-${timestamp}.log"
exec > >(tee -a "$log_path") 2>&1
echo "[$(date +"%Y-%m-%d %H:%M:%S")] Process-Instructor-Zip.command started"
echo "Workspace: $WORKSPACE_DIR"
if [ -z "${1:-}" ]; then
  inbox_items=$(find "runtime/incoming" -mindepth 1 -maxdepth 1 ! -name '.DS_Store' ! -name '._*' | wc -l | tr -d ' ')
  zip_count=$(find "runtime/incoming" -maxdepth 1 -type f -name '*.zip' ! -name '.DS_Store' ! -name '._*' | wc -l | tr -d ' ')
  if [ "$inbox_items" -ne 1 ] || [ "$zip_count" -ne 1 ]; then
    echo "runtime/incoming must contain exactly one item, and it must be a .zip file. Otherwise pass an explicit source path."
    exit 1
  fi
fi
python3 ../../scripts/runtime_pipeline.py build-course "${1:-}"
""",
    "Validate.command": """#!/bin/zsh
set -euo pipefail
WORKSPACE_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$WORKSPACE_DIR"
mkdir -p "runtime/logs"
timestamp=$(date +"%Y%m%d-%H%M%S")
log_path="runtime/logs/validate-${timestamp}.log"
exec > >(tee -a "$log_path") 2>&1
echo "[$(date +"%Y-%m-%d %H:%M:%S")] Validate.command started"
echo "Workspace: $WORKSPACE_DIR"
if [ -z "${1:-}" ]; then
  inbox_items=$(find "runtime/incoming" -mindepth 1 -maxdepth 1 ! -name '.DS_Store' ! -name '._*' | wc -l | tr -d ' ')
  zip_count=$(find "runtime/incoming" -maxdepth 1 -type f -name '*.zip' ! -name '.DS_Store' ! -name '._*' | wc -l | tr -d ' ')
  if [ "$inbox_items" -ne 1 ] || [ "$zip_count" -ne 1 ]; then
    echo "runtime/incoming must contain exactly one item, and it must be a .zip file. Otherwise pass an explicit source path."
    exit 1
  fi
fi
python3 ../../scripts/runtime_pipeline.py validate "${1:-}"
""",
    "Repair-Submission.command": """#!/bin/zsh
set -euo pipefail
WORKSPACE_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$WORKSPACE_DIR"
mkdir -p "runtime/logs"
timestamp=$(date +"%Y%m%d-%H%M%S")
log_path="runtime/logs/repair-submission-${timestamp}.log"
exec > >(tee -a "$log_path") 2>&1
echo "[$(date +"%Y-%m-%d %H:%M:%S")] Repair-Submission.command started"
echo "Workspace: $WORKSPACE_DIR"
if [ -z "${1:-}" ]; then
  inbox_items=$(find "runtime/incoming" -mindepth 1 -maxdepth 1 ! -name '.DS_Store' ! -name '._*' | wc -l | tr -d ' ')
  zip_count=$(find "runtime/incoming" -maxdepth 1 -type f -name '*.zip' ! -name '.DS_Store' ! -name '._*' | wc -l | tr -d ' ')
  if [ "$inbox_items" -ne 1 ] || [ "$zip_count" -ne 1 ]; then
    echo "runtime/incoming must contain exactly one item, and it must be a .zip file. Otherwise pass an explicit source path."
    exit 1
  fi
fi
python3 ../../scripts/runtime_pipeline.py repair-submission "${1:-}"
""",
    "Validate-All-Incoming.command": """#!/bin/zsh
set -euo pipefail
WORKSPACE_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$WORKSPACE_DIR"
mkdir -p "runtime/logs"
timestamp=$(date +"%Y%m%d-%H%M%S")
log_path="runtime/logs/validate-all-incoming-${timestamp}.log"
exec > >(tee -a "$log_path") 2>&1
echo "[$(date +"%Y-%m-%d %H:%M:%S")] Validate-All-Incoming.command started"
echo "Workspace: $WORKSPACE_DIR"
python3 ../../scripts/runtime_pipeline.py validate-all-incoming
""",
    "Build-All.command": """#!/bin/zsh
set -euo pipefail
WORKSPACE_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$WORKSPACE_DIR"
mkdir -p "runtime/logs"
timestamp=$(date +"%Y%m%d-%H%M%S")
log_path="runtime/logs/build-all-${timestamp}.log"
exec > >(tee -a "$log_path") 2>&1
echo "[$(date +"%Y-%m-%d %H:%M:%S")] Build-All.command started"
echo "Workspace: $WORKSPACE_DIR"
python3 ../../scripts/runtime_pipeline.py build-all
""",
    "Build-One-Course.command": """#!/bin/zsh
set -euo pipefail
WORKSPACE_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$WORKSPACE_DIR"
mkdir -p "runtime/logs"
timestamp=$(date +"%Y%m%d-%H%M%S")
log_path="runtime/logs/build-one-course-${timestamp}.log"
exec > >(tee -a "$log_path") 2>&1
echo "[$(date +"%Y-%m-%d %H:%M:%S")] Build-One-Course.command started"
echo "Workspace: $WORKSPACE_DIR"
course_folder_name="${1:-}"
if [ -z "$course_folder_name" ]; then
  course_list=()
  while IFS= read -r line; do
    course_list+=("$line")
  done < <(/usr/bin/find "$WORKSPACE_DIR/runtime/workspaces" -type f -path '*/Course-main-tex/*-main.tex' -print | /usr/bin/sed 's#.*/##' | /usr/bin/sed 's/-main\\.tex$//' | /usr/bin/sort -u)

  if [ "${#course_list[@]}" -eq 0 ]; then
    echo "No courses found in runtime/workspaces."
    exit 1
  fi

  echo "Available courses:"
  i=1
  for item in "${course_list[@]}"; do
    echo "  $i. $item"
    i=$((i + 1))
  done
  echo
  printf "Enter the course number or folder name to rebuild: "
  IFS= read -r selection

  if [ -z "$selection" ]; then
    echo "No course selection provided."
    exit 1
  fi

  if [[ "$selection" =~ ^[0-9]+$ ]]; then
    index=$((selection))
    if [ "$index" -lt 1 ] || [ "$index" -gt "${#course_list[@]}" ]; then
      echo "Invalid course number: $selection"
      exit 1
    fi
    course_folder_name="${course_list[$index]}"
  else
    course_folder_name="$selection"
  fi
fi
if [ -z "$course_folder_name" ]; then
  echo "No course folder name provided."
  exit 1
fi
python3 ../../scripts/runtime_pipeline.py build-one "$course_folder_name"
""",
    "Check-LMS-Updates.command": """#!/bin/zsh
set -euo pipefail
WORKSPACE_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$WORKSPACE_DIR"
mkdir -p "runtime/logs"
timestamp=$(date +"%Y%m%d-%H%M%S")
log_path="runtime/logs/check-lms-updates-${timestamp}.log"
exec > >(tee -a "$log_path") 2>&1
echo "[$(date +"%Y-%m-%d %H:%M:%S")] Check-LMS-Updates.command started"
echo "Workspace: $WORKSPACE_DIR"
python3 ../../scripts/check_lms_updates.py "$@"
""",
}

POWERSHELL_TEMPLATES = {
    "Process-Instructor-Zip.ps1": r"""$ErrorActionPreference = 'Stop'
$workspaceDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $workspaceDir

function Get-PythonLauncher {
    if (Get-Command py -ErrorAction SilentlyContinue) { return @('py', '-3') }
    if (Get-Command python -ErrorAction SilentlyContinue) { return @('python') }
    throw 'Python launcher not found. Install Python 3 and ensure py.exe or python.exe is on PATH.'
}

function Invoke-WithPython {
    param([string[]]$CommandArgs)
    $launcher = Get-PythonLauncher
    if ($launcher.Count -gt 1) {
        & $launcher[0] $launcher[1..($launcher.Count - 1)] @CommandArgs
    } else {
        & $launcher[0] @CommandArgs
    }
}

New-Item -ItemType Directory -Force -Path 'runtime/logs' | Out-Null
$timestamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$logPath = Join-Path $workspaceDir "runtime/logs/process-instructor-zip-$timestamp.log"
Start-Transcript -Path $logPath -Append | Out-Null

try {
    Write-Host "[$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] Process-Instructor-Zip.ps1 started"
    Write-Host "Workspace: $workspaceDir"
    $source = if ($args.Count -gt 0) { $args[0] } else { '' }
    if ([string]::IsNullOrWhiteSpace($source)) {
        $items = Get-ChildItem 'runtime/incoming' -Force | Where-Object { $_.Name -notlike '._*' -and $_.Name -ne '.DS_Store' }
        $zipItems = $items | Where-Object { -not $_.PSIsContainer -and $_.Extension -ieq '.zip' }
        if ($items.Count -ne 1 -or $zipItems.Count -ne 1) {
            throw 'runtime/incoming must contain exactly one item, and it must be a .zip file. Otherwise pass an explicit source path.'
        }
    }
    Invoke-WithPython @('..\..\scripts\runtime_pipeline.py', 'build-course', $source)
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    Stop-Transcript | Out-Null
}
""",
    "Validate.ps1": r"""$ErrorActionPreference = 'Stop'
$workspaceDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $workspaceDir

function Get-PythonLauncher {
    if (Get-Command py -ErrorAction SilentlyContinue) { return @('py', '-3') }
    if (Get-Command python -ErrorAction SilentlyContinue) { return @('python') }
    throw 'Python launcher not found. Install Python 3 and ensure py.exe or python.exe is on PATH.'
}

function Invoke-WithPython {
    param([string[]]$CommandArgs)
    $launcher = Get-PythonLauncher
    if ($launcher.Count -gt 1) {
        & $launcher[0] $launcher[1..($launcher.Count - 1)] @CommandArgs
    } else {
        & $launcher[0] @CommandArgs
    }
}

New-Item -ItemType Directory -Force -Path 'runtime/logs' | Out-Null
$timestamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$logPath = Join-Path $workspaceDir "runtime/logs/validate-$timestamp.log"
Start-Transcript -Path $logPath -Append | Out-Null

try {
    Write-Host "[$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] Validate.ps1 started"
    Write-Host "Workspace: $workspaceDir"
    $source = if ($args.Count -gt 0) { $args[0] } else { '' }
    if ([string]::IsNullOrWhiteSpace($source)) {
        $items = Get-ChildItem 'runtime/incoming' -Force | Where-Object { $_.Name -notlike '._*' -and $_.Name -ne '.DS_Store' }
        $zipItems = $items | Where-Object { -not $_.PSIsContainer -and $_.Extension -ieq '.zip' }
        if ($items.Count -ne 1 -or $zipItems.Count -ne 1) {
            throw 'runtime/incoming must contain exactly one item, and it must be a .zip file. Otherwise pass an explicit source path.'
        }
    }
    Invoke-WithPython @('..\..\scripts\runtime_pipeline.py', 'validate', $source)
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    Stop-Transcript | Out-Null
}
""",
    "Repair-Submission.ps1": r"""$ErrorActionPreference = 'Stop'
$workspaceDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $workspaceDir

function Get-PythonLauncher {
    if (Get-Command py -ErrorAction SilentlyContinue) { return @('py', '-3') }
    if (Get-Command python -ErrorAction SilentlyContinue) { return @('python') }
    throw 'Python launcher not found. Install Python 3 and ensure py.exe or python.exe is on PATH.'
}

function Invoke-WithPython {
    param([string[]]$CommandArgs)
    $launcher = Get-PythonLauncher
    if ($launcher.Count -gt 1) {
        & $launcher[0] $launcher[1..($launcher.Count - 1)] @CommandArgs
    } else {
        & $launcher[0] @CommandArgs
    }
}

New-Item -ItemType Directory -Force -Path 'runtime/logs' | Out-Null
$timestamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$logPath = Join-Path $workspaceDir "runtime/logs/repair-submission-$timestamp.log"
Start-Transcript -Path $logPath -Append | Out-Null

try {
    Write-Host "[$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] Repair-Submission.ps1 started"
    Write-Host "Workspace: $workspaceDir"
    $source = if ($args.Count -gt 0) { $args[0] } else { '' }
    if ([string]::IsNullOrWhiteSpace($source)) {
        $items = Get-ChildItem 'runtime/incoming' -Force | Where-Object { $_.Name -notlike '._*' -and $_.Name -ne '.DS_Store' }
        $zipItems = $items | Where-Object { -not $_.PSIsContainer -and $_.Extension -ieq '.zip' }
        if ($items.Count -ne 1 -or $zipItems.Count -ne 1) {
            throw 'runtime/incoming must contain exactly one item, and it must be a .zip file. Otherwise pass an explicit source path.'
        }
    }
    Invoke-WithPython @('..\..\scripts\runtime_pipeline.py', 'repair-submission', $source)
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    Stop-Transcript | Out-Null
}
""",
    "Validate-All-Incoming.ps1": r"""$ErrorActionPreference = 'Stop'
$workspaceDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $workspaceDir

function Get-PythonLauncher {
    if (Get-Command py -ErrorAction SilentlyContinue) { return @('py', '-3') }
    if (Get-Command python -ErrorAction SilentlyContinue) { return @('python') }
    throw 'Python launcher not found. Install Python 3 and ensure py.exe or python.exe is on PATH.'
}

function Invoke-WithPython {
    param([string[]]$CommandArgs)
    $launcher = Get-PythonLauncher
    if ($launcher.Count -gt 1) {
        & $launcher[0] $launcher[1..($launcher.Count - 1)] @CommandArgs
    } else {
        & $launcher[0] @CommandArgs
    }
}

New-Item -ItemType Directory -Force -Path 'runtime/logs' | Out-Null
$timestamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$logPath = Join-Path $workspaceDir "runtime/logs/validate-all-incoming-$timestamp.log"
Start-Transcript -Path $logPath -Append | Out-Null

try {
    Write-Host "[$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] Validate-All-Incoming.ps1 started"
    Write-Host "Workspace: $workspaceDir"
    Invoke-WithPython @('..\..\scripts\runtime_pipeline.py', 'validate-all-incoming')
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    Stop-Transcript | Out-Null
}
""",
    "Build-All.ps1": r"""$ErrorActionPreference = 'Stop'
$workspaceDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $workspaceDir

function Get-PythonLauncher {
    if (Get-Command py -ErrorAction SilentlyContinue) { return @('py', '-3') }
    if (Get-Command python -ErrorAction SilentlyContinue) { return @('python') }
    throw 'Python launcher not found. Install Python 3 and ensure py.exe or python.exe is on PATH.'
}

function Invoke-WithPython {
    param([string[]]$CommandArgs)
    $launcher = Get-PythonLauncher
    if ($launcher.Count -gt 1) {
        & $launcher[0] $launcher[1..($launcher.Count - 1)] @CommandArgs
    } else {
        & $launcher[0] @CommandArgs
    }
}

New-Item -ItemType Directory -Force -Path 'runtime/logs' | Out-Null
$timestamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$logPath = Join-Path $workspaceDir "runtime/logs/build-all-$timestamp.log"
Start-Transcript -Path $logPath -Append | Out-Null

try {
    Write-Host "[$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] Build-All.ps1 started"
    Write-Host "Workspace: $workspaceDir"
    Invoke-WithPython @('..\..\scripts\runtime_pipeline.py', 'build-all')
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    Stop-Transcript | Out-Null
}
""",
    "Build-One-Course.ps1": r"""$ErrorActionPreference = 'Stop'
$workspaceDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $workspaceDir

function Get-PythonLauncher {
    if (Get-Command py -ErrorAction SilentlyContinue) { return @('py', '-3') }
    if (Get-Command python -ErrorAction SilentlyContinue) { return @('python') }
    throw 'Python launcher not found. Install Python 3 and ensure py.exe or python.exe is on PATH.'
}

function Invoke-WithPython {
    param([string[]]$CommandArgs)
    $launcher = Get-PythonLauncher
    if ($launcher.Count -gt 1) {
        & $launcher[0] $launcher[1..($launcher.Count - 1)] @CommandArgs
    } else {
        & $launcher[0] @CommandArgs
    }
}

New-Item -ItemType Directory -Force -Path 'runtime/logs' | Out-Null
$timestamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$logPath = Join-Path $workspaceDir "runtime/logs/build-one-course-$timestamp.log"
Start-Transcript -Path $logPath -Append | Out-Null

try {
    Write-Host "[$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] Build-One-Course.ps1 started"
    Write-Host "Workspace: $workspaceDir"
    $courseFolderName = if ($args.Count -gt 0) { $args[0] } else { '' }
    if ([string]::IsNullOrWhiteSpace($courseFolderName)) {
        $courseList = Get-ChildItem 'runtime/workspaces' -Recurse -Filter '*-main.tex' -File -ErrorAction SilentlyContinue |
            Where-Object { $_.FullName -like '*\Course-main-tex\*' } |
            ForEach-Object { $_.BaseName -replace '-main$', '' } |
            Sort-Object -Unique
        if (-not $courseList) {
            throw 'No courses found in runtime/workspaces.'
        }
        Write-Host 'Available courses:'
        for ($i = 0; $i -lt $courseList.Count; $i++) {
            Write-Host ("  {0}. {1}" -f ($i + 1), $courseList[$i])
        }
        Write-Host ''
        $selection = Read-Host 'Enter the course number or folder name to rebuild'
        if ([string]::IsNullOrWhiteSpace($selection)) {
            throw 'No course selection provided.'
        }
        $index = 0
        if ([int]::TryParse($selection, [ref]$index)) {
            if ($index -lt 1 -or $index -gt $courseList.Count) {
                throw "Invalid course number: $selection"
            }
            $courseFolderName = $courseList[$index - 1]
        } else {
            $courseFolderName = $selection
        }
    }
    Invoke-WithPython @('..\..\scripts\runtime_pipeline.py', 'build-one', $courseFolderName)
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    Stop-Transcript | Out-Null
}
""",
    "Check-LMS-Updates.ps1": r"""$ErrorActionPreference = 'Stop'
$workspaceDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $workspaceDir

function Get-PythonLauncher {
    if (Get-Command py -ErrorAction SilentlyContinue) { return @('py', '-3') }
    if (Get-Command python -ErrorAction SilentlyContinue) { return @('python') }
    throw 'Python launcher not found. Install Python 3 and ensure py.exe or python.exe is on PATH.'
}

function Invoke-WithPython {
    param([string[]]$CommandArgs)
    $launcher = Get-PythonLauncher
    if ($launcher.Count -gt 1) {
        & $launcher[0] $launcher[1..($launcher.Count - 1)] @CommandArgs
    } else {
        & $launcher[0] @CommandArgs
    }
}

New-Item -ItemType Directory -Force -Path 'runtime/logs' | Out-Null
$timestamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$logPath = Join-Path $workspaceDir "runtime/logs/check-lms-updates-$timestamp.log"
Start-Transcript -Path $logPath -Append | Out-Null

try {
    Write-Host "[$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] Check-LMS-Updates.ps1 started"
    Write-Host "Workspace: $workspaceDir"
    $commandArgs = @('..\..\scripts\check_lms_updates.py') + $args
    Invoke-WithPython $commandArgs
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    Stop-Transcript | Out-Null
}
""",
}

BATCH_TEMPLATES = {
    "Process-Instructor-Zip.bat": r"""@echo off
setlocal
set "ACTION=Process-Instructor-Zip"
set "LAUNCHER_PS1=%~dp0Process-Instructor-Zip.ps1"
set "INTERACTIVE=0"
if "%~1"=="" set "INTERACTIVE=1"
echo Launching action "%ACTION%"
echo PowerShell launcher: "%LAUNCHER_PS1%"
powershell -NoProfile -ExecutionPolicy Bypass -File "%LAUNCHER_PS1%" %*
set "CODE=%ERRORLEVEL%"
if not "%CODE%"=="0" (
  echo.
  echo Action "%ACTION%" failed.
  echo Launcher path: "%LAUNCHER_PS1%"
  echo Exit code: %CODE%
)
if "%INTERACTIVE%"=="1" (
  echo.
  if "%CODE%"=="0" (
    echo Action "%ACTION%" finished.
  ) else (
    echo Review the output above before closing this window.
  )
  echo Press any key to close...
  pause >nul
)
exit /b %CODE%
""",
    "Validate.bat": r"""@echo off
setlocal
set "ACTION=Validate"
set "LAUNCHER_PS1=%~dp0Validate.ps1"
set "INTERACTIVE=0"
if "%~1"=="" set "INTERACTIVE=1"
echo Launching action "%ACTION%"
echo PowerShell launcher: "%LAUNCHER_PS1%"
powershell -NoProfile -ExecutionPolicy Bypass -File "%LAUNCHER_PS1%" %*
set "CODE=%ERRORLEVEL%"
if not "%CODE%"=="0" (
  echo.
  echo Action "%ACTION%" failed.
  echo Launcher path: "%LAUNCHER_PS1%"
  echo Exit code: %CODE%
)
if "%INTERACTIVE%"=="1" (
  echo.
  if "%CODE%"=="0" (
    echo Action "%ACTION%" finished.
  ) else (
    echo Review the output above before closing this window.
  )
  echo Press any key to close...
  pause >nul
)
exit /b %CODE%
""",
    "Repair-Submission.bat": r"""@echo off
setlocal
set "ACTION=Repair-Submission"
set "LAUNCHER_PS1=%~dp0Repair-Submission.ps1"
set "INTERACTIVE=0"
if "%~1"=="" set "INTERACTIVE=1"
echo Launching action "%ACTION%"
echo PowerShell launcher: "%LAUNCHER_PS1%"
powershell -NoProfile -ExecutionPolicy Bypass -File "%LAUNCHER_PS1%" %*
set "CODE=%ERRORLEVEL%"
if not "%CODE%"=="0" (
  echo.
  echo Action "%ACTION%" failed.
  echo Launcher path: "%LAUNCHER_PS1%"
  echo Exit code: %CODE%
)
if "%INTERACTIVE%"=="1" (
  echo.
  if "%CODE%"=="0" (
    echo Action "%ACTION%" finished.
  ) else (
    echo Review the output above before closing this window.
  )
  echo Press any key to close...
  pause >nul
)
exit /b %CODE%
""",
    "Validate-All-Incoming.bat": r"""@echo off
setlocal
set "ACTION=Validate-All-Incoming"
set "LAUNCHER_PS1=%~dp0Validate-All-Incoming.ps1"
set "INTERACTIVE=0"
if "%~1"=="" set "INTERACTIVE=1"
echo Launching action "%ACTION%"
echo PowerShell launcher: "%LAUNCHER_PS1%"
powershell -NoProfile -ExecutionPolicy Bypass -File "%LAUNCHER_PS1%" %*
set "CODE=%ERRORLEVEL%"
if not "%CODE%"=="0" (
  echo.
  echo Action "%ACTION%" failed.
  echo Launcher path: "%LAUNCHER_PS1%"
  echo Exit code: %CODE%
)
if "%INTERACTIVE%"=="1" (
  echo.
  if "%CODE%"=="0" (
    echo Action "%ACTION%" finished.
  ) else (
    echo Review the output above before closing this window.
  )
  echo Press any key to close...
  pause >nul
)
exit /b %CODE%
""",
    "Build-All.bat": r"""@echo off
setlocal
set "ACTION=Build-All"
set "LAUNCHER_PS1=%~dp0Build-All.ps1"
set "INTERACTIVE=0"
if "%~1"=="" set "INTERACTIVE=1"
echo Launching action "%ACTION%"
echo PowerShell launcher: "%LAUNCHER_PS1%"
powershell -NoProfile -ExecutionPolicy Bypass -File "%LAUNCHER_PS1%" %*
set "CODE=%ERRORLEVEL%"
if not "%CODE%"=="0" (
  echo.
  echo Action "%ACTION%" failed.
  echo Launcher path: "%LAUNCHER_PS1%"
  echo Exit code: %CODE%
)
if "%INTERACTIVE%"=="1" (
  echo.
  if "%CODE%"=="0" (
    echo Action "%ACTION%" finished.
  ) else (
    echo Review the output above before closing this window.
  )
  echo Press any key to close...
  pause >nul
)
exit /b %CODE%
""",
    "Build-One-Course.bat": r"""@echo off
setlocal
set "ACTION=Build-One-Course"
set "LAUNCHER_PS1=%~dp0Build-One-Course.ps1"
set "INTERACTIVE=0"
if "%~1"=="" set "INTERACTIVE=1"
echo Launching action "%ACTION%"
echo PowerShell launcher: "%LAUNCHER_PS1%"
powershell -NoProfile -ExecutionPolicy Bypass -File "%LAUNCHER_PS1%" %*
set "CODE=%ERRORLEVEL%"
if not "%CODE%"=="0" (
  echo.
  echo Action "%ACTION%" failed.
  echo Launcher path: "%LAUNCHER_PS1%"
  echo Exit code: %CODE%
)
if "%INTERACTIVE%"=="1" (
  echo.
  if "%CODE%"=="0" (
    echo Action "%ACTION%" finished.
  ) else (
    echo Review the output above before closing this window.
  )
  echo Press any key to close...
  pause >nul
)
exit /b %CODE%
""",
    "Check-LMS-Updates.bat": r"""@echo off
setlocal
set "ACTION=Check-LMS-Updates"
set "LAUNCHER_PS1=%~dp0Check-LMS-Updates.ps1"
set "INTERACTIVE=0"
if "%~1"=="" set "INTERACTIVE=1"
echo Launching action "%ACTION%"
echo PowerShell launcher: "%LAUNCHER_PS1%"
powershell -NoProfile -ExecutionPolicy Bypass -File "%LAUNCHER_PS1%" %*
set "CODE=%ERRORLEVEL%"
if not "%CODE%"=="0" (
  echo.
  echo Action "%ACTION%" failed.
  echo Launcher path: "%LAUNCHER_PS1%"
  echo Exit code: %CODE%
)
if "%INTERACTIVE%"=="1" (
  echo.
  if "%CODE%"=="0" (
    echo Action "%ACTION%" finished.
  ) else (
    echo Review the output above before closing this window.
  )
  echo Press any key to close...
  pause >nul
)
exit /b %CODE%
""",
}

PROGRAM_EXECUTABLE_TEMPLATE = """#!/usr/bin/env python3
\"\"\"Interactive launcher for one program workspace.\"\"\"

from __future__ import annotations

import argparse
import platform
import subprocess
import sys
from pathlib import Path

WORKSPACE_ROOT = Path(__file__).resolve().parent

ACTIONS = {
    "process-instructor-zip": "Process-Instructor-Zip",
    "validate": "Validate",
    "repair-submission": "Repair-Submission",
    "validate-all-incoming": "Validate-All-Incoming",
    "build-all": "Build-All",
    "build-one-course": "Build-One-Course",
    "check-lms-updates": "Check-LMS-Updates",
}


def launcher_extension() -> str:
    return ".bat" if platform.system().lower() == "windows" else ".command"


def run_action(action: str, passthrough: list[str]) -> int:
    base = ACTIONS[action]
    ext = launcher_extension()
    launcher = WORKSPACE_ROOT / f"{base}{ext}"
    if not launcher.is_file():
        print(f"Launcher not found: {launcher}", file=sys.stderr)
        return 1
    if ext == ".bat":
        cmd = ["cmd", "/c", str(launcher), *passthrough]
    else:
        cmd = [str(launcher), *passthrough]
    print(f"Dispatching action '{action}' via launcher: {launcher}")
    print(f"Command: {' '.join(cmd)}")
    try:
        proc = subprocess.run(cmd, cwd=WORKSPACE_ROOT)
    except OSError as exc:
        print(
            f"Action '{action}' failed to launch. launcher={launcher} error={exc}",
            file=sys.stderr,
        )
        return 1
    if proc.returncode != 0:
        print(
            f"Action '{action}' failed. launcher={launcher} exit_code={proc.returncode}",
            file=sys.stderr,
        )
    return proc.returncode


def pick_action_interactive() -> tuple[str, list[str]] | None:
    keys = sorted(ACTIONS.keys())
    print("Program Actions")
    print()
    for idx, key in enumerate(keys, start=1):
        print(f"{idx}. {key}")
    print("0. exit")
    print()
    while True:
        raw = input("Choose an action number: ").strip().lower()
        if raw in {"0", "q", "quit", "exit"}:
            return None
        if not raw.isdigit():
            print("Enter a number from the list.")
            continue
        choice = int(raw)
        if choice < 1 or choice > len(keys):
            print("Choice out of range.")
            continue
        return keys[choice - 1], []


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        prog="CourseProgram",
        description="Run program-level CourseDisplay actions through one OS-aware entrypoint.",
    )
    parser.add_argument("action", choices=sorted(ACTIONS.keys()))
    parser.add_argument("args", nargs=argparse.REMAINDER, help="Arguments passed through to the selected action")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = list(sys.argv[1:] if argv is None else argv)
    if not args:
        picked = pick_action_interactive()
        if picked is None:
            return 0
        action, passthrough = picked
        return run_action(action, passthrough)

    parsed = parse_args(args)
    passthrough = list(parsed.args)
    if passthrough and passthrough[0] == "--":
        passthrough = passthrough[1:]
    return run_action(parsed.action, passthrough)


if __name__ == "__main__":
    raise SystemExit(main())
"""

PROGRAM_BATCH_WRAPPER_TEMPLATE = r"""@echo off
setlocal
set "SCRIPT=%~dp0CourseProgram"
set "INTERACTIVE=0"
if "%~1"=="" set "INTERACTIVE=1"
where py >nul 2>nul
if %ERRORLEVEL% EQU 0 (
  py -3 "%SCRIPT%" %*
  set "CODE=%ERRORLEVEL%"
  goto :done
)
where python >nul 2>nul
if %ERRORLEVEL% EQU 0 (
  python "%SCRIPT%" %*
  set "CODE=%ERRORLEVEL%"
  goto :done
)
echo Python launcher not found. Install Python 3 and ensure py.exe or python.exe is on PATH.
set "CODE=1"
:done
if not "%CODE%"=="0" (
  echo.
  echo CourseProgram failed with exit code %CODE%.
)
if "%INTERACTIVE%"=="1" (
  echo.
  if "%CODE%"=="0" (
    echo CourseProgram finished.
  ) else (
    echo Review the output above before closing this window.
  )
  echo Press any key to close...
  pause >nul
)
exit /b %CODE%
"""


def validate_program_name(name: str) -> str:
    cleaned = name.strip()
    if not cleaned:
        raise SystemExit("Program name must not be empty.")
    if "/" in cleaned or cleaned in {".", ".."}:
        raise SystemExit("Program name must be a single folder name.")
    return cleaned


def ensure_new_workspace(path: Path, force: bool) -> None:
    if path.exists() and any(path.iterdir()) and not force:
        raise SystemExit(f"Target workspace already exists and is not empty: {path}")
    path.mkdir(parents=True, exist_ok=True)


def write_text(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def write_registry_csv(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=REGISTRY_HEADERS)
        writer.writeheader()


def write_course_notes_csv(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=COURSE_NOTES_HEADERS)
        writer.writeheader()


def write_lms_watch_csv(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=LMS_WATCH_HEADERS)
        writer.writeheader()


def read_csv_rows(path: Path) -> tuple[list[str], list[dict[str, str]]]:
    if not path.is_file():
        return [], []
    with path.open(newline="", encoding="utf-8-sig") as handle:
        reader = csv.DictReader(handle)
        fieldnames = list(reader.fieldnames or [])
        if not fieldnames:
            return [], []
        rows = [{name: (row.get(name) or "") for name in fieldnames} for row in reader]
        return fieldnames, rows


def write_csv_rows(path: Path, fieldnames: list[str], rows: list[dict[str, str]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=fieldnames)
        writer.writeheader()
        for row in rows:
            writer.writerow({name: row.get(name, "") for name in fieldnames})


def ensure_csv_headers(path: Path, required_headers: list[str]) -> bool:
    fieldnames, rows = read_csv_rows(path)
    if not fieldnames:
        write_csv_rows(path, list(required_headers), [])
        return True
    extra_headers = [name for name in fieldnames if name not in required_headers]
    normalized = list(required_headers) + extra_headers
    missing = any(name not in fieldnames for name in required_headers)
    if missing or normalized != fieldnames:
        write_csv_rows(path, normalized, rows)
        return True
    return False


def load_registry_rows(registry_path: Path) -> list[dict[str, str]]:
    _fieldnames, rows = read_csv_rows(registry_path)
    output: list[dict[str, str]] = []
    for row in rows:
        code = (row.get("course_code") or "").strip()
        if not code:
            continue
        output.append(
            {
                "course_code": code,
                "course_name": (row.get("course_name") or "").strip(),
                "semester_workspace": (row.get("semester_workspace") or "").strip(),
                "course_folder_name": (row.get("course_folder_name") or "").strip(),
            }
        )
    return output


def sync_lms_watch_courses_from_registry(workspace: Path) -> bool:
    registry_path = workspace / "config" / "staff" / "course_registry.csv"
    lms_path = workspace / "config" / "staff" / "lms_watch_courses.csv"
    ensure_csv_headers(lms_path, LMS_WATCH_HEADERS)
    fieldnames, rows = read_csv_rows(lms_path)
    if not fieldnames:
        fieldnames = list(LMS_WATCH_HEADERS)

    changed = False
    existing_by_code: dict[str, dict[str, str]] = {}
    for row in rows:
        code = (row.get("course_code") or "").strip()
        if code and code not in existing_by_code:
            existing_by_code[code] = row

    for entry in load_registry_rows(registry_path):
        code = entry["course_code"]
        row = existing_by_code.get(code)
        if row is None:
            new_row = {name: "" for name in fieldnames}
            for name in LMS_WATCH_HEADERS:
                new_row.setdefault(name, "")
            new_row["enabled"] = "FALSE"
            new_row["course_code"] = code
            new_row["course_name"] = entry["course_name"]
            rows.append(new_row)
            existing_by_code[code] = new_row
            changed = True
            continue

        if not (row.get("course_name") or "").strip() and entry["course_name"]:
            row["course_name"] = entry["course_name"]
            changed = True

    if changed:
        write_csv_rows(lms_path, fieldnames, rows)
    return changed


def ensure_lms_watch_settings_json(path: Path) -> bool:
    defaults: dict[str, Any] = json.loads(LMS_WATCH_SETTINGS)
    if not path.is_file():
        write_text(path, json.dumps(defaults, indent=2) + "\n")
        return True
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return False
    if not isinstance(data, dict):
        return False
    changed = False
    for key, value in defaults.items():
        if key not in data:
            data[key] = value
            changed = True
    if changed:
        write_text(path, json.dumps(data, indent=2) + "\n")
    return changed


def write_workspace_launchers(workspace: Path) -> None:
    for name, content in COMMAND_TEMPLATES.items():
        command_path = workspace / name
        write_text(command_path, content)
        command_path.chmod(0o755)
    for name, content in POWERSHELL_TEMPLATES.items():
        write_text(workspace / name, content)
    for name, content in BATCH_TEMPLATES.items():
        write_text(workspace / name, content)
    program_exec = workspace / "CourseProgram"
    write_text(program_exec, PROGRAM_EXECUTABLE_TEMPLATE)
    program_exec.chmod(0o755)
    write_text(workspace / "CourseProgram.bat", PROGRAM_BATCH_WRAPPER_TEMPLATE)


def scaffold_workspace(workspace: Path, *, refresh_launchers: bool) -> None:
    (workspace / "config" / "staff" / "expected_counts").mkdir(parents=True, exist_ok=True)
    (workspace / "config" / "generated").mkdir(parents=True, exist_ok=True)
    (workspace / "config" / "assets").mkdir(parents=True, exist_ok=True)
    for relpath in RUNTIME_DIRS:
        (workspace / "runtime" / relpath).mkdir(parents=True, exist_ok=True)
    for relpath in TRACKED_DIRS:
        (workspace / "tracked" / relpath).mkdir(parents=True, exist_ok=True)

    ensure_csv_headers(workspace / "config" / "staff" / "course_registry.csv", REGISTRY_HEADERS)
    ensure_csv_headers(workspace / "config" / "staff" / "course_notes.csv", COURSE_NOTES_HEADERS)
    ensure_csv_headers(workspace / "config" / "staff" / "lms_watch_courses.csv", LMS_WATCH_HEADERS)
    ensure_lms_watch_settings_json(workspace / "config" / "staff" / "lms_watch_settings.json")
    sync_lms_watch_courses_from_registry(workspace)
    (workspace / "config" / "staff" / "pdf_layout_overrides").mkdir(parents=True, exist_ok=True)
    write_text(workspace / "config" / "README.md", CONFIG_README)
    if refresh_launchers:
        write_workspace_launchers(workspace)

    os.environ["COURSEDISPLAY_WORKSPACE_ROOT"] = str(workspace)
    sync_expected_counts(
        registry_path=workspace / "config" / "staff" / "course_registry.csv",
        counts_dir=workspace / "config" / "staff" / "expected_counts",
        output_json=workspace / "config" / "generated" / "expected_counts.json",
        output_tex=workspace / "config" / "generated" / "expected_counts.tex",
    )
    sync_pdf_layout_overrides(
        overrides_dir=workspace / "config" / "staff" / "pdf_layout_overrides",
        output_json=workspace / "config" / "generated" / "pdf_layout_overrides.json",
        output_tex=workspace / "config" / "generated" / "pdf_layout_overrides.tex",
    )

    for filename in EXPECTED_COUNT_FILES:
        csv_path = workspace / "config" / "staff" / "expected_counts" / filename
        ensure_csv_headers(
            csv_path,
            ["semester_workspace", "course_code", "course_folder_name", "expected_count"],
        )


def backfill_workspace(workspace: Path, *, refresh_launchers: bool = True) -> bool:
    staff_dir = workspace / "config" / "staff"
    if not staff_dir.exists():
        return False
    scaffold_workspace(workspace, refresh_launchers=refresh_launchers)
    return True


def backfill_all_workspaces(programs_root: Path, *, refresh_launchers: bool = True) -> list[Path]:
    updated: list[Path] = []
    if not programs_root.is_dir():
        return updated
    for workspace in sorted(p for p in programs_root.iterdir() if p.is_dir() and not p.name.startswith(".")):
        if backfill_workspace(workspace, refresh_launchers=refresh_launchers):
            updated.append(workspace)
    return updated


def create_workspace(program_name: str, programs_root: Path, force: bool) -> Path:
    workspace = programs_root / program_name
    ensure_new_workspace(workspace, force=force)
    scaffold_workspace(workspace, refresh_launchers=True)
    return workspace


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Create a new program workspace scaffold")
    parser.add_argument("program_name", nargs="?", help="Folder name for the new program workspace")
    parser.add_argument(
        "--programs-root",
        default=str(DEFAULT_PROGRAMS_ROOT),
        help="Parent directory that contains program workspaces",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Allow writing into an existing workspace folder",
    )
    parser.add_argument(
        "--backfill-all",
        action="store_true",
        help="Apply the latest scaffold/migrations to all existing program workspaces without clearing staff data",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    programs_root = Path(args.programs_root).resolve()
    if args.backfill_all:
        for workspace in backfill_all_workspaces(programs_root, refresh_launchers=True):
            print(workspace)
        return 0
    if not args.program_name:
        raise SystemExit("program_name is required unless --backfill-all is provided")
    workspace = create_workspace(
        validate_program_name(args.program_name),
        programs_root,
        force=args.force,
    )
    print(workspace)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
