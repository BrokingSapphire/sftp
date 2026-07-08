#!/usr/bin/env bash
#
# redeploy.sh — pull the latest code and restart the Sapphire SFTP stack.
#
# Unlike deploy.sh, this NEVER re-asks the setup questions and never touches your
# secrets or data: it reuses the existing brand.config.json + .env, rebuilds the
# images, restarts the containers, and (re)installs the boot autostart unit. Safe
# to run any time you've pulled new code or changed branding.
#
#   ./redeploy.sh                # git pull + rebuild + restart + refresh autostart
#   ./redeploy.sh --no-pull      # skip git pull (use the code already on disk)
#   ./redeploy.sh --no-build     # restart without rebuilding images
#
set -euo pipefail

cd "$(dirname "$0")"

# ── pretty output ────────────────────────────────────────────────────────────
if [ -t 1 ]; then
  B=$'\033[1m'; DIM=$'\033[2m'; GRN=$'\033[32m'; YLW=$'\033[33m'; RED=$'\033[31m'; CYN=$'\033[36m'; RST=$'\033[0m'
else
  B=""; DIM=""; GRN=""; YLW=""; RED=""; CYN=""; RST=""
fi
say()  { printf '%s\n' "$*"; }
hdr()  { printf '\n%s%s%s\n' "$B$CYN" "$*" "$RST"; }
ok()   { printf '%s✓%s %s\n' "$GRN" "$RST" "$*"; }
warn() { printf '%s!%s %s\n' "$YLW" "$RST" "$*"; }
die()  { printf '%s✗ %s%s\n' "$RED" "$*" "$RST" >&2; exit 1; }

DO_PULL=1; DO_BUILD=1
for a in "$@"; do
  case "$a" in
    --no-pull)  DO_PULL=0 ;;
    --no-build) DO_BUILD=0 ;;
    -h|--help)  sed -n '2,13p' "$0"; exit 0 ;;
    *) warn "Unknown option: $a" ;;
  esac
done

OS="$(uname -s)"
SUDO=""; [ "$(id -u)" -ne 0 ] && command -v sudo >/dev/null 2>&1 && SUDO="sudo"

# ── sanity: this must be an existing deployment ──────────────────────────────
[ -f docker-compose.yml ] || die "docker-compose.yml not found. Run this from the checkout root."
[ -f .env ] || die "No .env found. This looks like a fresh install — run ./deploy.sh first."

# ── pick a working docker + compose invocation ───────────────────────────────
[ "$OS" = "Linux" ] && $SUDO systemctl enable docker >/dev/null 2>&1 || true
if ! docker info >/dev/null 2>&1 && ! $SUDO docker info >/dev/null 2>&1; then
  hdr "Starting the Docker daemon"
  $SUDO systemctl start docker 2>/dev/null || $SUDO service docker start 2>/dev/null || true
  for _ in 1 2 3 4 5; do docker info >/dev/null 2>&1 || $SUDO docker info >/dev/null 2>&1 && break; sleep 2; done
fi
if   docker info >/dev/null 2>&1; then DOCKER="docker"
elif $SUDO docker info >/dev/null 2>&1; then DOCKER="$SUDO docker"
else die "The Docker daemon is not reachable. Start Docker and re-run."; fi
if   $DOCKER compose version >/dev/null 2>&1; then DC="$DOCKER compose"
elif command -v docker-compose >/dev/null 2>&1; then DC="${SUDO:+$SUDO }docker-compose"
else die "Docker Compose is not available."; fi

# ── which optional profiles are enabled (from brand.config.json) ─────────────
brand_get() {
  [ -f brand.config.json ] || { echo ""; return; }
  python3 - "$1" <<'PY' 2>/dev/null || echo ""
import json,sys
try:
    d=json.load(open("brand.config.json"))
    for k in sys.argv[1].split("."): d=d[k]
    print(str(d).lower() if isinstance(d,bool) else d)
except Exception:
    print("")
PY
}
PROFILES=""
[ "$(brand_get ai.enabled)" = "true" ]     && PROFILES="$PROFILES --profile ai"
[ "$(brand_get editor.enabled)" = "true" ] && PROFILES="$PROFILES --profile office"

# ── pull latest code ─────────────────────────────────────────────────────────
if [ "$DO_PULL" = 1 ]; then
  if [ -d .git ] && command -v git >/dev/null 2>&1; then
    hdr "Pulling latest code"
    git pull --ff-only || warn "git pull failed (local changes or diverged branch) — continuing with the code on disk."
  else
    warn "Not a git checkout — skipping pull."
  fi
fi

# ── rebuild + restart ────────────────────────────────────────────────────────
hdr "Restarting the stack"
say "${DIM}(reuses existing .env and brand.config.json; data volumes are preserved)${RST}"
# shellcheck disable=SC2086
if [ "$DO_BUILD" = 1 ]; then
  $DC $PROFILES up -d --build
else
  $DC $PROFILES up -d
fi

# ── refresh the boot autostart unit (Linux/systemd) ──────────────────────────
if [ "$OS" = "Linux" ] && command -v systemctl >/dev/null 2>&1; then
  hdr "Refreshing boot autostart"
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    DC_UNIT="$(command -v docker) compose"
  elif command -v docker-compose >/dev/null 2>&1; then
    DC_UNIT="$(command -v docker-compose)"
  else
    DC_UNIT="$DC"
  fi
  UNIT=/etc/systemd/system/sftp.service
  # shellcheck disable=SC2086
  $SUDO tee "$UNIT" >/dev/null <<EOF
[Unit]
Description=Sapphire SFTP (docker compose stack)
Requires=docker.service
After=docker.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=$(pwd)
ExecStart=$DC_UNIT $PROFILES up -d
ExecStop=$DC_UNIT down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF
  $SUDO systemctl daemon-reload 2>/dev/null || true
  if $SUDO systemctl enable sftp.service >/dev/null 2>&1; then
    ok "Autostart enabled — the stack will come up on every boot ($SUDO systemctl status sftp)"
  else
    warn "Could not enable sftp.service. Enable it manually: sudo systemctl enable sftp"
  fi
fi

# ── health check ─────────────────────────────────────────────────────────────
hdr "Waiting for the service to become healthy"
for i in $(seq 1 60); do
  if curl -fsS "http://localhost/api/v1/health-check" >/dev/null 2>&1; then ok "Backend is healthy"; break; fi
  sleep 3
  [ "$i" = 60 ] && warn "Health check timed out — inspect logs with: $DC logs -f backend"
done

hdr "Done"
say "  Manage:   ${DIM}$DC ps · $DC logs -f · $DC down${RST}"
