#!/usr/bin/env bash
#
# deploy.sh — one-command interactive deploy for the Sapphire SFTP platform.
#
# Asks the basic company questions, writes brand.config.json + .env, then builds
# and starts the full stack (postgres + backend + frontend + nginx) with Docker
# Compose. Safe to re-run: it reuses existing answers as defaults and never
# deletes your data volumes.
#
#   ./deploy.sh                 # interactive
#   ./deploy.sh --yes           # non-interactive (uses existing/defaults)
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
hdr() { printf '\n%s%s%s\n' "$B$CYN" "$*" "$RST"; }
ok()   { printf '%s✓%s %s\n' "$GRN" "$RST" "$*"; }
warn() { printf '%s!%s %s\n' "$YLW" "$RST" "$*"; }
die()  { printf '%s✗ %s%s\n' "$RED" "$*" "$RST" >&2; exit 1; }

ASSUME_YES=0
[ "${1:-}" = "--yes" ] || [ "${1:-}" = "-y" ] && ASSUME_YES=1

# ── prerequisites (auto-install anything missing) ─────────────────────────────
OS="$(uname -s)"
SUDO=""; [ "$(id -u)" -ne 0 ] && command -v sudo >/dev/null 2>&1 && SUDO="sudo"

pkg_install() { # install packages with whatever package manager exists
  if   command -v apt-get >/dev/null 2>&1; then $SUDO apt-get update -qq && $SUDO apt-get install -y "$@"
  elif command -v dnf     >/dev/null 2>&1; then $SUDO dnf install -y "$@"
  elif command -v yum     >/dev/null 2>&1; then $SUDO yum install -y "$@"
  elif command -v brew    >/dev/null 2>&1; then brew install "$@"
  else return 1; fi
}
ensure() { # ensure <binary> [package]
  command -v "$1" >/dev/null 2>&1 && return 0
  hdr "Installing $1"
  pkg_install "${2:-$1}" >/dev/null 2>&1 || die "Could not auto-install '$1'. Install it manually and re-run."
  command -v "$1" >/dev/null 2>&1 || die "'$1' still not found after install."
}

ensure curl curl
ensure git git
ensure python3 python3
ensure openssl openssl

# Docker engine
if ! command -v docker >/dev/null 2>&1; then
  case "$OS" in
    Linux)
      hdr "Installing Docker Engine"
      curl -fsSL https://get.docker.com | $SUDO sh || die "Docker install failed. See https://docs.docker.com/engine/install/"
      $SUDO usermod -aG docker "${SUDO_USER:-$USER}" 2>/dev/null || true
      $SUDO systemctl enable --now docker 2>/dev/null || true ;;
    Darwin) die "Install Docker Desktop (https://www.docker.com/products/docker-desktop/), start it, then re-run." ;;
    *) die "Install Docker + Compose for your OS, then re-run." ;;
  esac
fi

# Start the daemon if it isn't running
if ! docker info >/dev/null 2>&1 && ! $SUDO docker info >/dev/null 2>&1; then
  hdr "Starting the Docker daemon"
  $SUDO systemctl start docker 2>/dev/null || $SUDO service docker start 2>/dev/null || true
  for _ in 1 2 3 4 5; do docker info >/dev/null 2>&1 || $SUDO docker info >/dev/null 2>&1 && break; sleep 2; done
fi

# Pick a working docker invocation (handle the "group not applied yet" case)
if   docker info >/dev/null 2>&1; then DOCKER="docker"
elif $SUDO docker info >/dev/null 2>&1; then DOCKER="$SUDO docker"; warn "Using sudo for docker (log out/in after being added to the 'docker' group to avoid this)."
else die "The Docker daemon is not reachable. Start Docker and re-run."; fi

