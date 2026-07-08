#!/usr/bin/env bash
#
# reset-db.sh — master database reset tool for the SFTP service.
#
#   ./scripts/reset-db.sh
#
# An INTERACTIVE, menu-driven tool for wiping selected parts of the database
# while keeping the rest intact. Built for the common cases:
#
#   • "Clear all files/folders but keep the users."
#   • "Storage usage still shows the old number after I deleted files."
#
# The second point is a data-integrity quirk, not a bug in this script:
# users.storage_used is a COUNTER maintained by the application. It is only
# decremented when a file is *hard*-deleted through the app. Deleting rows
# straight from the DB (or emptying trash outside the app) leaves the counter
# stale. Every destructive action here recomputes the counter from the actual
# files afterwards, and there is a dedicated menu entry to fix the counter
# WITHOUT deleting anything.
#
# What it will NEVER touch:
#   • goose_db_version  — migration bookkeeping (breaks migrations if wiped)
#   • roles / permissions / role_permissions — RBAC seed data (breaks login)
#
# Nothing is executed until you review a summary and type the confirmation word.
#
set -euo pipefail

# ── Locate the project + database ─────────────────────────────────────────
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_DIR"

PG_SERVICE="${PG_SERVICE:-postgres}"
BACKEND_SERVICE="${BACKEND_SERVICE:-backend}"
PG_USER="${POSTGRES_USER:-sftp}"
PG_DB="${POSTGRES_DB:-sftp}"

# Colours (fall back to plain text when not a TTY).
if [ -t 1 ]; then
  BOLD=$'\033[1m'; DIM=$'\033[2m'; RED=$'\033[31m'; GREEN=$'\033[32m'
  YELLOW=$'\033[33m'; CYAN=$'\033[36m'; RESET=$'\033[0m'
else
  BOLD=""; DIM=""; RED=""; GREEN=""; YELLOW=""; CYAN=""; RESET=""
fi

# Pick the compose command (v2 "docker compose" or legacy "docker-compose").
if docker compose version >/dev/null 2>&1; then
  DC=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  DC=(docker-compose)
else
  echo "${RED}Error: neither 'docker compose' nor 'docker-compose' is available.${RESET}" >&2
  exit 1
fi

# Run a psql command inside the postgres container.
psql_do() {
  "${DC[@]}" exec -T "$PG_SERVICE" psql -v ON_ERROR_STOP=1 -U "$PG_USER" -d "$PG_DB" "$@"
}

# Sanity check: is the DB reachable?
if ! psql_do -c '\q' >/dev/null 2>&1; then
  echo "${RED}Error: cannot reach the '${PG_SERVICE}' service / database '${PG_DB}'.${RESET}" >&2
  echo "Is the stack up?  Try:  ${DC[*]} up -d ${PG_SERVICE}" >&2
  exit 1
fi

# ── Category → table definitions ──────────────────────────────────────────
# Each category lists the tables it truncates. TRUNCATE ... CASCADE takes care
# of dependent rows (e.g. wiping "files" also clears file_versions, file_text,
# file_tags, favorites, embeddings, and any shares/permissions pointing at
# them). Order does not matter because of CASCADE.

declare -A CAT_LABEL CAT_TABLES CAT_NOTE
ORDER=(files tags shares uploads downloads activity notifications alerts sessions apikeys teams departments settings users)

CAT_LABEL[files]="Files & folders (content, versions, text, tags links, favorites)"
CAT_TABLES[files]="files folders"
CAT_NOTE[files]="Also purges the physical file blobs and recomputes storage usage."

CAT_LABEL[tags]="Tag definitions"
CAT_TABLES[tags]="tags"
CAT_NOTE[tags]="The user-defined tag list (file<->tag links go with the files)."

CAT_LABEL[shares]="Shares & resource permissions"
CAT_TABLES[shares]="shares resource_permissions"
CAT_NOTE[shares]="Public links + per-user/folder shares (implied if you wipe files)."

CAT_LABEL[uploads]="In-progress / resumable uploads"
CAT_TABLES[uploads]="uploads"
CAT_NOTE[uploads]="Pending multipart uploads and their chunks."

CAT_LABEL[downloads]="Download history"
CAT_TABLES[downloads]="downloads"
CAT_NOTE[downloads]="Audit trail of who downloaded what."

