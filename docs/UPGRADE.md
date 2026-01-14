# Upgrade Guide

This guide describes how to safely upgrade Status Incident Service without losing data.

## Version History

| Version | Schema | Changes |
|---------|--------|---------|
| v1.0.1  | 1      | Fix auto-refresh interrupting form editing |
| v1.0.0  | 1      | Initial stable release |

## Upgrading Between Versions

### v1.0.0 → v1.0.1

**No database changes.** Safe to upgrade without any special steps.

```bash
# Docker
docker pull ghcr.io/pomaxa/status-service:v1.0.1
docker-compose down && docker-compose up -d

# Binary
curl -LO https://github.com/pomaxa/status-service/releases/download/v1.0.1/status-incident-linux-amd64
```

Changes:
- Fixed: Auto-refresh no longer interrupts form editing
- Fixed: Modal dialogs don't get closed by auto-refresh

## How Updates Work

### Automatic Backup
When the application starts and detects pending database migrations, it automatically creates a backup:
```
status.db.backup-20240115-103045
```

The backup is created BEFORE applying any migrations, ensuring you can always restore to the previous state.

### Migration Versioning
Each migration is tracked in the `schema_migrations` table:
```sql
SELECT * FROM schema_migrations;
-- version | name           | applied_at
-- 1       | initial_schema | 2024-01-15 10:30:45
```

## Upgrade Process

### Docker (Recommended)

```bash
# 1. Pull the new image
docker pull ghcr.io/pomaxa/status-service:latest

# 2. Stop the current container
docker-compose down
# or: docker stop status-incident

# 3. Start with new image
docker-compose up -d
# or: docker run -p 8080:8080 -v status-data:/app/data ghcr.io/pomaxa/status-service:latest

# 4. Check logs for migration status
docker logs status-incident
```

### Binary

```bash
# 1. Stop the service
systemctl stop status-incident
# or: pkill status-incident

# 2. Download new binary
curl -LO https://github.com/pomaxa/status-service/releases/latest/download/status-incident-linux-amd64
chmod +x status-incident-linux-amd64

# 3. Replace binary
mv status-incident-linux-amd64 /usr/local/bin/status-incident

# 4. Start service
systemctl start status-incident
# or: ./status-incident
```

## Manual Backup

If you want to create a manual backup before upgrading:

```bash
# SQLite backup (simple copy)
cp status.db status.db.manual-backup

# Or using SQLite CLI
sqlite3 status.db ".backup 'status.db.manual-backup'"
```

## Restore from Backup

If something goes wrong after an upgrade:

### From Automatic Backup
```bash
# 1. Stop the service
systemctl stop status-incident

# 2. Find the latest backup
ls -la status.db.backup-*

# 3. Restore
cp status.db.backup-20240115-103045 status.db

# 4. (Optional) Downgrade binary if needed
# Download previous version from GitHub releases

# 5. Start service
systemctl start status-incident
```

### Using Export/Import
```bash
# Export before upgrade (via API)
curl http://localhost:8080/api/export > backup.json

# Import after fresh install
curl -X POST -H "Content-Type: application/json" \
  -d @backup.json http://localhost:8080/api/import
```

## Version Check

Check current version:
```bash
./status-incident -version
# Status Incident Service
#   Version:    v1.2.0
#   Commit:     abc123
#   Build time: 2024-01-15_10:30:00
```

Check schema version:
```bash
sqlite3 status.db "SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 1;"
```

## Troubleshooting

### Migration Failed
If migration fails, the application will not start. Check logs for error details.

1. Restore from automatic backup (created before migration attempt)
2. Report the issue on GitHub
3. Wait for fix or apply manual SQL fix

### Database Locked
If you see "database is locked" errors:
```bash
# Check for running processes
lsof status.db

# Force WAL checkpoint
sqlite3 status.db "PRAGMA wal_checkpoint(TRUNCATE);"
```

### Incompatible Version
If downgrading to an older version that doesn't support the current schema:
1. Export data via API: `GET /api/export`
2. Delete database: `rm status.db`
3. Start old version (creates fresh DB)
4. Import data: `POST /api/import`

Note: Some data may be lost if new fields were added in newer versions.

## Docker Compose Example

```yaml
version: '3.8'

services:
  status-incident:
    image: ghcr.io/pomaxa/status-service:latest
    container_name: status-incident
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    environment:
      - TZ=UTC
    restart: unless-stopped
```

Data is persisted in `./data/status.db`. Backups will be created in the same directory.
