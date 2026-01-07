#!/bin/bash
# Status Incident - Backup Script
# Run via cron: 0 2 * * * /opt/status-incident/backup.sh

set -e

# Configuration
BACKUP_DIR="/opt/status-incident/backups"
DB_PATH="/opt/status-incident/data/status.db"
API_URL="http://127.0.0.1:8080"
RETENTION_DAYS=30

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Timestamp
DATE=$(date +%Y%m%d_%H%M%S)

# Method 1: API Export (recommended - no service interruption)
echo "Exporting data via API..."
if curl -sf "$API_URL/api/export" > "$BACKUP_DIR/export_$DATE.json"; then
    echo "API export successful: export_$DATE.json"
else
    echo "Warning: API export failed, falling back to file copy"
fi

# Method 2: SQLite backup (requires brief lock)
echo "Creating SQLite backup..."
if [ -f "$DB_PATH" ]; then
    sqlite3 "$DB_PATH" ".backup '$BACKUP_DIR/status_$DATE.db'"
    echo "Database backup successful: status_$DATE.db"
else
    echo "Warning: Database file not found at $DB_PATH"
fi

# Compress old backups
echo "Compressing backups older than 1 day..."
find "$BACKUP_DIR" -name "*.json" -mtime +1 -exec gzip {} \;
find "$BACKUP_DIR" -name "*.db" -mtime +1 -exec gzip {} \;

# Delete old backups
echo "Removing backups older than $RETENTION_DAYS days..."
find "$BACKUP_DIR" -name "*.gz" -mtime +$RETENTION_DAYS -delete
find "$BACKUP_DIR" -name "*.json" -mtime +$RETENTION_DAYS -delete
find "$BACKUP_DIR" -name "*.db" -mtime +$RETENTION_DAYS -delete

# Summary
echo "Backup complete. Current backups:"
ls -lh "$BACKUP_DIR" | tail -10

echo "Done!"
