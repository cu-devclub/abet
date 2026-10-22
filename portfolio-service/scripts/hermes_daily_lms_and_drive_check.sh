#!/usr/bin/env bash
set -u

DATA_ROOT=${COURSEDISPLAY_DATA_ROOT:-/opt/data}
if [[ ! -d "$DATA_ROOT/workspace/CourseDisplay" && -n "${HOME:-}" && -d "$HOME/.hermes/workspace/CourseDisplay" ]]; then
  DATA_ROOT="$HOME/.hermes"
fi

REPO=${COURSEDISPLAY_REPO:-$DATA_ROOT/workspace/CourseDisplay}
WORKSPACE=${COURSEDISPLAY_WORKSPACE_ROOT:-$REPO/programs/ice-ay2025}
ENV_FILE=${COURSEDISPLAY_ENV_FILE:-$DATA_ROOT/secrets/coursedisplay-daily-lms.env}
PY="$REPO/.venv/bin/python"
STAMP=$(date -u +%Y%m%d-%H%M%S)
LOG_DIR="$WORKSPACE/runtime/logs"
LOG="$LOG_DIR/daily-lms-drive-check-hermes-$STAMP.log"
PENDING_UPLOADS="$WORKSPACE/runtime/lms-watch/pending_drive_uploads_latest.json"

mkdir -p "$LOG_DIR"

if [[ -f "$ENV_FILE" ]]; then
  # shellcheck disable=SC1090
  source "$ENV_FILE"
fi

export COURSEDISPLAY_WORKSPACE_ROOT="$WORKSPACE"
export PLAYWRIGHT_BROWSERS_PATH="${PLAYWRIGHT_BROWSERS_PATH:-$DATA_ROOT/ms-playwright}"
export COURSEDISPLAY_NOTIFY_DRIVE_FOLLOWUP_TEXT="${COURSEDISPLAY_NOTIFY_DRIVE_FOLLOWUP_TEXT:-After the zip file is uploaded to Google Drive, the course coordinator will be notified for local processing.}"
export COURSEDISPLAY_LMS_PROFILE_DIR="${COURSEDISPLAY_LMS_PROFILE_DIR:-$DATA_ROOT/coursedisplay/lms-watch/chrome-profile}"
if [[ -d "$DATA_ROOT/browser-libs/root/usr/lib/x86_64-linux-gnu" ]]; then
  export LD_LIBRARY_PATH="$DATA_ROOT/browser-libs/root/usr/lib/x86_64-linux-gnu:$DATA_ROOT/browser-libs/root/usr/lib:${LD_LIBRARY_PATH:-}"
fi
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$PATH"

cd "$REPO" || exit 1

if [[ ! -x "$PY" ]]; then
  echo "CourseDisplay Python venv is missing: $PY"
  exit 1
fi

"$PY" - <<'PY'
import os
import json
from pathlib import Path

p = Path("programs/ice-ay2025/config/staff/lms_watch_settings.json")
data = json.loads(p.read_text(encoding="utf-8"))
data["persistent_profile_scope"] = "program"
data["persistent_profile_dir"] = os.environ["COURSEDISPLAY_LMS_PROFILE_DIR"]
p.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
PY

{
  echo "CourseDisplay Hermes daily status"
  echo "Started UTC: $(date -u '+%Y-%m-%d %H:%M:%S')"
  echo "Workspace: $WORKSPACE"
  echo
} > "$LOG"

run_step() {
  local label=$1
  shift
  echo "== $label ==" >> "$LOG"
  "$@" >> "$LOG" 2>&1
  local code=$?
  echo "Exit code: $code" >> "$LOG"
  echo >> "$LOG"
  return "$code"
}

lms_code=0
pending_code=0
notify_code=0
drive_code=0

run_step "LMS watch" "$PY" scripts/check_lms_updates.py
lms_code=$?

if [[ "$lms_code" -eq 0 ]]; then
  run_step "LMS-to-Drive upload cross-check" "$PY" scripts/check_lms_drive_uploads.py --program-workspace programs/ice-ay2025 --output-json "$PENDING_UPLOADS" --max-items 20
  pending_code=$?

  notify_args=(scripts/notify_lms_updates.py --workspace "$WORKSPACE" --daily-report --pending-uploads "$PENDING_UPLOADS")
  case "${COURSEDISPLAY_NOTIFY_DRY_RUN:-0}" in
    1|true|TRUE|yes|YES) notify_args+=(--dry-run) ;;
  esac
  run_step "Staff daily email" "$PY" "${notify_args[@]}"
  notify_code=$?