CAT_LABEL[activity]="Audit & activity logs"
CAT_TABLES[activity]="audit_logs user_activity login_history"
CAT_NOTE[activity]="System audit log, per-user activity, and login history."

CAT_LABEL[notifications]="Notifications"
CAT_TABLES[notifications]="notifications"
CAT_NOTE[notifications]="In-app notifications for all users."

CAT_LABEL[alerts]="Security alerts"
CAT_TABLES[alerts]="security_alerts"
CAT_NOTE[alerts]="Raised security alerts."

CAT_LABEL[sessions]="Sessions (logs everyone out)"
CAT_TABLES[sessions]="sessions"
CAT_NOTE[sessions]="Active login sessions — all users must sign in again."

CAT_LABEL[apikeys]="API keys"
CAT_TABLES[apikeys]="api_keys"
CAT_NOTE[apikeys]="Programmatic API keys will stop working."

CAT_LABEL[teams]="Teams & memberships"
CAT_TABLES[teams]="teams team_members"
CAT_NOTE[teams]="${YELLOW}Warning:${RESET} deletes team-owned files too (files.team_id CASCADE)."

CAT_LABEL[departments]="Departments"
CAT_TABLES[departments]="departments"
CAT_NOTE[departments]="Users keep existing; their department link is cleared."

CAT_LABEL[settings]="User settings / preferences"
CAT_TABLES[settings]="settings"
CAT_NOTE[settings]="Per-user preferences revert to defaults."

CAT_LABEL[users]="USERS (accounts, profiles, everything they own)"
CAT_TABLES[users]="users"
CAT_NOTE[users]="${RED}DANGER:${RESET} removes all accounts incl. admins. RBAC roles are kept."

# ── Selection state ───────────────────────────────────────────────────────
declare -A SELECTED
for c in "${ORDER[@]}"; do SELECTED[$c]=0; done
PURGE_BLOBS=0
RECOMPUTE=1   # storage recompute is on by default; it is always safe

reset_selection() { for c in "${ORDER[@]}"; do SELECTED[$c]=0; done; PURGE_BLOBS=0; }

# ── Presets ───────────────────────────────────────────────────────────────
preset_files_only() {
  reset_selection
  SELECTED[files]=1; SELECTED[shares]=1; SELECTED[uploads]=1; SELECTED[downloads]=1
  PURGE_BLOBS=1
}
preset_all_content() {
  preset_files_only
  SELECTED[tags]=1
}
preset_content_activity() {
  preset_all_content
  SELECTED[activity]=1; SELECTED[notifications]=1; SELECTED[alerts]=1
}
preset_keep_users_only() {
  # Everything that isn't the users themselves or RBAC seed data.
  reset_selection
  for c in files tags shares uploads downloads activity notifications alerts \
           sessions apikeys teams departments settings; do
    SELECTED[$c]=1
  done
  PURGE_BLOBS=1
}

# ── Rendering ─────────────────────────────────────────────────────────────
ask() { # ask "prompt" "default(y/n)" -> returns 0 for yes
  local prompt="$1" def="${2:-n}" ans
  local hint="[y/N]"; [ "$def" = "y" ] && hint="[Y/n]"
  read -r -p "$prompt $hint " ans || true
  ans="${ans:-$def}"
  [[ "$ans" =~ ^[Yy] ]]
}

show_summary() {
  echo
  echo "${BOLD}Planned actions${RESET}"
  echo "${DIM}────────────────────────────────────────────────────────────${RESET}"
  local any=0
  for c in "${ORDER[@]}"; do
    if [ "${SELECTED[$c]}" = "1" ]; then
      printf "  ${RED}WIPE${RESET}  %s\n" "${CAT_LABEL[$c]}"
      printf "        ${DIM}tables: %s${RESET}\n" "${CAT_TABLES[$c]}"
      any=1
    fi
  done
  [ "$any" = "0" ] && echo "  ${DIM}(no tables selected for deletion)${RESET}"
  echo
  if [ "$PURGE_BLOBS" = "1" ]; then
    echo "  ${RED}PURGE${RESET} physical file blobs in the '${BACKEND_SERVICE}' volume (/app/storage/files, /app/storage/tmp)"
  fi
  if [ "$RECOMPUTE" = "1" ]; then
    echo "  ${GREEN}FIX${RESET}   recompute users.storage_used and teams.storage_used from actual files"
  fi
  echo
  echo "  ${GREEN}KEEP${RESET}  goose_db_version, roles, permissions, role_permissions (always)"
  local keep=""
  for c in "${ORDER[@]}"; do
    [ "${SELECTED[$c]}" = "0" ] && keep+=" ${CAT_TABLES[$c]}"
  done
  [ -n "$keep" ] && printf "        ${DIM}also keeping:%s${RESET}\n" "$keep"
  echo "${DIM}────────────────────────────────────────────────────────────${RESET}"
}

