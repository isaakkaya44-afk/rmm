#!/bin/bash
set -e

BACKUP_DIR="./backups"
mkdir -p "$BACKUP_DIR"

DATE=$(date +%Y%m%d_%H%M%S)
FILE="$BACKUP_DIR/rmm_$DATE.sql"

echo "Backing up PostgreSQL..."
docker exec rmm-postgres pg_dump -U rmm rmm_platform > "$FILE"
gzip "$FILE"

echo "Backup: ${FILE}.gz ($(du -h "${FILE}.gz" | cut -f1))"

# Keep last 7 days, delete older
find "$BACKUP_DIR" -name "rmm_*.sql.gz" -mtime +7 -delete