# Docker Compose (install the plugin if absent)
if   $DOCKER compose version >/dev/null 2>&1; then DC="$DOCKER compose"
elif command -v docker-compose >/dev/null 2>&1; then DC="${SUDO:+$SUDO }docker-compose"
else
  hdr "Installing the Docker Compose plugin"
  pkg_install docker-compose-plugin >/dev/null 2>&1 || true
  $DOCKER compose version >/dev/null 2>&1 && DC="$DOCKER compose" || die "Docker Compose is not available."
fi

gen() { # generate a random hex secret
  if command -v openssl >/dev/null 2>&1; then openssl rand -hex "${1:-32}"
  else head -c "${1:-32}" /dev/urandom | od -An -tx1 | tr -d ' \n'; fi
}

# read a value from the existing brand.config.json (dotted path) or ""
brand_get() {
  [ -f brand.config.json ] || { echo ""; return; }
  python3 - "$1" <<'PY' 2>/dev/null || echo ""
import json,sys
try:
    d=json.load(open("brand.config.json"))
    for k in sys.argv[1].split("."): d=d[k]
    print(d if not isinstance(d,list) else ",".join(d))
except Exception:
    print("")
PY
}

# read an existing value from .env or ""
env_get() { [ -f .env ] && sed -n "s/^$1=//p" .env | head -1 || true; }

ask() { # ask "Prompt" "default" -> echoes answer
  local prompt="$1" def="${2:-}" ans
  if [ "$ASSUME_YES" = 1 ]; then echo "$def"; return; fi
  if [ -n "$def" ]; then printf '%s %s[%s]%s: ' "$prompt" "$DIM" "$def" "$RST" >/dev/tty
  else printf '%s: ' "$prompt" >/dev/tty; fi
  read -r ans </dev/tty || true
  echo "${ans:-$def}"
}
ask_secret() { # hidden input; empty keeps default
  local prompt="$1" def="${2:-}" ans
  if [ "$ASSUME_YES" = 1 ]; then echo "$def"; return; fi
  printf '%s %s(hidden, Enter to keep)%s: ' "$prompt" "$DIM" "$RST" >/dev/tty
  read -rs ans </dev/tty || true; printf '\n' >/dev/tty
  echo "${ans:-$def}"
}
yesno() { # yesno "Prompt" "y|n" -> 0 if yes
  local def="${2:-n}" ans
  ans=$(ask "$1 (y/n)" "$def")
  case "$ans" in y|Y|yes|YES) return 0;; *) return 1;; esac
}

say "${B}Sapphire SFTP — deployment${RST}"
say "${DIM}Answer the questions (Enter accepts the [default]).${RST}"

# ── company basics ───────────────────────────────────────────────────────────
hdr "Company"
CO_NAME=$(ask     "Company name"            "$(brand_get company.name || echo 'Sapphire Broking')")
CO_SHORT=$(ask    "Short name"              "$(brand_get company.shortName || echo 'Sapphire')")
CO_PRODUCT=$(ask  "Product name"            "$(brand_get company.product || echo "$CO_SHORT SFTP")")
CO_PSHORT=$(ask   "Product short label"     "$(brand_get company.productShort || echo 'SFTP')")
CO_TAGLINE=$(ask  "Tagline"                 "$(brand_get company.tagline || echo 'Enterprise self-hosted file transfer')")
CO_URL=$(ask      "Company website URL"     "$(brand_get company.url || echo 'https://example.com')")
CO_COLOR=$(ask    "Brand colour (hex)"      "$(brand_get colors.primary || echo '#064D51')")

hdr "Organisation & access"
ORG_DOMAINS=$(ask "Org email domain(s), comma-separated" "$(brand_get org.domains || echo 'example.com')")
SUPPORT=$(ask     "Support email"           "$(brand_get org.supportEmail || echo "support@${ORG_DOMAINS%%,*}")")
MAIL_FROM=$(ask   "Outgoing mail 'From'"    "$(brand_get mail.from || echo "$CO_PRODUCT <no-reply@${ORG_DOMAINS%%,*}>")")
PUBLIC_URL=$(ask  "Public URL users visit"  "$(env_get PUBLIC_URL || echo 'http://localhost')")
# The backend requires a full URL (scheme + host). If the user typed a bare
# host/IP (e.g. "sftp.corp.com"), prepend https:// so config validation passes.
case "$PUBLIC_URL" in
  http://*|https://*) : ;;
  "") PUBLIC_URL="http://localhost" ;;
  *) PUBLIC_URL="https://$PUBLIC_URL"; warn "No scheme in Public URL — using $PUBLIC_URL" ;;