# ── Execution ─────────────────────────────────────────────────────────────
run_recompute() {
  echo "${CYAN}==> Recomputing storage counters…${RESET}"
  psql_do <<'SQL'
UPDATE users u SET storage_used = COALESCE((
  SELECT sum(sz) FROM (
    SELECT f.size_bytes AS sz
      FROM files f
     WHERE f.owner_id = u.id AND f.is_common = false
    UNION ALL
    SELECT fv.size_bytes
      FROM file_versions fv
      JOIN files f ON f.id = fv.file_id
     WHERE f.owner_id = u.id AND f.is_common = false
  ) x
), 0), updated_at = now();

UPDATE teams t SET storage_used = COALESCE((
  SELECT sum(f.size_bytes) FROM files f
   WHERE f.team_id = t.id AND f.is_common = false
), 0), updated_at = now();
SQL
  echo "${GREEN}    storage counters now reflect actual files.${RESET}"
}

run_purge_blobs() {
  echo "${CYAN}==> Purging physical file blobs…${RESET}"
  if "${DC[@]}" exec -T "$BACKEND_SERVICE" sh -c \
      'rm -rf /app/storage/files/* /app/storage/tmp/* 2>/dev/null; echo ok' >/dev/null 2>&1; then
    echo "${GREEN}    blob storage emptied.${RESET}"
  else
    echo "${YELLOW}    could not reach the '${BACKEND_SERVICE}' container — blobs left on disk.${RESET}"
    echo "${YELLOW}    (DB rows are gone; orphaned blobs are harmless but waste space.)${RESET}"
  fi
}

execute() {
  # Collect the tables to clear.
  local tables=()
  for c in "${ORDER[@]}"; do
    if [ "${SELECTED[$c]}" = "1" ]; then
      for t in ${CAT_TABLES[$c]}; do tables+=("$t"); done
    fi
  done

  if [ "${#tables[@]}" -gt 0 ]; then
    # De-duplicate.
    local uniq_tables; uniq_tables=$(printf '%s\n' "${tables[@]}" | sort -u)
    echo "${CYAN}==> Deleting rows from:${RESET} $(echo "$uniq_tables" | paste -sd, -)"
    # We use DELETE (not TRUNCATE) on purpose. TRUNCATE ... CASCADE would
    # STRUCTURALLY truncate every table with an FK pointing at the target —
    # e.g. users.department_id -> departments, so truncating departments would
    # also wipe users. DELETE honours each FK's ON DELETE rule instead
    # (SET NULL keeps the parent row, CASCADE removes intended children), so
    # kept tables such as users are never harmed. Order is irrelevant because
    # every cross-boundary FK is CASCADE or SET NULL (no RESTRICT/NO ACTION
    # points at anything we delete). Wrapped in a transaction: all or nothing.
    {
      echo "BEGIN;"
      echo "SET LOCAL session_replication_role = 'origin';"
      while IFS= read -r t; do
        [ -n "$t" ] && echo "DELETE FROM ${t};"
      done <<< "$uniq_tables"
      echo "COMMIT;"
    } | psql_do -q
    echo "${GREEN}    rows deleted.${RESET}"
  else
    echo "${DIM}No tables selected — skipping delete.${RESET}"
  fi

  [ "$PURGE_BLOBS" = "1" ] && run_purge_blobs
  [ "$RECOMPUTE" = "1" ] && run_recompute

  echo
  echo "${GREEN}${BOLD}Done.${RESET}"
}