else
  echo "== LMS-to-Drive upload cross-check ==" >> "$LOG"
  echo "Skipped because LMS watch exited with $lms_code." >> "$LOG"
  echo >> "$LOG"
  pending_code=1

  echo "== Staff daily email ==" >> "$LOG"
  echo "Skipped because LMS watch exited with $lms_code." >> "$LOG"
  echo >> "$LOG"
  notify_code=1
fi

run_step "Drive incoming check" "$PY" scripts/check_drive_incoming.py --program-workspace programs/ice-ay2025 --max-items 10
drive_code=$?

summary=$(PENDING_UPLOADS_JSON="$PENDING_UPLOADS" "$PY" - <<'PY'
import csv
import json
import os
from pathlib import Path

workspace = Path("programs/ice-ay2025")
csv_path = workspace / "runtime" / "output" / "reports" / "lms-watch" / "lms-watch-staff-latest.csv"
if not csv_path.is_file():
    print("LMS report: missing")
    raise SystemExit(0)
counts = {}
rows = []
with csv_path.open(newline="", encoding="utf-8-sig") as handle:
    reader = csv.DictReader(handle)
    for row in reader:
        rows.append(row)
        status = (row.get("status") or "unknown").strip().lower() or "unknown"
        counts[status] = counts.get(status, 0) + 1
changed = counts.get("changed", 0)
errors = counts.get("error", 0)
unchanged = counts.get("unchanged", 0)
baseline = counts.get("baseline-set", 0)
print(f"LMS rows: {len(rows)} total, {changed} changed, {errors} errors, {unchanged} unchanged, {baseline} baselines")
if changed:
    print("Changed LMS courses:")
    for row in rows:
        if (row.get("status") or "").strip().lower() == "changed":
            label = " ".join(part for part in [(row.get("course_code") or "").strip(), (row.get("course_name") or "").strip()] if part)
            item = (row.get("latest_item") or "").strip()
            print(f"- {label}: {item or 'latest item unknown'}")
if errors:
    print("LMS rows with errors:")
    shown = 0
    for row in rows:
        if (row.get("status") or "").strip().lower() == "error":
            label = " ".join(part for part in [(row.get("course_code") or "").strip(), (row.get("course_name") or "").strip()] if part)
            note = (row.get("note") or "").strip()
            print(f"- {label}: {note or 'no note'}")
            shown += 1
            if shown >= 8:
                break

pending_path = Path(os.environ.get("PENDING_UPLOADS_JSON", ""))
if pending_path.is_file():
    try:
        data = json.loads(pending_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        data = {}
    pending = data.get("pending", []) if isinstance(data, dict) else []
    resolved = data.get("resolved_this_run", []) if isinstance(data, dict) else []
    if isinstance(pending, list):
        print(f"Pending staff upload action items: {len(pending)}")
        for item in pending[:10]:
            if not isinstance(item, dict):
                continue
            label = " ".join(part for part in [(item.get("course_code") or "").strip(), (item.get("course_name") or "").strip()] if part)
            latest = (item.get("latest_item") or "").strip()
            print(f"- {label}: {latest or 'expected zip unknown'}")
    if isinstance(resolved, list) and resolved:
        print(f"Resolved staff upload action items this run: {len(resolved)}")
        for item in resolved[:10]:
            if not isinstance(item, dict):
                continue
            label = " ".join(part for part in [(item.get("course_code") or "").strip(), (item.get("course_name") or "").strip()] if part)
            matched = (item.get("matched_drive_name") or "").strip()
            print(f"- {label}: {matched or 'matched Drive zip'}")
PY
)

drive_summary=$("$PY" scripts/check_drive_incoming.py --program-workspace programs/ice-ay2025 --max-items 10 2>&1)
drive_summary_code=$?
if [[ "$drive_summary_code" -ne 0 ]]; then
  drive_code=$drive_summary_code
fi

overall=0
for code in "$lms_code" "$pending_code" "$notify_code" "$drive_code"; do
  if [[ "$code" -ne 0 ]]; then
    overall=1
  fi
done

{
  echo "Finished UTC: $(date -u '+%Y-%m-%d %H:%M:%S')"
  echo "Overall exit code: $overall"
} >> "$LOG"

status_label="PASS"
if [[ "$overall" -ne 0 ]]; then
  status_label="FAIL"
fi

echo "CourseDisplay daily status on Hermes: $status_label"
echo "LMS check: exit $lms_code"
echo "LMS-to-Drive cross-check: exit $pending_code"
echo "Staff email: exit $notify_code"
echo "Drive incoming check: exit $drive_code"
echo
echo "$summary"
echo
echo "$drive_summary"
echo
echo "Full log: $LOG"
echo "No Drive files were processed by Hermes."

exit "$overall"
