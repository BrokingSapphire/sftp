# Deployment Guide

A complete, step-by-step guide to deploying **Sapphire SFTP** (PostgreSQL + Go
backend + Next.js frontend + Nginx) on **Linux**, **macOS**, and **Windows**.

The whole platform runs in Docker containers, so the process is nearly identical
on every operating system — the only real difference is how you install Docker.
Follow the section for your OS to install the prerequisites, then continue with
the shared **[Deploy](#3-deploy-the-stack)** steps.

> **Which OS should I use in production?** Use **Linux** for any real deployment.
> macOS and Windows are fully supported for evaluation, development, and small
> internal setups, but Linux is the tested, resource-efficient choice for a
> server that many people rely on.

## Table of Contents

- [0. Overview and requirements](#0-overview-and-requirements)
- [1. Install prerequisites](#1-install-prerequisites)
  - [Linux](#linux)
  - [macOS](#macos)
  - [Windows](#windows)
- [2. Get the code](#2-get-the-code)
- [3. Deploy the stack](#3-deploy-the-stack)
  - [Option A — Guided deploy (recommended)](#option-a--guided-deploy-recommended)
  - [Option B — Manual configuration](#option-b--manual-configuration)
- [4. Verify the deployment](#4-verify-the-deployment)
- [5. First login](#5-first-login)
- [6. Optional features (AI, Office, monitoring)](#6-optional-features-ai-office-monitoring)
- [7. Enable HTTPS (production)](#7-enable-https-production)
- [8. Store files on a mounted drive / NAS](#8-store-files-on-a-mounted-drive--nas)
- [9. Native SFTP endpoint](#9-native-sftp-endpoint)
- [10. Day-2 operations](#10-day-2-operations)
- [11. Updating to a new version](#11-updating-to-a-new-version)
- [12. Uninstall](#12-uninstall)
- [13. Troubleshooting by OS](#13-troubleshooting-by-os)

---

## 0. Overview and requirements

**What gets deployed** (one `docker compose` command starts all of it):

| Container | Purpose | Default port |
| --- | --- | --- |
| `nginx` | Reverse proxy / entry point | 80 (443 for TLS) |
| `frontend` | Next.js web app | internal |
| `backend` | Go API + SFTP server | internal (SFTP 2222 if published) |
| `postgres` | Database | internal |

**Minimum hardware**

| Scenario | vCPU | RAM | Disk |
| --- | --- | --- | --- |
| Evaluation / small team | 2 | 4 GB | 20 GB + your files |
| Production (dozens of users) | 4 | 8 GB | 100 GB + your files |
| With AI profile (Ollama) | +2 | +4–8 GB | +5–10 GB (models) |
| With Office profile (OnlyOffice) | +1 | +2 GB | +3 GB |

**Software:** Docker Engine 24+ with the Docker Compose plugin, `git`, and
`python3` (only needed for the guided `deploy.sh`). Installation of each is
covered per-OS below.

---

## 1. Install prerequisites

### Linux

These commands target **Ubuntu 22.04+/Debian 12+**. For RHEL/Fedora, substitute
`dnf` and the CentOS Docker repo.

**Step 1 — Update the system and install git + python:**

```bash
sudo apt-get update
sudo apt-get install -y git python3 ca-certificates curl
```

**Step 2 — Install Docker Engine + Compose plugin (official repository):**

```bash
# Add Docker's GPG key
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
  sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# Add the repository
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

**Step 3 — Run Docker without `sudo` (log out and back in afterwards):**

```bash
sudo usermod -aG docker "$USER"
newgrp docker
```

**Step 4 — Enable Docker at boot and verify:**

```bash
sudo systemctl enable --now docker
docker --version
docker compose version
```

You should see version strings for both. Continue to
[Get the code](#2-get-the-code).

### macOS

**Step 1 — Install Docker Desktop.**

- **Apple Silicon (M1/M2/M3) or Intel:** download **Docker Desktop for Mac** from
  <https://www.docker.com/products/docker-desktop/> and install the `.dmg`, or via
  Homebrew:
  ```bash
  brew install --cask docker
  ```
- Launch **Docker Desktop** from Applications and wait until the whale icon in the
  menu bar shows "Docker Desktop is running".

**Step 2 — Give Docker enough resources.** Docker Desktop → **Settings →
Resources** → set at least **4 GB RAM** (8 GB+ if you'll enable AI), and enough
disk. Click **Apply & Restart**.

**Step 3 — Install git and python (Homebrew):**

```bash
brew install git python
```

**Step 4 — Verify:**

```bash
docker --version
docker compose version
```

> **Note on external drives (macOS):** Docker Desktop cannot bind-mount
> individual files from some external/removable volumes ("operation not
> permitted"). Keep the project on your internal disk. Store *user files* on an
> external drive by bind-mounting the **directory** (see
> [section 8](#8-store-files-on-a-mounted-drive--nas)), not single files.

### Windows

**Step 1 — Enable WSL 2** (Windows Subsystem for Linux — Docker Desktop uses it).
Open **PowerShell as Administrator** and run:

```powershell
wsl --install
```

Reboot when prompted. This installs WSL 2 and a default Ubuntu distribution.

**Step 2 — Install Docker Desktop for Windows** from
<https://www.docker.com/products/docker-desktop/> (or `winget install
Docker.DockerDesktop`). During setup keep **"Use WSL 2 instead of Hyper-V"**
enabled. Launch Docker Desktop and wait until it reports "running".

**Step 3 — Install git and python:**

```powershell
winget install Git.Git
winget install Python.Python.3.12
```

**Step 4 — Choose your shell.** You can run everything from **PowerShell**, but
the guided `./deploy.sh` script is a Bash script. The smoothest path on Windows is
to run the deployment **inside WSL 2 (Ubuntu)**:

```powershell
wsl        # drops you into Ubuntu
```

Inside WSL, `git` and `docker` are already available (Docker Desktop integrates
with WSL). Verify:

```bash
docker --version
docker compose version
```

> **Tip:** Clone the repo **inside the WSL filesystem** (e.g. `~/sftp`), not under
> `/mnt/c/...`, for much better file-system performance.

---

## 2. Get the code

On any OS (in WSL on Windows):

```bash
git clone https://github.com/BrokingSapphire/sftp.git
cd sftp
```

---

## 3. Deploy the stack

You have three paths. The **one-command installer** is the fastest;
**Option A** is the guided deploy from a clone; **Option B** is fully manual.

### One-command installer (fastest)

This bootstraps everything — it installs missing prerequisites, clones the repo,
and launches the guided deploy. You don't even need to clone first.

```bash
# Linux / macOS / Windows (WSL)
curl -fsSL https://raw.githubusercontent.com/BrokingSapphire/sftp/main/install.sh | bash
```

```powershell
# Windows (PowerShell, native — no WSL required)
irm https://raw.githubusercontent.com/BrokingSapphire/sftp/main/install.ps1 | iex
```

Pass `--yes` (bash) to accept all defaults non-interactively. Prefer to review the
script before piping to a shell? Download it, read it, then run `bash install.sh`.

### Option A — Guided deploy (recommended)

`deploy.sh` asks a short series of questions (company name, brand colour, org
domains, first admin account, and whether to enable SMTP, SSO, encryption, and
AI), then generates `brand.config.json` and `.env` with strong random secrets and
starts everything.

**Linux / macOS / Windows-WSL:**

```bash
chmod +x deploy.sh
./deploy.sh
```

Answer the prompts. When it finishes it prints your URL and the admin credentials.
To accept all defaults non-interactively:

```bash
./deploy.sh --yes
```

> **Windows without WSL:** `deploy.sh` needs Bash. If you're not using WSL, follow
> **Option B** in PowerShell instead.

### Option B — Manual configuration

**Step 1 — Create the environment file:**

```bash
cp backend/.env.example .env
```

**Step 2 — Set the required secrets.** Edit `.env` and set at minimum:

| Variable | What to set it to |
| --- | --- |
| `JWT_SECRET` | A random string, **minimum 32 characters** |
| `POSTGRES_PASSWORD` | A strong database password |
| `BOOTSTRAP_ADMIN_EMAIL` | The first super-admin's email |
| `BOOTSTRAP_ADMIN_PASSWORD` | The first super-admin's password |
| `PUBLIC_URL` | The URL users will reach, e.g. `http://localhost` |
| `BACKUP_ENCRYPTION_KEY` | (Optional) 64 hex chars to enable encrypted backups |
| `STORAGE_ENCRYPTION_KEY` | (Optional) 64 hex chars to encrypt files at rest |

Generate strong values:

- **Linux / macOS / WSL:**
  ```bash
  openssl rand -base64 48   # JWT_SECRET
  openssl rand -hex 32      # BACKUP_ENCRYPTION_KEY / STORAGE_ENCRYPTION_KEY
  ```
- **Windows PowerShell:**
  ```powershell
  # 48-byte base64 secret
  [Convert]::ToBase64String((1..48 | ForEach-Object { Get-Random -Max 256 }))
  # 32-byte hex key
  -join ((1..32) | ForEach-Object { '{0:x2}' -f (Get-Random -Max 256) })
  ```

> **Important:** If you set `STORAGE_ENCRYPTION_KEY`, do so **before** the first
> upload. Turning it on later makes existing plaintext files unreadable. Store all
> keys somewhere safe — losing them means losing encrypted data.

**Step 3 — (Optional) White-label.** Copy and edit the branding file to change the
name, logo, colours, org domains, SMTP, and SSO:

```bash
cp brand.config.example.json brand.config.json   # if not already present
# edit brand.config.json
```

**Step 4 — Build and start:**

```bash
docker compose up -d --build
```

The first build takes a few minutes. The backend automatically runs database
migrations and creates the super-admin on first boot.

---

## 4. Verify the deployment

**Check every container is healthy:**

```bash
docker compose ps
```

All services should show `running` / `healthy`. Then check the API:

```bash
curl http://localhost/api/v1/health-check
```

You should get a JSON success response. Tail the backend logs if anything is off:

```bash
docker compose logs -f backend
```

Look for `migrations applied` and `starting HTTP server`.

---

## 5. First login

1. Open **http://localhost** (or your `PUBLIC_URL`) in a browser.
2. Sign in as the bootstrap admin (the email/password from `deploy.sh` or `.env`).
3. You'll be prompted to **change the password on first login** — do it.
4. Go to **Administration → Users** to create accounts and assign roles.
5. New users can pick their **language** from the top bar; it follows them across
   devices.

---

## 6. Optional features (AI, Office, monitoring)

Each optional stack is a Compose **profile**. Enable the ones you want by adding
`--profile <name>`:

```bash
# On-prem AI (semantic search + "Ask your files") via Ollama
docker compose --profile ai up -d

# Live Office co-editing via OnlyOffice
docker compose --profile office up -d

# Prometheus + Grafana monitoring
docker compose --profile monitoring up -d

# Combine as needed
docker compose --profile ai --profile office up -d
```

**AI models** must be pulled once after the Ollama container is up:

```bash
docker compose exec ollama ollama pull nomic-embed-text   # embeddings
docker compose exec ollama ollama pull llama3.2:1b         # chat (small/fast)
```

Grafana is then at **http://localhost:3001**, Prometheus at
**http://localhost:9090**.

---

## 7. Enable HTTPS (production)

Never expose plain HTTP to users in production. Terminate TLS at Nginx:

1. Obtain certificates (your internal CA, or Let's Encrypt) and place them in
   `docker/nginx/certs/` as `fullchain.pem` and `privkey.pem`.
2. In `docker/nginx/nginx.conf`, uncomment the `443` server block, the TLS
   directives, and the HTTP→HTTPS redirect.
3. In `docker-compose.yml`, publish `"443:443"` on the `nginx` service.
4. Set `PUBLIC_URL=https://files.yourcompany.com` in `.env`.
5. Apply:
   ```bash
   docker compose up -d
   ```

Alternatively, put the stack behind your existing load balancer / reverse proxy
that already handles TLS, and keep Nginx on port 80 internally.

---

## 8. Store files on a mounted drive / NAS

By default, files live in a Docker volume. To use a mounted disk or NAS instead,
bind-mount the **directory** in `docker-compose.yml` on the `backend` service:

```yaml
    volumes:
      - /mnt/nas/sftp:/app/storage/files    # Linux / WSL
      # - /Volumes/BigDisk/sftp:/app/storage/files   # macOS
```

Ensure the path exists and is writable by the container, then
`docker compose up -d`.

---

## 9. Native SFTP endpoint

The backend also speaks **SFTP-over-SSH** (default port `2222`) using the same
accounts and API keys as the web app. To enable it, publish the port on the
`backend` service in `docker-compose.yml`:

```yaml
    ports:
      - "2222:2222"
```

Then connect with any SFTP client:

```bash
sftp -P 2222 username@your-server
```

---

## 10. Day-2 operations

```bash
docker compose ps                      # service status
docker compose logs -f backend         # tail backend logs
docker compose stop                    # stop (keeps data + volumes)
docker compose start                   # start again
docker compose down                    # stop and remove containers (keeps volumes)
docker compose restart backend         # restart one service
```

**Backups** of user drives are super-admin only and encrypted — see
[BACKUP.md](./BACKUP.md). **Database** dumps:

```bash
docker compose exec postgres pg_dump -U sftp sftp > backup.sql
```

---

## 11. Updating to a new version

```bash
cd sftp
git pull
docker compose up -d --build          # rebuilds and restarts changed services
```

Migrations run automatically on backend start. Review release notes before
upgrading across major versions.

---

## 12. Uninstall

```bash
docker compose down                   # remove containers, keep data
# — or —
docker compose down -v                # remove containers AND all data volumes (DESTRUCTIVE)
```

`down -v` deletes the database and stored files permanently. Back up first.

---

## 13. Troubleshooting by OS

**All OSes**

- **A container keeps restarting:** `docker compose logs <service>` shows why.
  A common one is Postgres refusing to start after an unclean shutdown
  (`bogus data in lock file`) — recreate it: `docker compose rm -sf postgres &&
  docker compose up -d postgres`.
- **Port 80 already in use:** another web server is running. Stop it, or change the
  published port in `docker-compose.yml` (e.g. `"8080:80"`).
- **Out of disk space during build:** `docker system prune -af` frees unused
  images/build cache.

**Linux**

- **`permission denied` talking to Docker:** you're not in the `docker` group —
  re-run `sudo usermod -aG docker $USER` and start a new shell.
- **Firewall blocks access:** allow the port, e.g. `sudo ufw allow 80/tcp`.

**macOS**

- **Docker Desktop is slow or crashes:** raise RAM/disk in Settings → Resources;
  a full restart of Docker Desktop (or the Mac) often stabilises it.
- **Bind-mount "operation not permitted":** you're mounting from a restricted/
  external volume. Keep the repo on the internal disk; mount directories, not
  single files.

**Windows**

- **Docker won't start:** ensure **WSL 2** is installed (`wsl --install`) and that
  virtualization is enabled in BIOS; in Docker Desktop settings keep the WSL 2
  backend enabled.
- **Very slow file access / builds:** the repo is under `/mnt/c/...`. Move it into
  the WSL home directory (`~/sftp`) and rebuild.
- **`./deploy.sh` won't run in PowerShell:** it's a Bash script — run it inside WSL
  (`wsl` then `./deploy.sh`), or use **Option B** manual steps in PowerShell.

Still stuck? See [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) or open a
[discussion](https://github.com/BrokingSapphire/sftp/discussions).