confirm_and_run() {
  show_summary
  if [ "$(printf '%s' "${SELECTED[users]}")" = "1" ]; then
    echo "${RED}${BOLD}You are about to DELETE ALL USER ACCOUNTS.${RESET}"
  fi
  echo
  local word
  read -r -p "Type ${BOLD}RESET${RESET} to proceed (anything else cancels): " word || true
  if [ "$word" != "RESET" ]; then
    echo "${YELLOW}Cancelled. Nothing was changed.${RESET}"
    return 1
  fi
  execute
}

# ── Custom (per-category) picker ──────────────────────────────────────────
custom_picker() {
  reset_selection
  echo
  echo "${BOLD}Choose what to delete${RESET} ${DIM}(y/N for each; default keeps it)${RESET}"
  echo
  for c in "${ORDER[@]}"; do
    echo "${BOLD}${CAT_LABEL[$c]}${RESET}"
    echo "  ${DIM}${CAT_NOTE[$c]}${RESET}"
    if ask "  Delete this?" "n"; then SELECTED[$c]=1; fi
    echo
  done
  if [ "${SELECTED[files]}" = "1" ]; then
    if ask "Also purge the physical file blobs from disk?" "y"; then PURGE_BLOBS=1; fi
  fi
  echo
  if ask "Recompute storage counters afterwards (recommended)?" "y"; then
    RECOMPUTE=1
  else
    RECOMPUTE=0
  fi
}

# ── Snapshot of current state ─────────────────────────────────────────────
show_state() {
  echo "${DIM}Current database state:${RESET}"
  psql_do -qtA -F ' | ' <<'SQL' || true
SELECT 'users='||(SELECT count(*) FROM users),
       'files='||(SELECT count(*) FROM files),
       'folders='||(SELECT count(*) FROM folders),
       'teams='||(SELECT count(*) FROM teams),
       'storage_used(sum)='||pg_size_pretty((SELECT COALESCE(sum(storage_used),0) FROM users));
SQL
}

# ── Main menu ─────────────────────────────────────────────────────────────
main_menu() {
  clear 2>/dev/null || true
  echo "${BOLD}${CYAN}SFTP service — database reset (master)${RESET}"
  echo "${DIM}project: ${PROJECT_DIR}${RESET}"
  echo "${DIM}db:      ${PG_DB} @ service '${PG_SERVICE}'${RESET}"
  echo
  show_state
  echo
  echo "${BOLD}What would you like to do?${RESET}"
  echo "  ${BOLD}1${RESET})  Clear files & folders only        ${DIM}(keep users, teams, logs)  [recommended]${RESET}"
  echo "  ${BOLD}2${RESET})  Clear all content                 ${DIM}(files + shares + uploads + tags)${RESET}"
  echo "  ${BOLD}3${RESET})  Clear content + activity logs     ${DIM}(above + audit/activity/notifications)${RESET}"
  echo "  ${BOLD}4${RESET})  Full reset, keep users only       ${DIM}(everything except accounts & RBAC)${RESET}"
  echo "  ${BOLD}5${RESET})  Custom…                           ${DIM}(choose each category)${RESET}"
  echo "  ${BOLD}6${RESET})  ${GREEN}Only fix storage usage${RESET}            ${DIM}(recompute counters, delete nothing)${RESET}"
  echo "  ${BOLD}q${RESET})  Quit"
  echo
  read -r -p "Select [1-6/q]: " choice || true
  case "$choice" in
    1) preset_files_only;      RECOMPUTE=1; confirm_and_run || true ;;
    2) preset_all_content;     RECOMPUTE=1; confirm_and_run || true ;;
    3) preset_content_activity;RECOMPUTE=1; confirm_and_run || true ;;
    4) preset_keep_users_only; RECOMPUTE=1; confirm_and_run || true ;;
    5) custom_picker;                       confirm_and_run || true ;;
    6) reset_selection; RECOMPUTE=1; PURGE_BLOBS=0
       echo; echo "This will ${GREEN}recompute storage counters only${RESET} and delete nothing."
       if ask "Proceed?" "y"; then run_recompute; echo; echo "${GREEN}${BOLD}Done.${RESET}"; else echo "${YELLOW}Cancelled.${RESET}"; fi ;;
    q|Q) echo "Bye."; exit 0 ;;
    *) echo "${YELLOW}Unknown choice.${RESET}"; exit 1 ;;
  esac
}

main_menu
