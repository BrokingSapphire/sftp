# Backup & Restore

Two things must be backed up together, ideally at the same time:

1. **PostgreSQL** — all metadata (users, folders, files, shares, audit).
2. **File storage** — the actual file content (the `sftp-files` volume / mount).

A database backup without the matching files (or vice-versa) is inconsistent.

## Backup

```bash
# 1. Database (compressed custom-format dump)
docker compose exec -T postgres pg_dump -U sftp -Fc sftp > backup/sftp-$(date +%F).dump

# 2. File storage volume
docker run --rm \
  -v sftp_sftp-files:/data -v "$PWD/backup":/backup alpine \
  tar czf /backup/sftp-files-$(date +%F).tar.gz -C /data .
```

Automate with cron; keep dumps and archives from the same run paired. Store
off-box (another server / NAS). Test restores periodically.

## Restore

```bash
# 1. Restore the database
docker compose up -d postgres
docker compose exec -T postgres pg_restore -U sftp -d sftp --clean --if-exists \
  < backup/sftp-2026-07-05.dump

# 2. Restore file storage
docker run --rm \
  -v sftp_sftp-files:/data -v "$PWD/backup":/backup alpine \
  sh -c "rm -rf /data/* && tar xzf /backup/sftp-files-2026-07-05.tar.gz -C /data"

# 3. Start the stack
docker compose up -d
```

## Notes

- Migrations run automatically on backend start; a restored DB at an older schema
  version is migrated forward safely.
- The `checksum_sha256` stored per file lets you verify content integrity after a
  restore.
