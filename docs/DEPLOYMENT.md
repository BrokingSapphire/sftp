# Deployment Guide

Deploy the full stack (PostgreSQL + Go backend + Next.js frontend + Nginx) on a
single on-premise Linux server with Docker.

## Prerequisites

- Linux server (Ubuntu 22.04+ / Debian 12+ / RHEL 9+)
- Docker Engine 24+ and the Docker Compose plugin
- A mounted volume/drive for file storage (optional; Docker volumes work too)

## 1. Get the code

```bash
git clone https://github.com/BrokingSapphire/sftp.git
cd sftp
```

## 2. Configure

```bash
cp .env.example .env
# Generate a strong JWT secret:
echo "JWT_SECRET=$(openssl rand -base64 48)" >> .env
# Edit .env: set POSTGRES_PASSWORD, BOOTSTRAP_ADMIN_PASSWORD, PUBLIC_URL
nano .env
```

| Variable | Description |
| -------- | ----------- |
| `PUBLIC_URL` | External URL users reach (`https://files.corp.local`) |
| `POSTGRES_PASSWORD` | Database password |
| `JWT_SECRET` | Token signing secret, **min 32 chars** |
| `BOOTSTRAP_ADMIN_PASSWORD` | Password for the auto-created `admin` super-admin |

## 3. Launch

```bash
docker compose up -d --build
```

This starts four services. The backend runs database migrations automatically on
first boot and creates the super-admin (`admin` / your `BOOTSTRAP_ADMIN_PASSWORD`).

- Web UI:  `http://<server>/`
- API:     `http://<server>/api/v1`
- Health:  `http://<server>/api/v1/health-check`

Log in as `admin`, then create users and assign roles under **Administration → Users**.

## 4. HTTPS (recommended)

1. Obtain certificates (internal CA or Let's Encrypt) and place them in
   `docker/nginx/certs/` as `fullchain.pem` and `privkey.pem`.
2. Uncomment the `443` port, the TLS lines, and the HTTP→HTTPS redirect in
   `docker/nginx/nginx.conf` and the `443` mapping in `docker-compose.yml`.
3. Set `PUBLIC_URL=https://...` in `.env`.
4. `docker compose up -d`.

## 5. Persistent storage on a mounted drive

To store files on a mounted NAS/drive instead of a Docker volume, edit
`docker-compose.yml` and replace the `sftp-files` volume with a bind mount:

```yaml
    volumes:
      - /mnt/nas/sftp:/app/storage/files
```

## Native SFTP endpoint

The backend also exposes SFTP-over-SSH (default port `2222`) for programmatic
transfers, authenticated with the same accounts and API keys. Publish the port
in `docker-compose.yml` (`"2222:2222"` on the backend service) to enable it.

## Operations

```bash
docker compose ps                 # status
docker compose logs -f backend    # tail logs
docker compose down               # stop (keeps data)
docker compose pull && docker compose up -d --build   # upgrade
```

See [BACKUP.md](./BACKUP.md) for backups and [TROUBLESHOOTING.md](./TROUBLESHOOTING.md).
