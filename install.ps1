# install.ps1 - one-command bootstrap installer for Sapphire SFTP on Windows.
#
# Ensures prerequisites (git, Docker Desktop), clones the repo, generates a .env
# with strong random secrets, and starts the full stack with Docker Compose.
#
#   Run in PowerShell:
#     irm https://raw.githubusercontent.com/BrokingSapphire/sftp/main/install.ps1 | iex
#
#   Or locally:
#     powershell -ExecutionPolicy Bypass -File install.ps1
#
# Optional environment variables:
#   $env:SFTP_DIR   = "C:\sftp"        # clone location (default: .\sftp)
#   $env:ADMIN_EMAIL = "admin@corp.com"
#
$ErrorActionPreference = "Stop"

function Info($m) { Write-Host "==> $m" -ForegroundColor Cyan }
function Ok($m)   { Write-Host "OK  $m" -ForegroundColor Green }
function Warn($m) { Write-Host "!   $m" -ForegroundColor Yellow }
function Die($m)  { Write-Host "X   $m" -ForegroundColor Red; exit 1 }
function Has($c)  { return [bool](Get-Command $c -ErrorAction SilentlyContinue) }

Info "Sapphire SFTP installer (Windows)"

# --- 1. git ------------------------------------------------------------------
if (-not (Has git)) {
  Info "Installing Git"
  if (Has winget) { winget install --id Git.Git -e --source winget --accept-package-agreements --accept-source-agreements }
  else { Die "git not found and winget unavailable. Install Git from https://git-scm.com/ and re-run." }
}
Ok "git available"

# --- 2. Docker ---------------------------------------------------------------
if (-not (Has docker)) {
  Warn "Docker Desktop is not installed."
  if (Has winget) {
    Info "Installing Docker Desktop (winget)"
    winget install --id Docker.DockerDesktop -e --accept-package-agreements --accept-source-agreements
  }
  Die "Start Docker Desktop (with the WSL 2 backend), wait until it reports 'running', then re-run this installer."
}
try { docker info | Out-Null } catch { Die "Docker is installed but not running. Start Docker Desktop and re-run." }
docker compose version | Out-Null
Ok "Docker + Compose available"

# --- 3. get the code ---------------------------------------------------------
if ((Test-Path ".\deploy.sh") -and (Test-Path ".\docker-compose.yml")) {
  $dir = (Get-Location).Path
  Ok "Running inside an existing checkout: $dir"
} else {
  $dir = if ($env:SFTP_DIR) { $env:SFTP_DIR } else { Join-Path (Get-Location).Path "sftp" }
  if (Test-Path (Join-Path $dir ".git")) {
    Info "Updating existing checkout at $dir"; git -C $dir pull --ff-only
  } else {
    Info "Cloning into $dir"; git clone --depth 1 https://github.com/BrokingSapphire/sftp.git $dir
  }
}
Set-Location $dir

# --- 4. prefer WSL for the full guided deploy --------------------------------
if (Has wsl) {
  Info "WSL detected - running the guided deploy inside WSL"
  wsl bash ./install.sh $args
  exit $LASTEXITCODE
}

# --- 5. native PowerShell deploy (no WSL) ------------------------------------
Info "Generating .env with strong secrets"
function RandHex($n) { -join ((1..$n) | ForEach-Object { '{0:x2}' -f (Get-Random -Max 256) }) }
$adminEmail = if ($env:ADMIN_EMAIL) { $env:ADMIN_EMAIL } else { "admin@example.com" }
$adminPass  = RandHex 12
if (-not (Test-Path ".env")) {
@"
JWT_SECRET=$(RandHex 48)
POSTGRES_PASSWORD=$(RandHex 16)
BOOTSTRAP_ADMIN_EMAIL=$adminEmail
BOOTSTRAP_ADMIN_USERNAME=admin
BOOTSTRAP_ADMIN_PASSWORD=$adminPass
PUBLIC_URL=http://localhost
BACKUP_ENCRYPTION_KEY=$(RandHex 32)
"@ | Set-Content -Path ".env" -Encoding ascii
  Ok ".env created"
} else {
  Warn ".env already exists - leaving it unchanged"
  $adminPass = "(unchanged - see your existing .env)"
}

Info "Building and starting the stack (this takes a few minutes)"
docker compose up -d --build

Write-Host ""
Ok "Sapphire SFTP is starting."
Write-Host "  URL:      http://localhost"
Write-Host "  Admin:    $adminEmail"
Write-Host "  Password: $adminPass"
Write-Host ""
Write-Host "Check status with:  docker compose ps"