esac

hdr "First administrator"
ADMIN_EMAIL=$(ask "Admin email"    "$(env_get BOOTSTRAP_ADMIN_EMAIL || echo "admin@${ORG_DOMAINS%%,*}")")
ADMIN_USER=$(ask  "Admin username" "$(env_get BOOTSTRAP_ADMIN_USERNAME || echo 'admin')")
ADMIN_PASS=$(ask_secret "Admin password" "$(env_get BOOTSTRAP_ADMIN_PASSWORD || echo '')")
if [ -z "$ADMIN_PASS" ]; then ADMIN_PASS="$(gen 12)"; ADMIN_GEN=1; else ADMIN_GEN=0; fi

# ── optional features ────────────────────────────────────────────────────────
ENC_KEY="$(env_get STORAGE_ENCRYPTION_KEY || true)"
hdr "Security & features (optional)"
if [ -z "$ENC_KEY" ] && yesno "Encrypt all files at rest (AES-256)?" "n"; then
  ENC_KEY="$(gen 32)"; ok "Generated a 32-byte encryption key (stored in .env — back it up!)"
fi

SMTP_ENABLED=false; SMTP_HOST=""; SMTP_PORT=587; SMTP_USER=""; SMTP_PASS=""
if yesno "Enable email sending (SMTP) for share notifications?" "n"; then
  SMTP_ENABLED=true
  SMTP_HOST=$(ask "  SMTP host" "$(brand_get smtp.host)")
  SMTP_PORT=$(ask "  SMTP port" "587")
  SMTP_USER=$(ask "  SMTP username" "$(brand_get smtp.username)")
  SMTP_PASS=$(ask_secret "  SMTP password" "")
fi

SSO_ENABLED=false; SSO_TENANT="organizations"; SSO_CLIENT=""; SSO_SECRET=""
if yesno "Enable Microsoft (Entra ID) single sign-on?" "n"; then
  SSO_ENABLED=true
  SSO_TENANT=$(ask "  Azure tenant ID (GUID for single-tenant)" "$(brand_get sso.microsoft.tenantId || echo 'organizations')")
  SSO_CLIENT=$(ask "  Application (client) ID" "$(brand_get sso.microsoft.clientId)")
  SSO_SECRET=$(ask_secret "  Client secret" "")
fi

AI_ENABLED=false; AI_OLLAMA="http://ollama:11434"
if yesno "Enable on-prem AI (semantic search + ask-your-files via Ollama)?" "n"; then
  AI_ENABLED=true
  AI_OLLAMA=$(ask "  Ollama server URL" "http://ollama:11434")
fi

ED_ENABLED=false; ED_DOCURL=""
ED_SECRET="$(brand_get editor.jwtSecret)"; [ -n "$ED_SECRET" ] || ED_SECRET="$(gen 32)"
if yesno "Enable live Office editing (OnlyOffice Document Server)?" "n"; then
  ED_ENABLED=true
  ED_DOCURL=$(ask "  OnlyOffice Document Server public URL" "$(brand_get editor.docServerUrl)")
fi

# ── generate brand.config.json ───────────────────────────────────────────────
hdr "Writing configuration"
export CO_NAME CO_SHORT CO_PRODUCT CO_PSHORT CO_TAGLINE CO_URL CO_COLOR \
       ORG_DOMAINS SUPPORT MAIL_FROM PUBLIC_URL \
       SMTP_ENABLED SMTP_HOST SMTP_PORT SMTP_USER SMTP_PASS \
       SSO_ENABLED SSO_TENANT SSO_CLIENT SSO_SECRET AI_ENABLED AI_OLLAMA \
       ED_ENABLED ED_DOCURL ED_SECRET

