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

# Brand logo: accept a local image path or an https URL. Blank keeps the bundled
# default Sapphire logo. A local file is copied into the frontend's public dir;
# a URL is referenced as-is. We patch brand.config.json's logo paths to match.
$logoSrc = if ($env:LOGO_PATH) { $env:LOGO_PATH } else { Read-Host "Brand logo (image path or https URL, blank = default Sapphire logo)" }
if ($logoSrc) {
  if ($logoSrc -match '^https?://') {
    $logoPath = $logoSrc
    Ok "Using logo URL: $logoSrc"
  } elseif (Test-Path $logoSrc) {
    $ext = [System.IO.Path]::GetExtension($logoSrc); if (-not $ext) { $ext = ".svg" }
    $dest = "logo-custom$ext"
    New-Item -ItemType Directory -Force -Path ".\frontend\public" | Out-Null
    Copy-Item -Force $logoSrc ".\frontend\public\$dest"
    $logoPath = "/$dest"
    Ok "Custom logo copied to frontend/public/$dest"
  } else {
    Warn "Logo not found: $logoSrc - keeping the default Sapphire logo."
    $logoPath = $null
  }
  if ($logoPath -and (Test-Path ".\brand.config.json")) {
    $cfg = Get-Content ".\brand.config.json" -Raw | ConvertFrom-Json
    $cfg.logo.full = $logoPath; $cfg.logo.light = $logoPath
    $cfg.logo.dark = $logoPath; $cfg.logo.favicon = $logoPath
    ($cfg | ConvertTo-Json -Depth 20) | Set-Content -Path ".\brand.config.json" -Encoding ascii
    Ok "brand.config.json logo paths updated"
  }
}

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

# --- 6. autostart on boot ----------------------------------------------------
# The compose files use `restart: unless-stopped`, so containers revive once the
# Docker engine is up. On Windows the engine only starts after Docker Desktop
# launches, so we (a) enable Docker Desktop's "start on login" and (b) register a
# startup Scheduled Task that runs `docker compose up -d` to cover cold boots and
# prior `docker compose down`.
Info "Configuring autostart on boot"
try {
  $settings = Join-Path $env:APPDATA "Docker\settings-store.json"
  if (-not (Test-Path $settings)) { $settings = Join-Path $env:APPDATA "Docker\settings.json" }
  if (Test-Path $settings) {
    $s = Get-Content $settings -Raw | ConvertFrom-Json
    $s | Add-Member -NotePropertyName autoStart -NotePropertyValue $true -Force
    ($s | ConvertTo-Json -Depth 20) | Set-Content -Path $settings -Encoding ascii
    Ok "Docker Desktop set to start on login"
  } else {
    Warn "Could not find Docker Desktop settings - enable 'Start Docker Desktop when you log in' in its settings."
  }
} catch { Warn "Could not update Docker Desktop autostart setting - enable it manually in Docker Desktop settings." }

try {
  $dcExe = (Get-Command docker).Source
  $action  = New-ScheduledTaskAction -Execute $dcExe -Argument "compose up -d" -WorkingDirectory $dir
  $trigger = New-ScheduledTaskTrigger -AtStartup
  $principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest
  $taskSettings = New-ScheduledTaskSettingsSet -StartWhenAvailable -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1)
  Register-ScheduledTask -TaskName "SapphireSFTP" -Action $action -Trigger $trigger -Principal $principal -Settings $taskSettings -Force | Out-Null
  Ok "Startup task 'SapphireSFTP' registered - the stack will come up on every boot"
} catch {
  Warn "Could not register the startup task (needs an elevated PowerShell). The stack still"
  Warn "auto-restarts while Docker Desktop is running. To enable full boot autostart, re-run"
  Warn "this installer in an Administrator PowerShell."
}

Write-Host ""
Ok "Sapphire SFTP is starting."
Write-Host "  URL:      http://localhost"
Write-Host "  Admin:    $adminEmail"
Write-Host "  Password: $adminPass"
Write-Host ""
Write-Host "Check status with:  docker compose ps"
