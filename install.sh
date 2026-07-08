#!/usr/bin/env bash
#
# install.sh — one-command bootstrap installer for Sapphire SFTP.
#
# Installs prerequisites (Docker, git, python3) if missing, clones the repo, then
# runs the guided deploy. Works on Linux, macOS, and Windows (WSL).
#
#   curl -fsSL https://raw.githubusercontent.com/BrokingSapphire/sftp/main/install.sh | bash
#
# Or download and run locally:
#
#   bash install.sh              # interactive
#   bash install.sh --yes        # non-interactive (accept defaults)
#
# Environment overrides:
#   SFTP_DIR=/opt/sftp           # where to clone (default: ./sftp or $HOME/sftp)
#   SFTP_REF=main                # git branch/tag to check out
#
set -euo pipefail

REPO="https://github.com/BrokingSapphire/sftp.git"
REF="${SFTP_REF:-main}"
ARGS="${*:-}"

# ── pretty output ────────────────────────────────────────────────────────────
if [ -t 1 ]; then
  B=$'\033[1m'; GRN=$'\033[32m'; YLW=$'\033[33m'; RED=$'\033[31m'; CYN=$'\033[36m'; RST=$'\033[0m'
else B=""; GRN=""; YLW=""; RED=""; CYN=""; RST=""; fi
head() { printf '\n%s%s%s\n' "$B$CYN" "$*" "$RST"; }
ok()   { printf '%s✓%s %s\n' "$GRN" "$RST" "$*"; }
warn() { printf '%s!%s %s\n' "$YLW" "$RST" "$*"; }
die()  { printf '%s✗ %s%s\n' "$RED" "$*" "$RST" >&2; exit 1; }
has()  { command -v "$1" >/dev/null 2>&1; }

OS="$(uname -s)"
head "Sapphire SFTP installer"
printf 'Detected OS: %s\n' "$OS"

# ── 1. package-manager helpers ───────────────────────────────────────────────
apt_install()  { sudo apt-get update -qq && sudo apt-get install -y "$@"; }
dnf_install()  { sudo dnf install -y "$@"; }
brew_install() { brew install "$@"; }

install_pkg() { # install_pkg <binary> <apt-name> <dnf-name> <brew-name>
  bin="$1"; apt="$2"; dnf="$3"; brew="$4"
  has "$bin" && return 0
  head "Installing $bin"
  if has apt-get; then apt_install "$apt"
  elif has dnf; then dnf_install "$dnf"
  elif has brew; then brew_install "$brew"
  else die "Could not install $bin automatically. Install it manually and re-run."; fi
}

# ── 2. git + python3 ─────────────────────────────────────────────────────────
install_pkg git git git git
install_pkg python3 python3 python3 python

# ── 3. Docker ────────────────────────────────────────────────────────────────
ensure_docker() {
  if has docker && docker info >/dev/null 2>&1; then ok "Docker is installed and running"; return; fi
  case "$OS" in
    Linux)
      if ! has docker; then
        head "Installing Docker Engine (official convenience script)"
        curl -fsSL https://get.docker.com | sudo sh
        sudo usermod -aG docker "$USER" || true
        sudo systemctl enable --now docker || true
        warn "You were added to the 'docker' group. If the next step fails with a"
        warn "permission error, log out and back in (or run: newgrp docker) and re-run."
      fi
      # Always ensure the daemon starts on boot, even if Docker was already
      # installed (the block above only runs on a fresh install).
      sudo systemctl enable docker >/dev/null 2>&1 || true
      docker info >/dev/null 2>&1 || sudo systemctl start docker || true
      ;;
    Darwin)
      die "Docker Desktop is required on macOS. Install it from
     https://www.docker.com/products/docker-desktop/  (or: brew install --cask docker),
     start it, then re-run this installer."
      ;;
    *)
      die "Docker not found. On Windows, install Docker Desktop with the WSL 2 backend
     and run this installer inside WSL. See docs/DEPLOYMENT.md."
      ;;
  esac
}
ensure_docker

# Compose plugin check
docker compose version >/dev/null 2>&1 || die "The 'docker compose' plugin is missing. See docs/DEPLOYMENT.md."
ok "Docker Compose available"

# ── 4. get the code ──────────────────────────────────────────────────────────
if [ -f "./deploy.sh" ] && [ -f "./docker-compose.yml" ]; then
  DIR="$(pwd)"
  ok "Running inside an existing checkout: $DIR"
else
  DIR="${SFTP_DIR:-$PWD/sftp}"
  if [ -d "$DIR/.git" ]; then
    head "Updating existing checkout at $DIR"
    git -C "$DIR" fetch --depth 1 origin "$REF" && git -C "$DIR" checkout -f "$REF" && git -C "$DIR" pull --ff-only || true
  else
    head "Cloning $REPO -> $DIR"
    git clone --depth 1 --branch "$REF" "$REPO" "$DIR"
  fi
fi

# ── 5. autostart on boot (Linux/systemd) ─────────────────────────────────────
# Installs a systemd unit that brings the whole compose stack up on every boot.
# The compose files already set `restart: unless-stopped`, but that only revives
# containers that were running at shutdown; this unit guarantees the stack comes
# up even after a cold boot or a prior `docker compose down`.
install_autostart() {
  case "$OS" in Linux) ;; *) return 0 ;; esac
  has systemctl || { warn "No systemd; skipping boot autostart. Start manually with: docker compose up -d"; return 0; }

  DC_BIN="$(command -v docker) compose"
  UNIT=/etc/systemd/system/sftp.service
  head "Installing boot autostart service ($UNIT)"
  sudo tee "$UNIT" >/dev/null <<EOF
[Unit]
Description=Sapphire SFTP (docker compose stack)
Requires=docker.service
After=docker.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=$DIR
ExecStart=$DC_BIN up -d
ExecStop=$DC_BIN down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF
  sudo systemctl daemon-reload || true
  sudo systemctl enable sftp.service >/dev/null 2>&1 \
    && ok "Autostart enabled — stack will come up on every boot (systemctl status sftp)" \
    || warn "Could not enable sftp.service; enable it manually: sudo systemctl enable sftp"
}
install_autostart

# ── 6. deploy ────────────────────────────────────────────────────────────────
cd "$DIR"
chmod +x deploy.sh
head "Starting guided deploy"
# shellcheck disable=SC2086
exec ./deploy.sh $ARGS