python3 - <<'PY'
import json, os
def b(v): return str(v).lower() == "true"
domains = [d.strip() for d in os.environ["ORG_DOMAINS"].split(",") if d.strip()]
pub = os.environ["PUBLIC_URL"].rstrip("/")
cfg = {
  "company": {
    "name": os.environ["CO_NAME"], "shortName": os.environ["CO_SHORT"],
    "product": os.environ["CO_PRODUCT"], "productShort": os.environ["CO_PSHORT"],
    "tagline": os.environ["CO_TAGLINE"],
    "description": f'{os.environ["CO_PRODUCT"]} — {os.environ["CO_TAGLINE"]}.',
    "url": os.environ["CO_URL"], "copyright": os.environ["CO_NAME"],
  },
  "logo": {"full": "/logo.svg", "light": "/logo-white.svg", "dark": "/logo-black.svg", "favicon": "/logo.svg"},
  "colors": {"primary": os.environ["CO_COLOR"], "primaryForeground": "#FFFFFF",
             "primaryDark": os.environ["CO_COLOR"], "primaryForegroundDark": "#FFFFFF"},
  "org": {"domains": domains, "supportEmail": os.environ["SUPPORT"]},
  "mail": {"from": os.environ["MAIL_FROM"]},
  "smtp": {"enabled": b(os.environ["SMTP_ENABLED"]), "host": os.environ["SMTP_HOST"],
           "port": int(os.environ["SMTP_PORT"] or 587), "username": os.environ["SMTP_USER"],
           "password": os.environ["SMTP_PASS"], "startTls": True},
  "sso": {"microsoft": {
     "enabled": b(os.environ["SSO_ENABLED"]), "tenantId": os.environ["SSO_TENANT"],
     "clientId": os.environ["SSO_CLIENT"], "clientSecret": os.environ["SSO_SECRET"],
     "redirectUrl": f"{pub}/api/v1/auth/sso/microsoft/callback",
     "successUrl": f"{pub}/auth/sso/callback", "allowedDomains": [], "defaultRole": "employee"}},
  "ai": {"enabled": b(os.environ["AI_ENABLED"]), "ollamaUrl": os.environ["AI_OLLAMA"],
         "embedModel": "nomic-embed-text", "chatModel": "llama3.1"},
  "editor": {"enabled": b(os.environ["ED_ENABLED"]), "docServerUrl": os.environ["ED_DOCURL"],
             "jwtSecret": os.environ["ED_SECRET"], "internalBaseUrl": "http://nginx"},
}
json.dump(cfg, open("brand.config.json", "w"), indent=2)
open("brand.config.json", "a").write("\n")
print("  brand.config.json written")
PY
ok "brand.config.json"

# ── generate/merge .env ──────────────────────────────────────────────────────
JWT_SECRET="$(env_get JWT_SECRET || true)"; [ -n "$JWT_SECRET" ] || JWT_SECRET="$(gen 32)"
PG_PASS="$(env_get POSTGRES_PASSWORD || true)"; [ -n "$PG_PASS" ] || PG_PASS="$(gen 16)"
# Always provision a backup-archive key so the super-admin backup/restore works
# out of the box (independent of at-rest storage encryption).
BACKUP_KEY="$(env_get BACKUP_ENCRYPTION_KEY || true)"; [ -n "$BACKUP_KEY" ] || BACKUP_KEY="$(gen 32)"

