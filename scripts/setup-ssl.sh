#!/usr/bin/env bash
#
# setup-ssl.sh — obtain a Let's Encrypt certificate for the dockerized nginx and
# set up automatic renewal.
#
#   sudo ./scripts/setup-ssl.sh files.yourcompany.com you@yourcompany.com
#
# What it does:
#   1. Installs certbot (if missing).
#   2. Temporarily frees port 80 (stops the nginx container), issues the cert
#      via the standalone challenge, then restarts nginx.
#   3. Copies fullchain.pem + privkey.pem into docker/nginx/certs/.
#   4. Installs a renewal hook + a daily systemd timer that renews, re-copies the
#      certs, and reloads the nginx container automatically before expiry.
#
set -euo pipefail

DOMAIN="${1:?Usage: setup-ssl.sh <domain> <email>}"
EMAIL="${2:?Usage: setup-ssl.sh <domain> <email>}"
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CERT_DIR="$PROJECT_DIR/docker/nginx/certs"
LIVE="/etc/letsencrypt/live/$DOMAIN"

if [ "$(id -u)" -ne 0 ]; then echo "Run with sudo." >&2; exit 1; fi

# 1. certbot
if ! command -v certbot >/dev/null 2>&1; then
  echo "==> Installing certbot"
  if command -v apt-get >/dev/null; then apt-get update -qq && apt-get install -y certbot
  elif command -v dnf >/dev/null; then dnf install -y certbot
  else echo "Install certbot manually and re-run." >&2; exit 1; fi
fi

DC="docker compose"
cd "$PROJECT_DIR"

# 2. issue the certificate (standalone needs port 80 free)
echo "==> Freeing port 80 and issuing certificate for $DOMAIN"
$DC stop nginx || true
certbot certonly --standalone --non-interactive --agree-tos \
  -m "$EMAIL" -d "$DOMAIN" --keep-until-expiring

# 3. copy certs into the nginx build context
mkdir -p "$CERT_DIR"
cp -L "$LIVE/fullchain.pem" "$CERT_DIR/fullchain.pem"
cp -L "$LIVE/privkey.pem"  "$CERT_DIR/privkey.pem"
chmod 644 "$CERT_DIR/fullchain.pem"; chmod 600 "$CERT_DIR/privkey.pem"

echo "==> Restarting nginx with TLS"
$DC up -d nginx

# 4. renewal hook: copy fresh certs + reload nginx after each renewal
HOOK=/etc/letsencrypt/renewal-hooks/deploy/sapphire-sftp.sh
mkdir -p "$(dirname "$HOOK")"
cat > "$HOOK" <<EOF
#!/usr/bin/env bash
cp -L "$LIVE/fullchain.pem" "$CERT_DIR/fullchain.pem"
cp -L "$LIVE/privkey.pem"  "$CERT_DIR/privkey.pem"
cd "$PROJECT_DIR" && docker compose exec -T nginx nginx -s reload 2>/dev/null || docker compose up -d nginx
EOF
chmod +x "$HOOK"

# daily renewal timer (certbot only renews when <30 days remain)
cat > /etc/systemd/system/sapphire-sftp-certrenew.service <<EOF
[Unit]
Description=Renew Sapphire SFTP TLS certificate
[Service]
Type=oneshot
ExecStart=/usr/bin/certbot renew --quiet
EOF
cat > /etc/systemd/system/sapphire-sftp-certrenew.timer <<EOF
[Unit]
Description=Daily Sapphire SFTP certificate renewal check
[Timer]
OnCalendar=daily
RandomizedDelaySec=1h
Persistent=true
[Install]
WantedBy=timers.target
EOF
systemctl daemon-reload
systemctl enable --now sapphire-sftp-certrenew.timer

cat <<EOF

✓ TLS certificate installed for https://$DOMAIN
✓ Auto-renewal enabled (daily check; renews within 30 days of expiry, then
  reloads nginx automatically).

Next steps (once only):
  1. In docker/nginx/nginx.conf, uncomment the 443 server block + HTTP->HTTPS redirect.
  2. In docker-compose.yml, publish "443:443" on the nginx service.
  3. Set PUBLIC_URL=https://$DOMAIN in .env
  4. docker compose up -d --build nginx

Check the timer:  systemctl list-timers sapphire-sftp-certrenew.timer
Dry-run renewal:  sudo certbot renew --dry-run
EOF