umask 077
{
  echo "# Generated by deploy.sh — contains secrets. Do not commit."
  echo "PUBLIC_URL=$PUBLIC_URL"
  echo "POSTGRES_USER=sftp"
  echo "POSTGRES_PASSWORD=$PG_PASS"
  echo "POSTGRES_DB=sftp"
  echo "JWT_SECRET=$JWT_SECRET"
  echo "BOOTSTRAP_ADMIN_EMAIL=$ADMIN_EMAIL"
  echo "BOOTSTRAP_ADMIN_USERNAME=$ADMIN_USER"
  echo "BOOTSTRAP_ADMIN_PASSWORD=$ADMIN_PASS"
  echo "ORG_DOMAINS=$ORG_DOMAINS"
  echo "AI_ENABLED=$AI_ENABLED"
  echo "AI_OLLAMA_URL=$AI_OLLAMA"
  echo "EDITOR_JWT_SECRET=$ED_SECRET"
  echo "BACKUP_ENCRYPTION_KEY=$BACKUP_KEY"
  [ -n "$ENC_KEY" ] && echo "STORAGE_ENCRYPTION_KEY=$ENC_KEY"
} > .env
ok ".env (secrets, permissions 600)"

# ── deploy ───────────────────────────────────────────────────────────────────
hdr "Building and starting the stack"
say "${DIM}(first build can take a few minutes)${RST}"
$DC up -d --build

hdr "Waiting for the service to become healthy"
# Poll the LOCAL nginx, not PUBLIC_URL — the public hostname may not resolve from
# the server itself yet (DNS/TLS get set up afterwards).
for i in $(seq 1 60); do
  if curl -fsS "http://localhost/api/v1/health-check" >/dev/null 2>&1; then ok "Backend is healthy"; break; fi
  sleep 3
  [ "$i" = 60 ] && warn "Health check timed out — inspect logs with: $DC logs -f backend"
done

# ── HTTPS (Let's Encrypt) ─────────────────────────────────────────────────────
# If the public URL is an https:// real domain, offer to obtain a certificate and
# set up auto-renewal now (default yes). Skipped for localhost / behind Cloudflare
# proxy (where Cloudflare terminates TLS and the origin stays on http).
case "$PUBLIC_URL" in
  https://localhost*|https://127.*|https://0.0.0.0*) ;;
  https://*)
    SSL_DOMAIN="${PUBLIC_URL#https://}"; SSL_DOMAIN="${SSL_DOMAIN%%/*}"
    hdr "HTTPS for $SSL_DOMAIN"
    say "${DIM}Requires: DNS A record -> this server, ports 80+443 reachable, and NOT proxied${RST}"
    say "${DIM}by Cloudflare (if you use Cloudflare's orange-cloud proxy, choose 'No' and set${RST}"
    say "${DIM}SSL mode to Full there instead).${RST}"
    if yesno "Set up a Let's Encrypt certificate + auto-renewal now?" "y"; then
      if sudo ./scripts/setup-ssl.sh "$SSL_DOMAIN" "$ADMIN_EMAIL"; then
        ok "TLS configured for https://$SSL_DOMAIN (auto-renewing)"
      else
        warn "SSL setup didn't complete — the app is still up on http."
        warn "Re-run later: sudo ./scripts/setup-ssl.sh $SSL_DOMAIN $ADMIN_EMAIL"
      fi
    fi
    ;;
esac

# ── summary ──────────────────────────────────────────────────────────────────
hdr "Done"
say "  URL       ${B}$PUBLIC_URL${RST}"
say "  Admin     ${B}$ADMIN_USER${RST}  (${ADMIN_EMAIL})"
if [ "$ADMIN_GEN" = 1 ]; then
  say "  Password  ${B}$ADMIN_PASS${RST}  ${YLW}(generated — you'll change it on first login)${RST}"
else
  say "  Password  ${DIM}(as you entered)${RST}"
fi
[ -n "$ENC_KEY" ] && say "  ${YLW}Encryption is ON. Back up STORAGE_ENCRYPTION_KEY in .env — losing it makes files unreadable.${RST}"
[ "$AI_ENABLED" = "true" ] && say "  ${DIM}AI is enabled — ensure an Ollama server is reachable at $AI_OLLAMA with the models pulled.${RST}"
say ""
say "  Manage:   ${DIM}$DC ps · $DC logs -f · $DC down${RST}"
say "  Re-run this script any time to change branding or settings."
